/*
Copyright 2021 The Kubernetes Authors.
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
package endpoints

import (
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type WatchEventFunc func(ns, name string)

type CacheItem struct {
	E []*discovery.EndpointSlice
}

type Cache interface {
	Init() error
	List(ns string)
	Get(ns, name string) (*discovery.EndpointSlice, error)
	Watch(f WatchEventFunc)
}

func NewCache(k8s kubernetes.Interface) (Cache, error) {
	hasEPS, err := hasEndpointSliceSupport(k8s)
	if err != nil {
		return nil, err
	}
	if hasEPS {
		klog.Info("API supports EndpointSlices, using EndpointSlice cache")
		ret := &cacheForEndpointSlice{}
		return ret, nil
	}
	ret := &cacheForEndpoints{}
	klog.Info("API does not support EndpointSlices, using Endpoints cache")

	return ret, nil
}

func hasEndpointSliceSupport(k8s kubernetes.Interface) (bool, error) {
	apis, err := k8s.Discovery().ServerResourcesForGroupVersion(discovery.SchemeGroupVersion.String())
	if err != nil {
		return false, err
	}
	for _, a := range apis.APIResources {
		if a.Kind == "EndpointSlice" {
			return true, nil
		}
	}

	// TODO(bowei): May want to disable to wait for EndpointMirroring.

	return false, nil
}
