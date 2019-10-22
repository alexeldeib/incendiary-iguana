package authorizer

import (
	"errors"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type Builder struct {
	env      azure.Environment
	app      string
	key      string
	tenant   string
	resource string
}

func NewBuilder() *Builder {
	return &Builder{
		env: azure.PublicCloud,
	}
}

func (b *Builder) In(env azure.Environment) *Builder {
	b.env = env
	return b
}

func (b *Builder) For(resource string) *Builder {
	b.resource = resource
	return b
}

func (b *Builder) WithClientCredentials(app, key, tenant string) *Builder {
	b.app = app
	b.key = key
	b.tenant = tenant
	return b
}

func (b *Builder) Build() (autorest.Authorizer, error) {
	if b.app == "" || b.tenant == "" || b.key == "" {
		return nil, errors.New("app, tenant, and key must all be provided as options for authenticating with client credentials")
	}
	if b.resource == "" {
		b.resource = b.env.ResourceManagerEndpoint
	}
	creds := &auth.ClientCredentialsConfig{
		ClientID:     b.app,
		ClientSecret: b.key,
		TenantID:     b.tenant,
		Resource:     b.resource,
		AADEndpoint:  b.env.ActiveDirectoryEndpoint,
	}
	return creds.Authorizer()
}

func GetEnvironment(env string) (azure.Environment, error) {
	if env == "" {
		return azure.PublicCloud, nil
	}
	return azure.EnvironmentFromName(env)
}

// mgmtAuthorizer, err := NewBuilder()
// 					.In(azure.PublicCloud)
// 					.WithClientCredentials(app, key, tenant)
// 					.Build()
// kvAuthorizer, err := NewBuilder()
// 					.In(azure.PublicCloud)
// 					.For(strings.TrimSuffix(azure.PublicCloud.KeyVaultEndpoint))
// 					.WithClientCredentials(app, key, tenant)
// 					.Build()
// kvAuthorizer := NewBuilder()
// 					.In(azure.PublicCloud)
// 					.For(strings.TrimSuffix(azure.PublicCloud.KeyVaultEndpoint))
// 					.WithClientCredentials(app, key, tenant)
// 					.Build()
