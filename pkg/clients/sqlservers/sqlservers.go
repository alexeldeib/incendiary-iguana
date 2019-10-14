/*
Copyright 2019 Alexander Eldeib.
*/

package sqlservers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2015-05-01-preview/sql"
	"github.com/sanity-io/litter"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/clientutil"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/sqlfirewallrules"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory    factoryFunc
	internal   sql.ServersClient
	firewalls  *sqlfirewallrules.Client
	kubeclient *client.Client
	config     *config.Config
	scheme     *runtime.Scheme
}

type factoryFunc func(subscriptionID string) sql.ServersClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config, kubeclient *client.Client, scheme *runtime.Scheme) *Client {
	return NewWithFactory(configuration, kubeclient, sql.NewServersClient, scheme)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration *config.Config, kubeclient *client.Client, factory factoryFunc, scheme *runtime.Scheme) *Client {
	return &Client{
		config:     configuration,
		factory:    factory,
		kubeclient: kubeclient,
		scheme:     scheme,
		firewalls:  sqlfirewallrules.New(configuration),
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	c.internal = c.factory(local.Spec.SubscriptionID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a SQL server in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}

	// Set status
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return err
	}

	// TODO(ace): create something like SQLServerCredential CRD, and pivot on state of that
	// Will allow for higher level orchestration better than the raw Kubernetes secret (?)
	targetSecret, err := c.ensureSecret(ctx, local)
	if err != nil {
		return err
	}

	// Pull from secret. Known to exist by construction.
	adminLogin := string(targetSecret.Data["username"])
	adminPassword := string(targetSecret.Data["password"])

	// Wrap, check status, and exit early if appropriate
	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		// TODO(ace): this is not checking whether the secret needs to be updated
		// TODO(ace): this should be an extension point to gracefully handle immutable updates
		// if !spec.NeedsUpdate(local) {
		// 	return nil
		// }
	} else {
		spec = NewSpec()
	}

	// Overlay new properties over old/default spec
	spec.Set(
		Name(&local.Spec.Name),
		Location(&local.Spec.Location),
		AdminLogin(&adminLogin), // n.b., immutable
		AdminPassword(&adminPassword),
	)

	// Apply to Azure. Use Update() if the object was found, to ensure that we set the password.
	if found {
		updateProps := sql.ServerUpdate{
			ServerProperties: spec.Build().ServerProperties,
		}
		future, err := c.internal.Update(ctx, local.Spec.ResourceGroup, local.Spec.Name, updateProps)
		if err != nil {
			return err
		}
		b, err := future.MarshalJSON()
		litter.Dump(string(b))
		if err := future.WaitForCompletionRef(ctx, c.internal.Client); err != nil {
			return err
		}
	} else {
		// Opt to allow for blocking calls, highly parallelizing the controller instead.
		future, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec.Build())
		if err != nil {
			return nil
		}
		b, err := future.MarshalJSON()
		litter.Dump(string(b))
		if err := future.WaitForCompletionRef(ctx, c.internal.Client); err != nil {
			return err
		}
	}

	// Block access after creation if desired
	if err := c.ensureRule(ctx, local); err != nil {
		return err
	}
	return nil
}

func (c *Client) ensureSecret(ctx context.Context, local *azurev1alpha1.SQLServer) (*corev1.Secret, error) {
	// Set up secret name/object
	targetName := types.NamespacedName{
		Name:      local.Spec.Name,
		Namespace: local.ObjectMeta.Namespace,
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	// Get SQL Server secret
	getErr := (*c.kubeclient).Get(ctx, targetName, targetSecret)
	if client.IgnoreNotFound(getErr) != nil {
		return nil, getErr
	}

	// Ensure secret exists
	if apierrs.IsNotFound(getErr) {
		targetSecret.Data = map[string][]byte{
			"username":           []byte(clientutil.GenerateRandomString(8)),
			"password":           []byte(clientutil.GenerateRandomString(16)),
			"sqlservernamespace": []byte(local.ObjectMeta.Namespace),
			"sqlservername":      []byte(local.ObjectMeta.Name),
		}
		if local.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(local, targetSecret, c.scheme); ownerErr != nil {
				return nil, ownerErr
			}
		}
		if err := (*c.kubeclient).Create(ctx, targetSecret); err != nil {
			return nil, err
		}
	}

	return targetSecret, nil
}

func (c *Client) ensureRule(ctx context.Context, local *azurev1alpha1.SQLServer) error {
	rule := &azurev1alpha1.SQLFirewallRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
		Spec: azurev1alpha1.SQLFirewallRuleSpec{
			Name:           "AllowAzureAccess",
			SubscriptionID: local.Spec.SubscriptionID,
			ResourceGroup:  local.Spec.ResourceGroup,
			Server:         local.Spec.Name,
			Start:          "0.0.0.0",
			End:            "0.0.0.0",
		},
	}

	if err := c.firewalls.ForSubscription(ctx, rule); err != nil {
		return err
	}

	// probably should be bool, not *bool
	if local.Spec.AllowAzureServiceAccess != nil && *local.Spec.AllowAzureServiceAccess {
		if local.ObjectMeta.UID != "" {
			if err := controllerutil.SetControllerReference(local, rule, c.scheme); err != nil {
				return err
			}
		}
		// TODO(ace): improve delegation//handle this somewhere else?
		// n.b.: reusing idempotent clients and CRD!
		if err := c.firewalls.Ensure(ctx, rule); err != nil {
			fmt.Printf("err: %s", err.Error()) // hoist this up
			return err
		}
	} else {
		if err := c.firewalls.Delete(ctx, rule); err != nil {
			fmt.Printf("err: %s", err.Error())
			return err
		}
	}
	return nil
}

// Get returns a SQL server.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (sql.Server, error) {
	local, err := c.convert(obj)
	if err != nil {
		return sql.Server{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// Delete handles deletion of a SQL server.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	// Get SQL Server secret
	if err = (*c.kubeclient).Delete(ctx, targetSecret); client.IgnoreNotFound(err) != nil {
		return err
	}

	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		return err
	}
	err = future.WaitForCompletionRef(ctx, c.internal.Client)
	if err != nil {
		return err
	}
	return nil
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.SQLServer, remote sql.Server) {
	local.Status.ID = remote.ID
	local.Status.State = nil
	if remote.ServerProperties != nil {
		local.Status.State = remote.ServerProperties.State
	}
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.SQLServer, error) {
	local, ok := obj.(*azurev1alpha1.SQLServer)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
