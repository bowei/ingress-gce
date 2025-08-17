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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/ingress-gce/pkg/apis/negbinding/v1"
	"k8s.io/ingress-gce/pkg/context"
	negbindingfake "k8s.io/ingress-gce/pkg/negbinding/clientset/versioned/fake"
)

func newTestController() (*Controller, *context.ControllerContext) {
	kubeClient := fake.NewSimpleClientset()
	negbindingClient := negbindingfake.NewSimpleClientset()
	ctx, err := context.NewControllerContext(kubeClient, negbindingClient, nil, nil, nil, nil, nil, "", 0)
	if err != nil {
		panic(err) // TODO: fixme
	}
	return NewController(ctx), ctx
}

func TestController(t *testing.T) {
	c, ctx := newTestController()
	stopCh := make(chan struct{})
	defer close(stopCh)
	go c.Run(stopCh)

	// Create a NetworkEndpointGroupBinding resource
	negBinding := &v1.NetworkEndpointGroupBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-binding",
			Namespace: "default",
		},
		Spec: v1.NetworkEndpointGroupBindingSpec{
			ServiceRef: v1.ServiceReference{
				Name: "test-service",
			},
			NetworkEndpointGroups: []v1.NetworkEndpointGroupReference{},
		},
	}
	_, err := c.negbindingClient.K8sV1().NetworkEndpointGroupBindings("default").Create(negBinding)
	if err != nil {
		t.Fatalf("Failed to create NetworkEndpointGroupBinding: %v", err)
	}

	// Update the NetworkEndpointGroupBinding resource
	negBinding.Spec.ServiceRef.Name = "other-service"
	_, err = c.negbindingClient.K8sV1().NetworkEndpointGroupBindings("default").Update(negBinding)
	if err != nil {
		t.Fatalf("Failed to update NetworkEndpointGroupBinding: %v", err)
	}

	// Delete the NetworkEndpointGroupBinding resource
	err = c.negbindingClient.K8sV1().NetworkEndpointGroupBindings("default").Delete(negBinding.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete NetworkEndpointGroupBinding: %v", err)
	}

	// Allow time for the controller to process the events
	time.Sleep(1 * time.Second)
}
