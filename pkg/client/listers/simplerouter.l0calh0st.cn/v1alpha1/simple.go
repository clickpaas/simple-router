/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/l0calh0st/simple-router/pkg/apis/simplerouter.l0calh0st.cn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SimpleLister helps list Simples.
// All objects returned here must be treated as read-only.
type SimpleLister interface {
	// List lists all Simples in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Simple, err error)
	// Simples returns an object that can list and get Simples.
	Simples(namespace string) SimpleNamespaceLister
	SimpleListerExpansion
}

// simpleLister implements the SimpleLister interface.
type simpleLister struct {
	indexer cache.Indexer
}

// NewSimpleLister returns a new SimpleLister.
func NewSimpleLister(indexer cache.Indexer) SimpleLister {
	return &simpleLister{indexer: indexer}
}

// List lists all Simples in the indexer.
func (s *simpleLister) List(selector labels.Selector) (ret []*v1alpha1.Simple, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Simple))
	})
	return ret, err
}

// Simples returns an object that can list and get Simples.
func (s *simpleLister) Simples(namespace string) SimpleNamespaceLister {
	return simpleNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// SimpleNamespaceLister helps list and get Simples.
// All objects returned here must be treated as read-only.
type SimpleNamespaceLister interface {
	// List lists all Simples in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Simple, err error)
	// Get retrieves the Simple from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Simple, error)
	SimpleNamespaceListerExpansion
}

// simpleNamespaceLister implements the SimpleNamespaceLister
// interface.
type simpleNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Simples in the indexer for a given namespace.
func (s simpleNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Simple, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Simple))
	})
	return ret, err
}

// Get retrieves the Simple from the indexer for a given namespace and name.
func (s simpleNamespaceLister) Get(name string) (*v1alpha1.Simple, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("simple"), name)
	}
	return obj.(*v1alpha1.Simple), nil
}