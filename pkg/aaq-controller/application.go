/*
 * This file is part of the AAQ project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023,Red Hat, Inc.
 *
 */
package aaq_controller

import (
	"context"
	goflag "flag"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	v14 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"kubevirt.io/applications-aware-quota/pkg/aaq-controller/aaq-evaluator"
	arq_controller2 "kubevirt.io/applications-aware-quota/pkg/aaq-controller/aaq-gate-controller"
	"kubevirt.io/applications-aware-quota/pkg/aaq-controller/arq-controller"
	"kubevirt.io/applications-aware-quota/pkg/aaq-controller/leaderelectionconfig"
	rq_controller "kubevirt.io/applications-aware-quota/pkg/aaq-controller/rq-controller"
	"kubevirt.io/applications-aware-quota/pkg/certificates/bootstrap"
	"kubevirt.io/applications-aware-quota/pkg/client"
	"kubevirt.io/applications-aware-quota/pkg/informers"
	"kubevirt.io/applications-aware-quota/pkg/util"
	golog "log"
	"net/http"
	"os"
	"strconv"
)

type AaqControllerApp struct {
	ctx               context.Context
	aaqNs             string
	host              string
	LeaderElection    leaderelectionconfig.Configuration
	aaqCli            client.AAQClient
	arqController     *arq_controller.ArqController
	aaqGateController *arq_controller2.AaqGateController
	rqController      *rq_controller.RQController
	podInformer       cache.SharedIndexInformer
	arqInformer       cache.SharedIndexInformer
	aaqInformer       cache.SharedIndexInformer
	rqInformer        cache.SharedIndexInformer
	aaqjqcInformer    cache.SharedIndexInformer
	calcRegistry      *aaq_evaluator.AaqEvaluatorsRegistry
	readyChan         chan bool
	leaderElector     *leaderelection.LeaderElector
}

func Execute() {
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	numberOfRequestedEvaluatorsSidecars := flag.Uint(util.SidecarEvaluatorsNumberFlag, 0, "number of requested evaluators sidecars")
	flag.Parse()
	var err error
	var app = AaqControllerApp{}

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()
	app.readyChan = make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.ctx = ctx

	webService := new(restful.WebService)
	webService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	webService.Route(webService.GET("/leader").To(app.leaderProbe).Doc("Leader endpoint"))
	restful.Add(webService)

	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		panic(err)
	}
	app.aaqNs = string(nsBytes)

	host, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}
	app.host = host

	app.aaqCli, err = client.GetAAQClient()
	app.arqInformer = informers.GetApplicationsResourceQuotaInformer(app.aaqCli)
	app.rqInformer = informers.GetResourceQuotaInformer(app.aaqCli)
	app.aaqjqcInformer = informers.GetAAQJobQueueConfig(app.aaqCli)
	app.podInformer = informers.GetPodInformer(app.aaqCli)
	app.aaqInformer = informers.GetAAQInformer(app.aaqCli)

	// Block until all requested evaluatorsSidecars are ready
	evaluatorsRegistry := aaq_evaluator.GetAaqEvaluatorsRegistry()
	err = evaluatorsRegistry.Collect(*numberOfRequestedEvaluatorsSidecars, util.DefaultSidecarsEvaluatorsStartTimeout)
	if err != nil {
		panic(err)
	}

	app.calcRegistry = aaq_evaluator.GetAaqEvaluatorsRegistry()
	stop := ctx.Done()
	app.initArqController(stop)
	app.initAaqGateController(stop)
	app.initRQController(stop)
	app.Run(stop)

	klog.V(2).Infoln("AAQ controller exited")

}

func (mca *AaqControllerApp) leaderProbe(_ *restful.Request, response *restful.Response) {
	res := map[string]interface{}{}
	select {
	case _, opened := <-mca.readyChan:
		if !opened {
			res["apiserver"] = map[string]interface{}{"leader": "true"}
			if err := response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON); err != nil {
				klog.Warningf("failed to return 200 OK reply: %v", err)
			}
			return
		}
	default:
	}
	res["apiserver"] = map[string]interface{}{"leader": "false"}
	if err := response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON); err != nil {
		klog.Warningf("failed to return 200 OK reply: %v", err)
	}
}

func (mca *AaqControllerApp) initArqController(stop <-chan struct{}) {
	mca.arqController = arq_controller.NewArqController(mca.aaqCli,
		mca.podInformer,
		mca.arqInformer,
		mca.rqInformer,
		mca.aaqjqcInformer,
		mca.calcRegistry,
		stop,
	)
}

func (mca *AaqControllerApp) initAaqGateController(stop <-chan struct{}) {
	mca.aaqGateController = arq_controller2.NewAaqGateController(mca.aaqCli,
		mca.podInformer,
		mca.arqInformer,
		mca.aaqjqcInformer,
		mca.calcRegistry,
		stop,
	)
}

func (mca *AaqControllerApp) initRQController(stop <-chan struct{}) {
	mca.rqController = rq_controller.NewRQController(mca.aaqCli,
		mca.rqInformer,
		mca.arqInformer,
		stop,
	)
}

func (mca *AaqControllerApp) Run(stop <-chan struct{}) {
	secretInformer := informers.GetSecretInformer(mca.aaqCli, mca.aaqNs)
	go secretInformer.Run(stop)
	if !cache.WaitForCacheSync(stop, secretInformer.HasSynced) {
		os.Exit(1)
	}

	secretCertManager := bootstrap.NewFallbackCertificateManager(
		bootstrap.NewSecretCertificateManager(
			util.SecretResourceName,
			mca.aaqNs,
			secretInformer.GetStore(),
		),
	)

	secretCertManager.Start()
	defer secretCertManager.Stop()

	tlsConfig := util.SetupTLS(secretCertManager)

	go func() {
		server := http.Server{
			Addr:      fmt.Sprintf("%s:%s", util.DefaultHost, strconv.Itoa(util.DefaultPort)),
			Handler:   http.DefaultServeMux,
			TLSConfig: tlsConfig,
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			golog.Fatal(err)
		}
	}()
	if err := mca.setupLeaderElector(); err != nil {
		golog.Fatal(err)
	}
	mca.leaderElector.Run(mca.ctx)
	panic("unreachable")
}

func (mca *AaqControllerApp) setupLeaderElector() (err error) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v14.EventSinkImpl{Interface: mca.aaqCli.CoreV1().Events(v1.NamespaceAll)})
	rl, err := resourcelock.New(mca.LeaderElection.ResourceLock,
		mca.aaqNs,
		"aaq-controller",
		mca.aaqCli.CoreV1(),
		mca.aaqCli.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      mca.host,
			EventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "aaq-controller"}),
		})

	if err != nil {
		return
	}

	mca.leaderElector, err = leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: mca.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: mca.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   mca.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: mca.onStartedLeading(),
				OnStoppedLeading: func() {
					golog.Fatal("leaderelection lost")
				},
			},
		})

	return
}

func (mca *AaqControllerApp) onStartedLeading() func(ctx context.Context) {
	return func(ctx context.Context) {
		stop := ctx.Done()

		go mca.podInformer.Run(stop)
		go mca.arqInformer.Run(stop)
		go mca.rqInformer.Run(stop)
		go mca.aaqjqcInformer.Run(stop)
		go mca.aaqInformer.Run(stop)

		if !cache.WaitForCacheSync(stop,
			mca.podInformer.HasSynced,
			mca.arqInformer.HasSynced,
			mca.aaqjqcInformer.HasSynced,
			mca.rqInformer.HasSynced,
			mca.aaqInformer.HasSynced,
		) {
			klog.Warningf("failed to wait for caches to sync")
		}

		go func() {
			mca.arqController.Run(context.Background(), 3)
		}()
		go func() {
			mca.aaqGateController.Run(context.Background(), 3)
		}()
		go func() {
			mca.rqController.Run(3)
		}()
		close(mca.readyChan)
	}
}
