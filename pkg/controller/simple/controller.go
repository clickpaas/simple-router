/*
   Copyright 2021 The simple-router Authors.
   Licensed under the Apache License, PROJECT_VERSION 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package simple

import (
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	apicorev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerappsv1 "k8s.io/client-go/listers/apps/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	crcontroller "github.com/l0calh0st/simple-router/pkg/controller"
)

// controller is implement Controller for Simple resources
type controller struct {
	crcontroller.Base
	register      prometheus.Registerer
	kubeClientSet kubeclientset.Interface
	queue         workqueue.RateLimitingInterface
	recorder      record.EventRecorder

	deploymentLister listerappsv1.DeploymentLister
	serviceLister    listercorev1.ServiceLister
	cacheSynced      []cache.InformerSynced

	config     Config
	globalTags map[string]tagItem
}

// NewController create a new controller for Simple resources
func NewController(cfg Config, kubeClientSet kubeclientset.Interface, kubeInformers informers.SharedInformerFactory, reg prometheus.Registerer) crcontroller.Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events(apicorev1.NamespaceAll)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, apicorev1.EventSource{Component: "Simple-operator"})

	return newSimpleController(cfg, kubeClientSet, kubeInformers, recorder, reg)
}

// newSimpleController is really
func newSimpleController(cfg Config, kubeClientSet kubeclientset.Interface, kubeInformers informers.SharedInformerFactory, recorder record.EventRecorder, reg prometheus.Registerer) *controller {
	c := &controller{
		register:      reg,
		kubeClientSet: kubeClientSet,
		recorder:      recorder,
		config:        cfg,
		globalTags:    map[string]tagItem{},
	}

	c.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	serviceInformer := kubeInformers.Core().V1().Services()
	c.serviceLister = serviceInformer.Lister()
	c.cacheSynced = append(c.cacheSynced, serviceInformer.Informer().HasSynced)

	deploymentInformer := kubeInformers.Apps().V1().Deployments()
	c.deploymentLister = deploymentInformer.Lister()
	deploymentInformer.Informer().AddEventHandler(newDeploymentEventHandler(c.enqueueFunc, c.deploymentLister, &c.config))
	c.cacheSynced = append(c.cacheSynced, deploymentInformer.Informer().HasSynced)

	// get tags list from command line
	for _, tag := range c.config.parseTags() {
		if _, ok := c.globalTags[tag.firstLevel+"|"+tag.secondLevel]; !ok {
			c.globalTags[tag.firstLevel+"|"+tag.secondLevel] = tag
		}
	}

	return c
}

func (c *controller) Start(worker int, stopCh <-chan struct{}) error {
	fmt.Fprintf(os.Stdout, "%v", c.globalTags)
	klog.Infof("Starting Simple-Router Controller, Waiting All Informer Synced......")
	// wait for all involved cached to be synced , before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, func() bool {
		for _, synced := range c.cacheSynced {
			if !synced() {
				return false
			}
		}
		return true
	}) {
		return fmt.Errorf("timeout wait for cache to be synced")
	}
	klog.Infof("Simple-Router Controller Started")
	for i := 0; i < worker; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	return nil
}

// runWorker for loop
func (c *controller) runWorker() {
	defer utilruntime.HandleCrash()
	for c.processNextItem() {
	}
}

func (c *controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	klog.Infof("Ending processing key %q", key)

	if err := c.Reconcile(key.(string)); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to reconcile simple %q: %v", key, err))
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *controller) Stop() {
	klog.Info("Stopping the simple operator controller")
	c.queue.ShutDown()
}

func (c *controller) enqueueFunc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("failed to get key for %v: %v", obj, err)
		return
	}
	c.queue.AddRateLimited(key)
}
