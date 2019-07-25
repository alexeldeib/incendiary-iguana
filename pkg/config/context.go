/*
Copyright 2019 Alexander Eldeib.
*/

package config

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Context struct {
	client.Client
	Log logr.Logger
	Config
}
