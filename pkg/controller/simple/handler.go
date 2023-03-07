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
	"strings"

	apiappsv1 "k8s.io/api/apps/v1"
	apicorev1 "k8s.io/api/core/v1"
	listerappsv1 "k8s.io/client-go/listers/apps/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

type serviceEventHandler struct {
	enqueueFn     func(obj interface{})
	serviceLister listercorev1.ServiceLister
}

func (h *serviceEventHandler) OnAdd(obj interface{}) {
	svc, ok := obj.(*apicorev1.Service)
	if !ok {
		return
	}
	if !strings.HasPrefix(svc.GetName(), "dmo") || !strings.HasSuffix(svc.GetName(), "-rest") {
		return
	}
	klog.Infof("Service %s/%s added", svc.GetName(), svc.GetNamespace())
	h.enqueueFn(svc)
}

func (h *serviceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	newSvc, ok := newObj.(*apicorev1.Service)
	if !ok {
		return
	}
	oldSvc, ok := oldObj.(*apicorev1.Service)
	if !ok {
		return
	}
	if !strings.HasPrefix(newSvc.GetName(), "dmo") || !strings.HasSuffix(newSvc.GetName(), "-rest") {
		return
	}
	if newSvc.GetObjectMeta().GetResourceVersion() == oldSvc.GetObjectMeta().GetResourceVersion() {
		// oldservice and newservice resource are the same, not need do change
		return
	}
	klog.Infof("Service %s/%s updated", newSvc.GetName(), newSvc.GetNamespace())
	h.enqueueFn(newSvc)
}

func (h *serviceEventHandler) OnDelete(obj interface{}) {
	svc, ok := obj.(*apicorev1.Service)
	if !ok {
		return
	}
	klog.Infof("Service %s/%s deleted", svc.GetName(), svc.GetNamespace())
	h.enqueueFn(svc)
}

func newServiceEventHandler(enqueueFn func(obj interface{}), lister listercorev1.ServiceLister) *serviceEventHandler {
	return &serviceEventHandler{
		enqueueFn:     enqueueFn,
		serviceLister: lister,
	}
}

type deploymentEventHandler struct {
	enqueueFn        func(key interface{})
	deploymentLister listerappsv1.DeploymentLister
	config           *Config
}

func (h *deploymentEventHandler) OnAdd(obj interface{}) {
	dpl, ok := obj.(*apiappsv1.Deployment)
	if !ok {
		return
	}
	if !h.deploymentFilter(dpl.GetName()) {
		return
	}
	h.enqueueFn(dpl)
}

func (h *deploymentEventHandler) OnUpdate(oldObj, newObj interface{}) {
	newDpl, ok := newObj.(*apiappsv1.Deployment)
	if !ok {
		return
	}
	oldDpl, ok := oldObj.(*apiappsv1.Deployment)
	if !ok {
		return
	}
	if newDpl.ResourceVersion == oldDpl.ResourceVersion {
		// resource doesn't changed
		return
	}
	if !h.deploymentFilter(newDpl.GetName()) {
		return
	}
	h.enqueueFn(newDpl)
}

func (h *deploymentEventHandler) OnDelete(obj interface{}) {
	dpl, ok := obj.(*apiappsv1.Deployment)
	if !ok {
		return
	}
	if !h.deploymentFilter(dpl.GetName()) {
		return
	}
}

func newDeploymentEventHandler(enqueueFn func(key interface{}), lister listerappsv1.DeploymentLister, config *Config) *deploymentEventHandler {
	return &deploymentEventHandler{
		enqueueFn:        enqueueFn,
		deploymentLister: lister,
		config:           config,
	}
}

// return true skip, return false continue handler
func (h *deploymentEventHandler) deploymentFilter(key string) bool {
	if !strings.HasPrefix(key, "dmo") {
		return false
	}
    // 包含tag的不做处理
    if strings.Contains(key, "-tag") {
        return false
    }
	// 在白名单里面继续处理
	for _, wk := range h.config.WhiteDplList {
		if strings.HasPrefix(key, wk) {
			return true
		}
	}
	// 不在白名单里面的，必须带有rest的，否则不继续处理
	if strings.Contains(key, "-rest-") {
		return true
	}
	return false
}
