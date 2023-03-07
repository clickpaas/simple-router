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

	"github.com/prometheus/client_golang/prometheus"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	crclientset "github.com/l0calh0st/simple-router/pkg/client/clientset/versioned"
	crlisterv1alpha1 "github.com/l0calh0st/simple-router/pkg/client/listers/simplerouter.l0calh0st.cn/v1alpha1"
	croperator "github.com/l0calh0st/simple-router/pkg/operator"
)

type operator struct {
	simpleClient  crclientset.Interface
	kubeClientSet kubernetes.Interface
	recorder      record.EventRecorder
	simpleLister  crlisterv1alpha1.SimpleLister
	reg           prometheus.Registerer
}

func NewOperator(kubeClientSet kubernetes.Interface, crClient crclientset.Interface, simpleLister crlisterv1alpha1.SimpleLister, recorder record.EventRecorder, reg prometheus.Registerer) croperator.Operator {
	return &operator{
		simpleClient:  crClient,
		kubeClientSet: kubeClientSet,
		simpleLister:  simpleLister,
		reg:           reg,
	}
}

func (o *operator) Reconcile(object interface{}) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(object.(string))
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to get the namespace and name from key: %v : %v", object, err))
		return nil
	}

	simple, err := o.simpleLister.Simples(namespace).Get(name)
	if err != nil {
		return handlerError(err)
	}
	_ = simple
	// write your code here
	return nil
}

func handlerError(err error) error {
	if k8serror.IsNotFound(err) {
		utilruntime.HandleError(err)
		return nil
	} else {
		return err
	}
}
