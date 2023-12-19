package aaq_evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	quota "k8s.io/apiserver/pkg/quota/v1"
	"kubevirt.io/applications-aware-quota/pkg/log"
	"kubevirt.io/applications-aware-quota/pkg/util"
	"kubevirt.io/applications-aware-quota/pkg/util/net/grpc"
	sidecar_evaluator "kubevirt.io/applications-aware-quota/staging/evaluator-server-com"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const dialSockErr = "Failed to Dial socket: %s"

var aaqEvaluatorsRegistry *AaqEvaluatorsRegistry
var once sync.Once

func newAaqEvaluatorsRegistry(retriesOnMatchFailure int, socketSharedDirectory string) *AaqEvaluatorsRegistry {
	return &AaqEvaluatorsRegistry{
		retriesOnMatchFailure: retriesOnMatchFailure,
		socketSharedDirectory: socketSharedDirectory,
	}
}

type (
	Registry interface {
		Collect(numberOfRequestedEvaluatorsSidecars uint, timeout time.Duration) error
		Usage(corev1.Pod, []corev1.Pod) (corev1.ResourceList, error)
	}
	AaqEvaluatorsRegistry struct {
		AaqSocketCalculator   []AaqSocketCalculator
		socketSharedDirectory string
		// used to track time
		retriesOnMatchFailure int
	}
)

func GetAaqEvaluatorsRegistry() *AaqEvaluatorsRegistry {
	once.Do(func() {
		aaqEvaluatorsRegistry = newAaqEvaluatorsRegistry(10, util.SocketsSharedDirectory)
	})
	return aaqEvaluatorsRegistry
}

func (aaqe *AaqEvaluatorsRegistry) Collect(numberOfRequestedEvaluatorsSidecars uint, timeout time.Duration) error {
	socketsPaths, err := aaqe.collectSidecarSockets(numberOfRequestedEvaluatorsSidecars, timeout)
	if err != nil {
		return err
	}
	for _, socketPath := range socketsPaths {
		aaqe.AaqSocketCalculator = append(aaqe.AaqSocketCalculator, &AaqSocketEvaluators{socketPath})
	}
	log.Log.Info("Collected all requested evaluators sidecars sockets")
	return nil
}

func (aaqe *AaqEvaluatorsRegistry) collectSidecarSockets(numberOfRequestedEvaluatorsSidecars uint, timeout time.Duration) ([]string, error) {
	var sidecarSockets []string
	processedSockets := make(map[string]bool)

	timeoutCh := time.After(timeout)

	for uint(len(processedSockets)) < numberOfRequestedEvaluatorsSidecars {
		sockets, err := os.ReadDir(aaqe.socketSharedDirectory)
		if err != nil {
			return nil, err
		}
		for _, socket := range sockets {
			select {
			case <-timeoutCh:
				return nil, fmt.Errorf("Failed to collect all expected evaluators sidecars sockets within given timeout")
			default:
				if _, processed := processedSockets[socket.Name()]; processed {
					continue
				}

				callBackClient, notReady, err := processSideCarSocket(filepath.Join(aaqe.socketSharedDirectory, socket.Name()))
				if notReady {
					log.Log.Info("Sidecar server might not be ready yet, retrying in the next iteration")
					continue
				} else if err != nil {
					log.Log.Reason(err).Infof("Failed to process sidecar socket: %s", socket.Name())
					return nil, err
				}
				sidecarSockets = append(sidecarSockets, callBackClient)
				processedSockets[socket.Name()] = true
			}
		}
		time.Sleep(time.Second)
	}
	return sidecarSockets, nil
}

func processSideCarSocket(socketPath string) (string, bool, error) {
	conn, err := grpc.DialSocketWithTimeout(socketPath, 1)
	if err != nil {
		log.Log.Reason(err).Infof(dialSockErr, socketPath)
		return "", true, nil
	}
	defer conn.Close()

	infoClient := sidecar_evaluator.NewPodUsageClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	health, err := infoClient.HealthCheck(ctx, &sidecar_evaluator.HealthCheckRequest{})
	if err != nil || health == nil || !health.Healthy {
		return "", false, err
	}
	return socketPath, false, nil
}

func (aaqe *AaqEvaluatorsRegistry) Usage(pod corev1.Pod, podsState []corev1.Pod) (rlToRet corev1.ResourceList, acceptedErr error) {
	accepted := false
	for _, aaqSocketEvaluator := range aaqe.AaqSocketCalculator {
		for retries := 0; retries < aaqe.retriesOnMatchFailure; retries++ {
			rl, err, match := aaqSocketEvaluator.podUsageFunc(pod, podsState)
			if !match && err == nil {
				break
			} else if err == nil {
				accepted = true
				rlToRet = quota.Add(rlToRet, rl)
				break
			} else {
				log.Log.Infof(fmt.Sprintf("Retries: %v Error: %v ", retries, err))
			}
		}
	}
	if !accepted {
		acceptedErr = fmt.Errorf("pod didn't match any usageFunc")
	}
	return rlToRet, acceptedErr
}

type (
	AaqSocketCalculator interface {
		podUsageFunc(pod corev1.Pod, podsState []corev1.Pod) (corev1.ResourceList, error, bool)
	}
	AaqSocketEvaluators struct {
		sidecarSocketPath string
	}
)

func (aaqse *AaqSocketEvaluators) podUsageFunc(pod corev1.Pod, podsState []corev1.Pod) (corev1.ResourceList, error, bool) {
	conn, err := grpc.DialSocketWithTimeout(aaqse.sidecarSocketPath, 1)
	if err != nil {
		log.Log.Reason(err).Errorf(dialSockErr, aaqse.sidecarSocketPath)
		return nil, err, false
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client := sidecar_evaluator.NewPodUsageClient(conn)
	podData, err := json.Marshal(pod)
	if err != nil {
		return nil, err, false
	}
	var podsStateData []*sidecar_evaluator.Pod
	for _, p := range podsState {
		pData, err := json.Marshal(p)
		if err != nil {
			return nil, err, false
		}
		podsStateData = append(podsStateData, &sidecar_evaluator.Pod{pData})
	}
	result, err := client.PodUsageFunc(ctx, &sidecar_evaluator.PodUsageRequest{
		Pod:       &sidecar_evaluator.Pod{podData},
		PodsState: podsStateData,
	})
	if err != nil {
		log.Log.Reason(err).Error(fmt.Sprintf("Failed to call PodUsageFunc with pod %v", pod))
		return nil, err, false
	}
	rl := corev1.ResourceList{}
	if err := json.Unmarshal(result.ResourceList.ResourceListJson, &rl); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal given rl : %s due %v", result.ResourceList.ResourceListJson, err), false
	}
	var resErr error
	if result.Error.Error {
		resErr = fmt.Errorf(result.Error.ErrorMessage)
	}
	return rl, resErr, result.Match
}
