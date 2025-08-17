/*
Copyright 2023 The Kubernetes Authors.

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

package negbinding

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	nbv1 "k8s.io/ingress-gce/pkg/apis/negbinding/v1"
	"k8s.io/ingress-gce/pkg/negbinding/clientset/versioned"
	negfake "k8s.io/ingress-gce/pkg/negbinding/clientset/versioned/fake"
	informersv1 "k8s.io/ingress-gce/pkg/negbinding/informers/externalversions"
)

const (
	testServiceNamespace = "test-ns"
	testServiceName      = "test-name"
)

type testContext struct {
	client             kubernetes.Interface
	negbindingClient   versioned.Interface
	negbindingInformer cache.SharedIndexInformer
	serviceInformer    cache.SharedIndexInformer
	recorder           *record.FakeRecorder
	stopCh             chan struct{}
}

func (tc *testContext) k8s() kubernetes.Interface {
	return tc.client
}

func (tc *testContext) negBinding() cache.SharedIndexInformer {
	return tc.negbindingInformer
}

func (tc *testContext) negBindingClient() versioned.Interface {
	return tc.negbindingClient
}

func (tc *testContext) service() cache.SharedIndexInformer {
	return tc.serviceInformer
}

func (tc *testContext) newRecorder(ns string) record.EventRecorder {
	return tc.recorder
}

func newTestContext() *testContext {
	client := fake.NewSimpleClientset()
	negbindingClient := negfake.NewSimpleClientset()
	informerFactory := kubeinformers.NewSharedInformerFactory(client, 0)
	negInformerFactory := informersv1.NewSharedInformerFactory(negbindingClient, 0)

	return &testContext{
		client:             client,
		negbindingClient:   negbindingClient,
		negbindingInformer: negInformerFactory.K8s().V1().NetworkEndpointGroupBindings().Informer(),
		serviceInformer:    informerFactory.Core().V1().Services().Informer(),
		recorder:           record.NewFakeRecorder(100),
		stopCh:             make(chan struct{}),
	}
}

func newTestController() (*Controller, *testContext) {
	tc := newTestContext()
	c := NewController(tc)
	return c, tc
}

func newTestService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testServiceNamespace,
			Name:      testServiceName,
		},
	}
}

func newTestNetworkEndpointGroupBinding() *nbv1.NetworkEndpointGroupBinding {
	return &nbv1.NetworkEndpointGroupBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testServiceNamespace,
			Name:      testServiceName,
		},
		Spec: nbv1.NetworkEndpointGroupBindingSpec{
			ServiceRef: nbv1.ServiceReference{
				Name: testServiceName,
			},
		},
	}
}

func TestController(t *testing.T) {
	c, tc := newTestController()
	defer close(tc.stopCh)
	go c.Run(tc.stopCh)

	// Add a NetworkEndpointGroupBinding.
	negBinding := newTestNetworkEndpointGroupBinding()
	tc.negbindingInformer.GetStore().Add(negBinding)
	c.handleNetworkEndpointGroupBindingAdd(negBinding)

	// Add a Service.
	service := newTestService()
	tc.serviceInformer.GetStore().Add(service)
	c.handleServiceAdd(service)

	// TODO: Add assertions to verify the controller's behavior.
	// For now, we just check that the controller runs without crashing.
}

func TestControllerProcessService(t *testing.T) {
	// TODO: Implement tests for service processing logic.
}

func TestControllerProcessNetworkEndpointGroupBinding(t *testing.T) {
	// TODO: Implement tests for NetworkEndpointGroupBinding processing logic.
}
