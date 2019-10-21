/*
Copyright 2019 Alexander Eldeib.
*/

package sqlservers

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2015-05-01-preview/sql"
	"github.com/google/go-cmp/cmp"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/clientutil"
)

type Spec struct {
	internal *sql.Server
}

func NewSpec() *Spec {
	return &Spec{
		internal: &sql.Server{},
	}
}

func NewSpecWithRemote(remote *sql.Server) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() sql.Server {
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

func AdminLogin(login *string) func(s *Spec) {
	return func(s *Spec) {
		clientutil.Initialize(
			[]func() bool{
				s.checkServerProperties,
			},
			[]func(){
				s.initServerProperties,
			},
		)
		s.internal.ServerProperties.AdministratorLogin = login
	}
}

func AdminPassword(password *string) func(s *Spec) {
	return func(s *Spec) {
		clientutil.Initialize(
			[]func() bool{
				s.checkServerProperties,
			},
			[]func(){
				s.initServerProperties,
			},
		)
		s.internal.ServerProperties.AdministratorLoginPassword = password
	}
}

func (s *Spec) Name() *string {
	return s.internal.Name
}

func (s *Spec) Location() *string {
	return s.internal.Location
}

func (s *Spec) ID() *string {
	return s.internal.ID
}

func (s *Spec) State() *string {
	if s.internal.ServerProperties == nil {
		return nil
	}
	return s.internal.State
}

func (s *Spec) AdminLogin() *string {
	if s.internal.ServerProperties == nil {
		return nil
	}
	return s.internal.AdministratorLogin
}

func (s *Spec) AdminPassword() *string {
	if s.internal.ServerProperties == nil {
		return nil
	}
	return s.internal.AdministratorLoginPassword
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.SQLServer) bool {
	return clientutil.Any([]func() bool{
		func() bool { return !cmp.Equal(s.Name(), &local.Spec.Name) },
		func() bool { return !cmp.Equal(s.Location(), &local.Spec.Location) },
		// func() bool { return !cmp.Equal(s.AdminLogin(), &local.Spec.Location) },
		// func() bool { return !cmp.Equal(s.AdminPassword(), &local.Spec.Location) },
	})
}

func (s *Spec) checkServerProperties() bool {
	return s.internal.ServerProperties == nil
}

func (s *Spec) initServerProperties() {
	s.internal.ServerProperties = &sql.ServerProperties{}
}
