/*
Copyright 2019 Alexander Eldeib.
*/

package sqlfirewallrules

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2015-05-01-preview/sql"
	"github.com/google/go-cmp/cmp"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/clientutil"
)

type Spec struct {
	internal *sql.FirewallRule
}

func NewSpec() *Spec {
	return &Spec{
		internal: &sql.FirewallRule{},
	}
}

func NewSpecWithRemote(remote *sql.FirewallRule) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() sql.FirewallRule {
	return *s.internal
}

func Start(start *string) func(s *Spec) {
	return func(s *Spec) {
		// TODO(ace): move all initialization to spec.Set()
		clientutil.Initialize(
			[]func() bool{
				s.checkProperties,
			},
			[]func(){
				s.initProperties,
			},
		)
		s.internal.FirewallRuleProperties.StartIPAddress = start
	}
}

func End(end *string) func(s *Spec) {
	return func(s *Spec) {
		clientutil.Initialize(
			[]func() bool{
				s.checkProperties,
			},
			[]func(){
				s.initProperties,
			},
		)
		s.internal.FirewallRuleProperties.EndIPAddress = end
	}
}

func (s *Spec) Name() *string {
	return s.internal.Name
}

func (s *Spec) Start() *string {
	if s.internal.FirewallRuleProperties == nil {
		return nil
	}
	return s.internal.FirewallRuleProperties.StartIPAddress
}

func (s *Spec) End() *string {
	if s.internal.FirewallRuleProperties == nil {
		return nil
	}
	return s.internal.FirewallRuleProperties.EndIPAddress
}

func (s *Spec) ID() *string {
	return s.internal.ID
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.SQLFirewallRule) bool {
	return clientutil.Any([]func() bool{
		func() bool { return !cmp.Equal(s.Start(), &local.Spec.Start) },
		func() bool { return !cmp.Equal(s.End(), &local.Spec.End) },
		// func() bool { return !cmp.Equal(s.AdminLogin(), &local.Spec.Location) },
		// func() bool { return !cmp.Equal(s.AdminPassword(), &local.Spec.Location) },
	})
}

func (s *Spec) checkProperties() bool {
	return s.internal.FirewallRuleProperties == nil
}

func (s *Spec) initProperties() {
	s.internal.FirewallRuleProperties = &sql.FirewallRuleProperties{}
}
