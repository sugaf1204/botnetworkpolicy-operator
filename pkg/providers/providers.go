package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/sugaf1204/botnetworkpolicy-operator/api/v1alpha1"
)

const (
	// DefaultSyncPeriod determines how frequently providers are re-polled if not specified on the resource.
	DefaultSyncPeriod = 1 * time.Hour
)

// Provider represents a fetcher that returns CIDR blocks.
type Provider interface {
	Fetch(ctx context.Context) ([]string, error)
}

// Factory constructs providers from CRD specs.
type Factory struct {
	kubeClient     client.Reader
	httpClient     *http.Client
	googleEndpoint string
	awsEndpoint    string
	githubEndpoint string
}

// NewFactory returns a provider factory.
func NewFactory(kubeClient client.Reader, httpClient *http.Client, opts ...FactoryOption) *Factory {
	factory := &Factory{
		kubeClient:     kubeClient,
		httpClient:     httpClient,
		googleEndpoint: defaultGoogleEndpoint,
		awsEndpoint:    defaultAWSEndpoint,
		githubEndpoint: defaultGitHubEndpoint,
	}
	for _, opt := range opts {
		opt(factory)
	}
	return factory
}

// FactoryOption mutates Factory construction parameters.
type FactoryOption func(*Factory)

// WithGoogleEndpoint overrides the Google provider endpoint.
func WithGoogleEndpoint(endpoint string) FactoryOption {
	return func(f *Factory) {
		if strings.TrimSpace(endpoint) != "" {
			f.googleEndpoint = endpoint
		}
	}
}

// WithAWSEndpoint overrides the AWS provider endpoint.
func WithAWSEndpoint(endpoint string) FactoryOption {
	return func(f *Factory) {
		if strings.TrimSpace(endpoint) != "" {
			f.awsEndpoint = endpoint
		}
	}
}

// WithGitHubEndpoint overrides the GitHub provider endpoint.
func WithGitHubEndpoint(endpoint string) FactoryOption {
	return func(f *Factory) {
		if strings.TrimSpace(endpoint) != "" {
			f.githubEndpoint = endpoint
		}
	}
}

// FromSpec constructs a Provider from the given specification.
func (f *Factory) FromSpec(namespace string, spec v1alpha1.ProviderSpec) (Provider, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	switch strings.ToLower(spec.Name) {
	case "google":
		url := f.googleEndpoint
		var scopes []string
		if spec.Google != nil {
			if spec.Google.URL != "" {
				url = spec.Google.URL
			}
			scopes = spec.Google.Scope
		}
		selector := func(data map[string]any) ([]string, error) {
			return googleSelectorWithScope(data, scopes)
		}
		return &staticHTTPProvider{client: f.httpClient, url: url, selector: selector}, nil

	case "aws":
		url := f.awsEndpoint
		var services, regions, nbgs []string

		// When spec.AWS is provided, respect the API contract:
		// - Empty services = all services
		// - Empty regions = all regions
		// - Empty NBGs = all NBGs
		if spec.AWS != nil {
			if spec.AWS.URL != "" {
				url = spec.AWS.URL
			}
			services = spec.AWS.Services
			regions = spec.AWS.Regions
			nbgs = spec.AWS.NetworkBorderGroups
		}
		// If spec.AWS is nil (name: aws only), all fields are empty = all IPs

		selector := func(data map[string]any) ([]string, error) {
			return awsSelectorWithFilter(data, services, regions, nbgs)
		}
		return &staticHTTPProvider{client: f.httpClient, url: url, selector: selector}, nil

	case "github":
		url := f.githubEndpoint
		var roles []string
		if spec.GitHub != nil {
			if spec.GitHub.URL != "" {
				url = spec.GitHub.URL
			}
			roles = spec.GitHub.Roles
		}
		selector := func(data map[string]any) ([]string, error) {
			return githubSelectorWithRoles(data, roles)
		}
		return &staticHTTPProvider{client: f.httpClient, url: url, selector: selector}, nil

	case "configmap":
		cfg := spec.ConfigMap
		ns := cfg.Namespace
		if ns == "" {
			ns = namespace
		}
		return &configMapProvider{client: f.kubeClient, namespace: ns, name: cfg.Name, key: cfg.Key}, nil

	case "jsonendpoint":
		cfg := spec.JSONEndpoint
		headers := http.Header{}
		for k, v := range cfg.Headers {
			headers.Set(k, v)
		}
		secretHeaders := make([]secretHeaderRef, 0, len(cfg.HeaderSecretRefs))
		for _, ref := range cfg.HeaderSecretRefs {
			secretHeaders = append(secretHeaders, secretHeaderRef{name: ref.Name, selector: ref.SecretKeyRef})
		}

		var filter *jsonFilter
		if cfg.Filter != nil && len(cfg.Filter.FieldConditions) > 0 {
			filter = &jsonFilter{
				fieldConditions: make([]fieldCondition, len(cfg.Filter.FieldConditions)),
			}
			for i, fc := range cfg.Filter.FieldConditions {
				filter.fieldConditions[i] = fieldCondition{
					field:  fc.Field,
					values: fc.Values,
				}
			}
		}

		return &jsonEndpointProvider{
			client:        f.httpClient,
			kubeClient:    f.kubeClient,
			namespace:     namespace,
			url:           cfg.URL,
			fieldPath:     cfg.FieldPath,
			headers:       headers,
			secretHeaders: secretHeaders,
			filter:        filter,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", spec.Name)
	}
}

// sanitize ensures CIDRs are trimmed and non-empty.
func sanitize(cidrs []string) ([]string, error) {
	results := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		trimmed := strings.TrimSpace(cidr)
		if trimmed == "" {
			continue
		}
		results = append(results, trimmed)
	}
	if len(results) == 0 {
		return nil, errors.New("provider returned no CIDRs")
	}
	return results, nil
}
