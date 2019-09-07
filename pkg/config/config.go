/*
Copyright 2019 Alexander Eldeib.
*/

package config

import (
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/go-logr/logr"
)

const userAgent = "ace/incendiary-iguana"

// AuthorizationMode is a string to identify the Azure authentication mode at startup.
type AuthorizationMode string

const (
	FileMode        AuthorizationMode = "file"
	CLIMode         AuthorizationMode = "cli"
	EnvironmentMode AuthorizationMode = "environment"
)

var (
	mode AuthorizationMode
)

type Config struct {
	log        logr.Logger
	internal   autorest.Authorizer
	kvinternal *autorest.Authorizer
}

// New returns a concrete implementation of the Config interface
func New(log logr.Logger) *Config {
	return &Config{
		log: log,
	}
}

func (c *Config) Settings() (auth.EnvironmentSettings, error) {
	return auth.GetSettingsFromEnvironment()
}

// GetAuthorizer creates a new ARM authorizer, preferring cli => file => env vars => msi.
func (c *Config) DetectAuthorizer() error {
	internal, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	if err == nil {
		mode = FileMode
		c.internal = internal
		c.log.WithValues("authorization_mode", "file").Info("")
		return nil
	}
	c.log.WithValues("authorization_mode", "file").Error(err, "failed to get authorizer")
	internal, err = auth.NewAuthorizerFromCLI()
	if err == nil {
		c.internal = internal
		mode = CLIMode
		c.log.WithValues("authorization_mode", "cli").Info("")
		return nil
	}
	c.log.WithValues("authorization_mode", "cli").Error(err, "failed to get authorizer")
	internal, err = auth.NewAuthorizerFromEnvironment()
	if err == nil {
		mode = EnvironmentMode
		c.internal = internal
		c.log.WithValues("authorization_mode", "environment").Info("")
		return nil
	}
	c.log.WithValues("authorization_mode", "environment").Error(err, "failed to get authorizer")
	return err
}

// AuthorizerClient configures the provided client with an authorizer or returns an error.

// GetAuthorizer creates a new ARM authorizer, preferring cli => file => env vars => msi.
func (c *Config) GetAuthorizer() (autorest.Authorizer, error) {
	// TODO(ace): use detected mode and don't do the whole loop every time.
	// authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	// if err == nil {
	// 	return authorizer, nil
	// }
	// authorizer, err = auth.NewAuthorizerFromCLI()
	// if err == nil {
	// 	return authorizer, nil
	// }
	// return auth.NewAuthorizerFromEnvironment()
	return c.internal, nil
}

// GetKeyvaultAuthorizer creates a new Keyvault authorizer, preferring cli => file => env vars => msi.
func (c *Config) GetKeyvaultAuthorizer() (autorest.Authorizer, error) {
	// TODO(ace): use detected mode and don't do the whole loop every time.
	if c.kvinternal != nil {
		return *c.kvinternal, nil
	}
	authorizer, err := kvauth.NewAuthorizerFromFile(azure.PublicCloud.KeyVaultEndpoint)
	if err == nil {
		c.kvinternal = &authorizer
		return authorizer, nil
	}
	authorizer, err = kvauth.NewAuthorizerFromCLI()
	if err == nil {
		c.kvinternal = &authorizer
		return authorizer, nil
	}
	authorizer, err = kvauth.NewAuthorizerFromEnvironment()
	if err == nil {
		c.kvinternal = &authorizer
		return authorizer, nil
	}
	return authorizer, err
}

// AuthorizerClient configures the provided client with an authorizer or returns an error.
// It takes an autorest client and configures its user agent as well as Azure credentials.
func (c *Config) AuthorizeClient(client *autorest.Client) error {
	authorizer, err := c.GetAuthorizer()
	if err != nil {
		return err
	}
	client.Authorizer = authorizer
	if err := client.AddToUserAgent(userAgent); err != nil {
		return err
	}
	return nil
}
