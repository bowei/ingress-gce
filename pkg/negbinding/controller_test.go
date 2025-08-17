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
	"context"
	"testing"
	"time"

	gcpfirewallclient "github.com/GoogleCloudPlatform/gke-networking-api/client/gcpfirewall/clientset/versioned/fake"
	networkclient "github.com/GoogleCloudPlatform/gke-networking-api/client/network/clientset/versioned/fake"
	nodetopologyclient "github.com/GoogleCloudPlatform/gke-networking-api/client/nodetopology/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/ingress-gce/pkg/apis/negbinding/v1"
	backendconfigclient "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned/fake"
	gce_context "k8s.io/ingress-gce/pkg/context"
	frontendconfigclient "k8s.io/ingress-gce/pkg/frontendconfig/client/clientset/versioned/fake"
	negbindingfake "k8s.io/ingress-gce/pkg/negbinding/clientset/versioned/fake"
	serviceattachmentclient "k8s.io/ingress-gce/pkg/serviceattachment/client/clientset/versioned/fake"
	svcnegclient "k8s.io/ingress-gce/pkg/svcneg/client/clientset/versioned/fake"
	"k8s.io/klog/v2"
)

func newTestController() *Controller {
	kubeClient := fake.NewSimpleClientset()
	negbindingClient := negbindingfake.NewSimpleClientset()
	backendconfigClient := backendconfigclient.NewSimpleClientset()
	frontendconfigClient := frontendconfigclient.NewSimpleClientset()
	svcnegClient := svcnegclient.NewSimpleClientset()
	serviceattachmentClient := serviceattachmentclient.NewSimpleClientset()
	gcpfirewallClient := gcpfirewallclient.NewSimpleClientset()
	networkClient := networkclient.NewSimpleClientset()
	nodetopologyClient := nodetopologyclient.NewSimpleClientset()

	ctx, err := gce_context.NewControllerContext(
		kubeClient,
		backendconfigClient,
		frontendconfigClient,
		gcpfirewallClient,
		svcnegClient,
		serviceattachmentClient,
		networkClient,
		nodetopologyClient,
		kubeClient,
		nil, // gce cloud
		nil, // namer
		"",  // kube system uid
		gce_context.ControllerContextConfig{},
		klog.TODO(),
	)
	if err != nil {
		panic(err)
	}

	return NewController(ctx, negbindingClient)
}

func TestController(t *testing.T) {
	c := newTestController()
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
	_, err := c.negbindingClient.K8sV1().NetworkEndpointGroupBindings("default").Create(context.TODO(), negBinding, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create NetworkEndpointGroupBinding: %v", err)
	}
	// Update the NetworkEndpointGroupBinding resource
	negBinding.Spec.ServiceRef.Name = "other-service"
	_, err = c.negbindingClient.K8sV1().NetworkEndpointGroupBindings("default").Update(context.TODO(), negBinding, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update NetworkEndpointGroupBinding: %v", err)
	}

	// Delete the NetworkEndpointGroupBinding resource
	err = c.negbindingClient.K8sV1().NetworkEndpointGroupBindings("default").Delete(context.TODO(), negBinding.Name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete NetworkEndpointGroupBinding: %v", err)
	}

	// Allow time for the controller to process the events
	time.Sleep(1 * time.Second)
}
