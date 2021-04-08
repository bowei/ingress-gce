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
	"reflect"
	"testing"

	"github.com/kr/pretty"
	core "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
)

func Test_endpointsSubsetToSlice(t *testing.T) {
	type args struct {
		s *core.EndpointSubset
	}
	tests := []struct {
		name string
		args args
		want *discovery.EndpointSlice
	}{
		{
			name: "empty",
			args: args{
				s: &core.EndpointSubset{
					Ports: []core.EndpointPort{
						{
							Name:     "web",
							Port:     8080,
							Protocol: core.ProtocolTCP,
						},
					},
				},
			},
			want: &discovery.EndpointSlice{
				Ports: []discovery.EndpointPort{
					{
						Name:     stringPtr("web"),
						Port:     int32Ptr(8080),
						Protocol: protocolPtr(core.ProtocolTCP),
					},
				},
			},
		},
		{
			name: "1-ready",
			args: args{
				s: &core.EndpointSubset{
					Addresses: []core.EndpointAddress{
						{IP: "1.2.3.4"},
					},
					Ports: []core.EndpointPort{
						{
							Name:     "web",
							Port:     8080,
							Protocol: core.ProtocolTCP,
						},
					},
				},
			},
			want: &discovery.EndpointSlice{
				Endpoints: []discovery.Endpoint{
					{
						Addresses: []string{"1.2.3.4"},
						Conditions: discovery.EndpointConditions{
							Ready: boolPtr(true),
						},
					},
				},
				Ports: []discovery.EndpointPort{
					{
						Name:     stringPtr("web"),
						Port:     int32Ptr(8080),
						Protocol: protocolPtr(core.ProtocolTCP),
					},
				},
			},
		},
		{
			name: "1-not-ready",
			args: args{
				s: &core.EndpointSubset{
					NotReadyAddresses: []core.EndpointAddress{
						{IP: "1.2.3.4"},
					},
					Ports: []core.EndpointPort{
						{
							Name:     "web",
							Port:     8080,
							Protocol: core.ProtocolTCP,
						},
					},
				},
			},
			want: &discovery.EndpointSlice{
				Endpoints: []discovery.Endpoint{
					{
						Addresses: []string{"1.2.3.4"},
						Conditions: discovery.EndpointConditions{
							Ready: boolPtr(false),
						},
					},
				},
				Ports: []discovery.EndpointPort{
					{
						Name:     stringPtr("web"),
						Port:     int32Ptr(8080),
						Protocol: protocolPtr(core.ProtocolTCP),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endpointsSubsetToSlice(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("endpointsSubsetToSlice() = %s\nwant %s", pretty.Sprint(got), pretty.Sprint(tt.want))
			}
		})
	}
}
