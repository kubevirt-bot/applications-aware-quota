package arq_controller

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sadmission "k8s.io/apiserver/pkg/admission"
	resourcequota2 "k8s.io/apiserver/pkg/admission/plugin/resourcequota"
	"k8s.io/apiserver/pkg/admission/plugin/resourcequota/apis/resourcequota"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	_ "kubevirt.io/api/core/v1"
	aaq_evaluator "kubevirt.io/applications-aware-quota/pkg/aaq-controller/aaq-evaluator"
	"kubevirt.io/applications-aware-quota/pkg/client"
	"kubevirt.io/applications-aware-quota/pkg/log"
	"kubevirt.io/applications-aware-quota/pkg/util"
	v1alpha12 "kubevirt.io/applications-aware-quota/staging/src/kubevirt.io/applications-aware-quota-api/pkg/apis/core/v1alpha1"
	"time"
)

type enqueueState string

const (
	Immediate  enqueueState = "Immediate"
	Forget     enqueueState = "Forget"
	BackOff    enqueueState = "BackOff"
	AaqjqcName string       = "aaqjqc"
)

type AaqGateController struct {
	podInformer    cache.SharedIndexInformer
	arqInformer    cache.SharedIndexInformer
	aaqjqcInformer cache.SharedIndexInformer
	arqQueue       workqueue.RateLimitingInterface
	aaqCli         client.AAQClient
	aaqEvaluator   *aaq_evaluator.AaqEvaluator
	stop           <-chan struct{}
}

func NewAaqGateController(aaqCli client.AAQClient,
	podInformer cache.SharedIndexInformer,
	arqInformer cache.SharedIndexInformer,
	aaqjqcInformer cache.SharedIndexInformer,
	calcRegistry *aaq_evaluator.AaqEvaluatorsRegistry,
	stop <-chan struct{},
) *AaqGateController {

	ctrl := AaqGateController{
		aaqCli:         aaqCli,
		aaqjqcInformer: aaqjqcInformer,
		podInformer:    podInformer,
		arqInformer:    arqInformer,
		arqQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "arq-queue"),
		aaqEvaluator:   aaq_evaluator.NewAaqEvaluator(podInformer, calcRegistry, clock.RealClock{}),
		stop:           stop,
	}

	_, err := ctrl.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.addPod,
		UpdateFunc: ctrl.updatePod,
	})
	if err != nil {
		panic("something is wrong")

	}
	_, err = ctrl.arqInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: ctrl.deleteArq,
		UpdateFunc: ctrl.updateArq,
		AddFunc:    ctrl.addArq,
	})
	if err != nil {
		panic("something is wrong")
	}
	_, err = ctrl.aaqjqcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: ctrl.deleteAaqjqc,
		UpdateFunc: ctrl.updateAaqjqc,
	})
	if err != nil {
		panic("something is wrong")
	}

	return &ctrl
}

func (ctrl *AaqGateController) addAllArqsInNamespace(ns string) {
	objs, err := ctrl.arqInformer.GetIndexer().ByIndex(cache.NamespaceIndex, ns)
	atLeastOneArq := false
	if err != nil {
		log.Log.Infof("AaqGateController: Error failed to list pod from podInformer")
	}
	for _, obj := range objs {
		arq := obj.(*v1alpha12.ApplicationsResourceQuota)
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(arq)
		if err != nil {
			return
		}
		atLeastOneArq = true
		ctrl.arqQueue.Add(key)
	}
	if !atLeastOneArq {
		ctrl.arqQueue.Add(ns + "/fake")
	}
}

// enqueueAll is called at the fullResyncPeriod interval to force a full recalculation of quota usage statistics
func (ctrl *AaqGateController) enqueueAll() {
	arqObjs := ctrl.arqInformer.GetIndexer().List()
	for _, arqObj := range arqObjs {
		arq := arqObj.(*v1alpha12.ApplicationsResourceQuota)
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(arqObj)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("couldn't get key for object %+v: %v", arq, err))
			continue
		}
		ctrl.arqQueue.Add(key)
	}
}

// When a ApplicationsResourceQuotas is deleted, enqueue all gated pods for revaluation
func (ctrl *AaqGateController) deleteArq(obj interface{}) {
	arq := obj.(*v1alpha12.ApplicationsResourceQuota)
	ctrl.addAllArqsInNamespace(arq.Namespace)
	return
}

// When a ApplicationsResourceQuotas is updated, enqueue all gated pods for revaluation
func (ctrl *AaqGateController) addArq(obj interface{}) {
	arq := obj.(*v1alpha12.ApplicationsResourceQuota)
	ctrl.addAllArqsInNamespace(arq.Namespace)
	return
}

// When a ApplicationsResourceQuotas is updated, enqueue all gated pods for revaluation
func (ctrl *AaqGateController) updateArq(old, cur interface{}) {
	arq := cur.(*v1alpha12.ApplicationsResourceQuota)
	ctrl.addAllArqsInNamespace(arq.Namespace)
	return
}

// When a ApplicationsResourceQuotaaqjqc.Status.PodsInJobQueuea is updated, enqueue all gated pods for revaluation
func (ctrl *AaqGateController) updateAaqjqc(old, cur interface{}) {
	aaqjqc := cur.(*v1alpha12.AAQJobQueueConfig)
	if aaqjqc.Status.PodsInJobQueue == nil || len(aaqjqc.Status.PodsInJobQueue) == 0 {
		ctrl.addAllArqsInNamespace(aaqjqc.Namespace)

	}
	return
}

// When a ApplicationsResourceQuotas is updated, enqueue all gated pods for revaluation
func (ctrl *AaqGateController) deleteAaqjqc(obj interface{}) {
	arq := obj.(*v1alpha12.AAQJobQueueConfig)
	ctrl.addAllArqsInNamespace(arq.Namespace)
	return
}

func (ctrl *AaqGateController) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)

	if pod.Spec.SchedulingGates != nil &&
		len(pod.Spec.SchedulingGates) == 1 &&
		pod.Spec.SchedulingGates[0].Name == util.AAQGate {
		ctrl.addAllArqsInNamespace(pod.Namespace)
	}
}
func (ctrl *AaqGateController) updatePod(old, curr interface{}) {
	pod := curr.(*v1.Pod)

	if pod.Spec.SchedulingGates != nil &&
		len(pod.Spec.SchedulingGates) == 1 &&
		pod.Spec.SchedulingGates[0].Name == util.AAQGate {
		ctrl.addAllArqsInNamespace(pod.Namespace)
	}
}

func (ctrl *AaqGateController) runWorker() {
	for ctrl.Execute() {
	}
}

func (ctrl *AaqGateController) Execute() bool {
	key, quit := ctrl.arqQueue.Get()
	if quit {
		return false
	}
	defer ctrl.arqQueue.Done(key)

	err, enqueueState := ctrl.execute(key.(string))
	if err != nil {
		klog.Errorf(fmt.Sprintf("AaqGateController: Error with key: %v err: %v", key, err))
	}
	switch enqueueState {
	case BackOff:
		ctrl.arqQueue.AddRateLimited(key)
	case Forget:
		ctrl.arqQueue.Forget(key)
	case Immediate:
		ctrl.arqQueue.Add(key)
	}

	return true
}

func (ctrl *AaqGateController) execute(key string) (error, enqueueState) {
	var aaqjqc *v1alpha12.AAQJobQueueConfig
	arqNS, _, err := cache.SplitMetaNamespaceKey(key)
	aaqjqcObj, exists, err := ctrl.aaqjqcInformer.GetIndexer().GetByKey(arqNS + "/" + AaqjqcName)
	if err != nil {
		return err, Immediate
	} else if !exists {
		aaqjqc, err = ctrl.aaqCli.AAQJobQueueConfigs(arqNS).Create(context.Background(),
			&v1alpha12.AAQJobQueueConfig{ObjectMeta: metav1.ObjectMeta{Name: AaqjqcName}, Spec: v1alpha12.AAQJobQueueConfigSpec{}},
			metav1.CreateOptions{})
		if err != nil {
			return err, Immediate
		}
	} else {
		aaqjqc = aaqjqcObj.(*v1alpha12.AAQJobQueueConfig).DeepCopy()
	}
	if len(aaqjqc.Status.PodsInJobQueue) != 0 {
		err := ctrl.releasePods(aaqjqc.Status.PodsInJobQueue, arqNS)
		return err, Immediate //wait for the calculator to process changes
	}

	arqsObjs, err := ctrl.arqInformer.GetIndexer().ByIndex(cache.NamespaceIndex, arqNS)
	if err != nil {
		return err, Immediate
	}
	var rqs []v1.ResourceQuota
	for _, arqObj := range arqsObjs {
		arq := arqObj.(*v1alpha12.ApplicationsResourceQuota)
		rq := v1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{Name: arq.Name, Namespace: arqNS},
			Spec:   v1.ResourceQuotaSpec{Hard: arq.Spec.Hard},
			Status: v1.ResourceQuotaStatus{Hard: arq.Status.Hard, Used: arq.Status.Used},
		}
		rqs = append(rqs, rq)
	}

	podsList, err := ctrl.aaqCli.CoreV1().Pods(arqNS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err, Immediate
	}
	for _, pod := range podsList.Items {
		if pod.Spec.SchedulingGates != nil &&
			len(pod.Spec.SchedulingGates) == 1 &&
			pod.Spec.SchedulingGates[0].Name == "ApplicationsAwareQuotaGate" {
			podCopy := pod.DeepCopy()
			podCopy.Spec.SchedulingGates = []v1.PodSchedulingGate{}

			podToCreateAttr := k8sadmission.NewAttributesRecord(podCopy, nil,
				apiextensions.Kind("Pod").WithVersion("version"), podCopy.Namespace, podCopy.Name,
				v1alpha12.Resource("pods").WithVersion("version"), "", k8sadmission.Create,
				&metav1.CreateOptions{}, false, nil)

			currPodLimitedResource, err := getCurrLimitedResource(ctrl.aaqEvaluator, podCopy)
			if err != nil {
				return nil, Immediate
			}

			newRq, err := resourcequota2.CheckRequest(rqs, podToCreateAttr, ctrl.aaqEvaluator, []resourcequota.LimitedResource{currPodLimitedResource})
			if err == nil {
				rqs = newRq
				aaqjqc.Status.PodsInJobQueue = append(aaqjqc.Status.PodsInJobQueue, pod.Name)
			} //todo: create an event if we are blocked for a while
		}
	}

	if len(aaqjqc.Status.PodsInJobQueue) > 0 {
		aaqjqc, err = ctrl.aaqCli.AAQJobQueueConfigs(arqNS).UpdateStatus(context.Background(), aaqjqc, metav1.UpdateOptions{})
		if err != nil {
			return err, Immediate
		}
	}
	err = ctrl.releasePods(aaqjqc.Status.PodsInJobQueue, arqNS)
	if err != nil {
		return err, Immediate
	}
	return nil, Forget
}

func (ctrl *AaqGateController) releasePods(podsToRelease []string, ns string) error {
	for _, podName := range podsToRelease {
		obj, exists, err := ctrl.podInformer.GetIndexer().GetByKey(ns + "/" + podName)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		pod := obj.(*v1.Pod).DeepCopy()
		if pod.Spec.SchedulingGates != nil && len(pod.Spec.SchedulingGates) == 1 && pod.Spec.SchedulingGates[0].Name == util.AAQGate {
			pod.Spec.SchedulingGates = []v1.PodSchedulingGate{}
			_, err = ctrl.aaqCli.CoreV1().Pods(ns).Update(context.Background(), pod, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil

}

func (ctrl *AaqGateController) Run(ctx context.Context, threadiness int) {
	defer utilruntime.HandleCrash()
	klog.Info("Starting Arq controller")
	defer klog.Info("Shutting down Arq controller")
	defer ctrl.arqQueue.ShutDown()

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.runWorker, time.Second, ctrl.stop)
	}

	<-ctrl.stop

}

func getCurrLimitedResource(podEvaluator *aaq_evaluator.AaqEvaluator, podToCreate *v1.Pod) (resourcequota.LimitedResource, error) {
	launcherLimitedResource := resourcequota.LimitedResource{
		Resource:      "pods",
		MatchContains: []string{},
	}
	usage, err := podEvaluator.Usage(podToCreate)
	if err != nil {
		return launcherLimitedResource, err
	}
	for k, _ := range usage {
		launcherLimitedResource.MatchContains = append(launcherLimitedResource.MatchContains, string(k))
	}
	return launcherLimitedResource, nil
}
