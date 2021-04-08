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
	"fmt"

	core "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
)

// cacheForEndpoints supports both Endpoints and EndpointSlice, automatically
// defaulting to using EndpointSlice if it is available.
type cacheForEndpoints struct {
}

func (c *cacheForEndpoints) Init() error {

	return nil // XXX/bowei
}

func (c *cacheForEndpoints) List(ns string) {
}

func (c *cacheForEndpoints) Get(ns, name string) (*discovery.EndpointSlice, error) {
	return nil, nil // XXX/bowei
}

func (c *cacheForEndpoints) Watch(f WatchEventFunc) {
}

func endpointsToCacheItem(ep *core.Endpoints) *CacheItem {
	ret := &CacheItem{}
	eps := &discovery.EndpointSlice{}
	// ret.E = eps

	eps.TypeMeta = ep.TypeMeta
	eps.ObjectMeta = ep.ObjectMeta

	// XXX/bowei -- address family?

	// Ports x Subset => Slice

	for _, s := range ep.Subsets {
		// Each subset is a single slice.
		sl := endpointsSubsetToSlice(&s)
		// XXX/bowei -- metadata

		fmt.Println(s)
		fmt.Println(sl)
		/*
			for _, addr := range s.Addresses {
				e := discovery.Endpoint{}
				eps.Endpoints = append(eps.Endpoints, e)
			}
			for _, addr := range s.NotReadyAddresses {

			}*/
		//s.Ports

	}

	return ret
}

func endpointsSubsetToSlice(s *core.EndpointSubset) *discovery.EndpointSlice {
	eps := discovery.EndpointSlice{}
	// eps.TypeMeta = ep.TypeMeta
	// eps.ObjectMeta = ep.ObjectMeta
	for _, addr := range s.Addresses {
		eps.Endpoints = append(eps.Endpoints, *convertEndpoint( /*ready*/ true, &addr))
	}
	for _, addr := range s.NotReadyAddresses {
		eps.Endpoints = append(eps.Endpoints, *convertEndpoint( /*ready*/ false, &addr))
	}
	for _, port := range s.Ports {
		slicePort := discovery.EndpointPort{}
		// XXX/bowei -- there are subtle differences in the fields due to
		// optional and pointer types.
		if port.Name != "" {
			slicePort.Name = stringPtr(port.Name)
		}
		// XXX/bowei -- protocol is both optional and defaults to TCP?? Does it
		// ever show up as empty?
		slicePort.Protocol = protocolPtr(port.Protocol)
		slicePort.Port = int32Ptr(port.Port)
		if port.AppProtocol != nil {
			slicePort.AppProtocol = stringPtr(*port.AppProtocol)
		}
		eps.Ports = append(eps.Ports, slicePort)
	}
	return &eps
}

func convertEndpoint(ready bool, a *core.EndpointAddress) *discovery.Endpoint {
	ret := &discovery.Endpoint{
		Addresses:  []string{a.IP},
		Conditions: discovery.EndpointConditions{Ready: boolPtr(ready)},
	}
	if a.Hostname != "" {
		ret.Hostname = stringPtr(a.Hostname)
	}
	if a.NodeName != nil {
		ret.Topology = map[string]string{
			"topology.kubernetes.io/hostname": *a.NodeName,
		}
	}
	if a.TargetRef != nil {
		ret.TargetRef = a.TargetRef.DeepCopy()
	}
	return ret
}

func int32Ptr(v int32) *int32                    { return &v }
func boolPtr(v bool) *bool                       { return &v }
func stringPtr(v string) *string                 { return &v }
func protocolPtr(v core.Protocol) *core.Protocol { return &v }
