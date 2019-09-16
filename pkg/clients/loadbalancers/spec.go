/*
Copyright 2019 Alexander Eldeib.
*/

package loadbalancers

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
)

type Spec struct {
	internal *network.LoadBalancer
}

func NewSpec() *Spec {
	return &Spec{
		internal: &network.LoadBalancer{},
	}
}

func NewSpecWithRemote(remote *network.LoadBalancer) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() network.LoadBalancer {
	return *s.internal
}

func Name(name *string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Name = name
	}
}

func Location(location *string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Location = location
	}
}
