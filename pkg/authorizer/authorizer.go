package authorizer

import (
	"errors"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/go-logr/logr"
)

type builder struct {
	log      logr.Logger
	env      azure.Environment
	app      string
	key      string
	tenant   string
	token    string
	file     string
	resource string
}

type Option func(*builder) *builder

func New(opts ...Option) (autorest.Authorizer, error) {
	b := &builder{
		env: azure.PublicCloud,
	}
	for _, opt := range opts {
		b = opt(b)
	}
	return b.build()
}

func Logger(log logr.Logger) func(*builder) *builder {
	return func(b *builder) *builder {
		b.log = log
		return b
	}
}

func Environment(env azure.Environment) func(*builder) *builder {
	return func(b *builder) *builder {
		b.env = env
		return b
	}
}

func Resource(resource string) func(*builder) *builder {
	return func(b *builder) *builder {
		b.resource = resource
		return b
	}
}

func ClientCredentials(app, key, tenant string) func(*builder) *builder {
	return func(b *builder) *builder {
		b.app = app
		b.key = key
		b.tenant = tenant
		return b
	}
}

func Token(token string) func(*builder) *builder {
	return func(b *builder) *builder {
		b.token = token
		return b
	}
}

func File(file string) func(*builder) *builder {
	return func(b *builder) *builder {
		b.file = file
		return b
	}
}

func (b *builder) build() (autorest.Authorizer, error) {
	if b.resource == "" {
		b.resource = b.env.ResourceManagerEndpoint
	}

	// TODO(ace): msi, token, file, switch on mode?
	return b.getClientCredentials()
}

func (b *builder) getClientCredentials() (autorest.Authorizer, error) {
	if err := validateClientCredentials(b.app, b.key, b.tenant); err != nil {
		return nil, err
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

func validateClientCredentials(app, key, tenant string) error {
	if app == "" || tenant == "" || key == "" {
		return errors.New("app, tenant, and key must all be provided as options for authenticating with client credentials")
	}
	return nil
}

func GetEnvironment(env string) (azure.Environment, error) {
	if env == "" {
		return azure.PublicCloud, nil
	}
	return azure.EnvironmentFromName(env)
}
