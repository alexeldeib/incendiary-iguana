/*
Copyright 2019 Alexander Eldeib.
*/

//go:generate mockgen -package mock_config -copyright_file ../../../hack/boilerplate.go.txt -destination ./zz_generated_mock.go github.com/alexeldeib/incendiary-iguana/pkg/config Config

package mock_config

import "github.com/alexeldeib/incendiary-iguana/pkg/config"

type Config config.Config
