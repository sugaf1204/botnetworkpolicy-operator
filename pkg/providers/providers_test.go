package providers

import (
	"net/http"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/sugaf1204/botnetworkpolicy-operator/api/v1alpha1"
)

func TestFactory_NewFactory(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	httpClient := &http.Client{}

	factory := NewFactory(kubeClient, httpClient)

	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}
	if factory.kubeClient == nil {
		t.Error("factory.kubeClient is nil")
	}
	if factory.httpClient == nil {
		t.Error("factory.httpClient is nil")
	}
	if factory.googleEndpoint != defaultGoogleEndpoint {
		t.Errorf("factory.googleEndpoint = %v, want %v", factory.googleEndpoint, defaultGoogleEndpoint)
	}
	if factory.awsEndpoint != defaultAWSEndpoint {
		t.Errorf("factory.awsEndpoint = %v, want %v", factory.awsEndpoint, defaultAWSEndpoint)
	}
	if factory.githubEndpoint != defaultGitHubEndpoint {
		t.Errorf("factory.githubEndpoint = %v, want %v", factory.githubEndpoint, defaultGitHubEndpoint)
	}
}

func TestFactory_WithOptions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	httpClient := &http.Client{}

	customGoogle := "https://custom.google.endpoint/test"
	customAWS := "https://custom.aws.endpoint/test"
	customGitHub := "https://custom.github.endpoint/test"

	factory := NewFactory(
		kubeClient,
		httpClient,
		WithGoogleEndpoint(customGoogle),
		WithAWSEndpoint(customAWS),
		WithGitHubEndpoint(customGitHub),
	)

	if factory.googleEndpoint != customGoogle {
		t.Errorf("factory.googleEndpoint = %v, want %v", factory.googleEndpoint, customGoogle)
	}
	if factory.awsEndpoint != customAWS {
		t.Errorf("factory.awsEndpoint = %v, want %v", factory.awsEndpoint, customAWS)
	}
	if factory.githubEndpoint != customGitHub {
		t.Errorf("factory.githubEndpoint = %v, want %v", factory.githubEndpoint, customGitHub)
	}
}

func TestFactory_WithEmptyOptions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	httpClient := &http.Client{}

	// Empty strings should not override defaults
	factory := NewFactory(
		kubeClient,
		httpClient,
		WithGoogleEndpoint(""),
		WithAWSEndpoint("  "),
		WithGitHubEndpoint("\t\n"),
	)

	if factory.googleEndpoint != defaultGoogleEndpoint {
		t.Errorf("factory.googleEndpoint = %v, want %v (empty option should not override)", factory.googleEndpoint, defaultGoogleEndpoint)
	}
	if factory.awsEndpoint != defaultAWSEndpoint {
		t.Errorf("factory.awsEndpoint = %v, want %v (empty option should not override)", factory.awsEndpoint, defaultAWSEndpoint)
	}
	if factory.githubEndpoint != defaultGitHubEndpoint {
		t.Errorf("factory.githubEndpoint = %v, want %v (empty option should not override)", factory.githubEndpoint, defaultGitHubEndpoint)
	}
}

func TestFactory_FromSpec_Google(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "google",
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	if provider == nil {
		t.Fatal("FromSpec() returned nil provider")
	}

	_, ok := provider.(*staticHTTPProvider)
	if !ok {
		t.Errorf("FromSpec() returned type %T, want *staticHTTPProvider", provider)
	}
}

func TestFactory_FromSpec_AWS(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "aws",
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	if provider == nil {
		t.Fatal("FromSpec() returned nil provider")
	}

	_, ok := provider.(*staticHTTPProvider)
	if !ok {
		t.Errorf("FromSpec() returned type %T, want *staticHTTPProvider", provider)
	}
}

func TestFactory_FromSpec_GitHub(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "github",
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	if provider == nil {
		t.Fatal("FromSpec() returned nil provider")
	}

	_, ok := provider.(*staticHTTPProvider)
	if !ok {
		t.Errorf("FromSpec() returned type %T, want *staticHTTPProvider", provider)
	}
}

func TestFactory_FromSpec_ConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "configmap",
		ConfigMap: &v1alpha1.ConfigMapProviderSpec{
			Name: "test-config",
			Key:  "cidrs",
		},
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	if provider == nil {
		t.Fatal("FromSpec() returned nil provider")
	}

	p, ok := provider.(*configMapProvider)
	if !ok {
		t.Fatalf("FromSpec() returned type %T, want *configMapProvider", provider)
	}

	if p.namespace != "default" {
		t.Errorf("configMapProvider.namespace = %v, want default", p.namespace)
	}
	if p.name != "test-config" {
		t.Errorf("configMapProvider.name = %v, want test-config", p.name)
	}
	if p.key != "cidrs" {
		t.Errorf("configMapProvider.key = %v, want cidrs", p.key)
	}
}

func TestFactory_FromSpec_ConfigMap_CustomNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "configmap",
		ConfigMap: &v1alpha1.ConfigMapProviderSpec{
			Name:      "test-config",
			Namespace: "custom-namespace",
			Key:       "cidrs",
		},
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	p, ok := provider.(*configMapProvider)
	if !ok {
		t.Fatalf("FromSpec() returned type %T, want *configMapProvider", provider)
	}

	if p.namespace != "custom-namespace" {
		t.Errorf("configMapProvider.namespace = %v, want custom-namespace", p.namespace)
	}
}

func TestFactory_FromSpec_JSONEndpoint(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "jsonendpoint",
		JSONEndpoint: &v1alpha1.JSONEndpointProviderSpec{
			URL:       "https://example.com/api/cidrs",
			FieldPath: "data.cidrs",
			Headers: map[string]string{
				"X-API-Key": "test-key",
			},
		},
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	if provider == nil {
		t.Fatal("FromSpec() returned nil provider")
	}

	p, ok := provider.(*jsonEndpointProvider)
	if !ok {
		t.Fatalf("FromSpec() returned type %T, want *jsonEndpointProvider", provider)
	}

	if p.url != "https://example.com/api/cidrs" {
		t.Errorf("jsonEndpointProvider.url = %v, want https://example.com/api/cidrs", p.url)
	}
	if p.fieldPath != "data.cidrs" {
		t.Errorf("jsonEndpointProvider.fieldPath = %v, want data.cidrs", p.fieldPath)
	}
	if p.headers.Get("X-API-Key") != "test-key" {
		t.Errorf("jsonEndpointProvider.headers['X-API-Key'] = %v, want test-key", p.headers.Get("X-API-Key"))
	}
}

func TestFactory_FromSpec_JSONEndpoint_WithSecretHeaders(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "jsonendpoint",
		JSONEndpoint: &v1alpha1.JSONEndpointProviderSpec{
			URL:       "https://example.com/api/cidrs",
			FieldPath: "data.cidrs",
			HeaderSecretRefs: []v1alpha1.HTTPHeaderSecretRef{
				{
					Name: "Authorization",
					SecretKeyRef: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "api-secret",
						},
						Key: "token",
					},
				},
			},
		},
	}

	provider, err := factory.FromSpec("default", spec)
	if err != nil {
		t.Fatalf("FromSpec() error = %v", err)
	}

	p, ok := provider.(*jsonEndpointProvider)
	if !ok {
		t.Fatalf("FromSpec() returned type %T, want *jsonEndpointProvider", provider)
	}

	if len(p.secretHeaders) != 1 {
		t.Errorf("jsonEndpointProvider.secretHeaders length = %d, want 1", len(p.secretHeaders))
	}
	if p.secretHeaders[0].name != "Authorization" {
		t.Errorf("jsonEndpointProvider.secretHeaders[0].name = %v, want Authorization", p.secretHeaders[0].name)
	}
	if p.secretHeaders[0].selector.Name != "api-secret" {
		t.Errorf("jsonEndpointProvider.secretHeaders[0].selector.Name = %v, want api-secret", p.secretHeaders[0].selector.Name)
	}
}

func TestFactory_FromSpec_UnsupportedProvider(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	spec := v1alpha1.ProviderSpec{
		Name: "unsupported-provider",
	}

	_, err := factory.FromSpec("default", spec)
	if err == nil {
		t.Error("FromSpec() expected error for unsupported provider, got nil")
	}
}

func TestFactory_FromSpec_CaseInsensitive(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	factory := NewFactory(kubeClient, &http.Client{})

	tests := []struct {
		name         string
		providerName string
		wantType     string
	}{
		{
			name:         "Google uppercase",
			providerName: "GOOGLE",
			wantType:     "*providers.staticHTTPProvider",
		},
		{
			name:         "aws lowercase",
			providerName: "aws",
			wantType:     "*providers.staticHTTPProvider",
		},
		{
			name:         "GitHub mixed case",
			providerName: "GitHuB",
			wantType:     "*providers.staticHTTPProvider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := v1alpha1.ProviderSpec{
				Name: tt.providerName,
			}

			provider, err := factory.FromSpec("default", spec)
			if err != nil {
				t.Fatalf("FromSpec() error = %v", err)
			}

			if provider == nil {
				t.Fatal("FromSpec() returned nil provider")
			}

			_, ok := provider.(*staticHTTPProvider)
			if !ok {
				t.Errorf("FromSpec() returned type %T, want *staticHTTPProvider", provider)
			}
		})
	}
}
