package providers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConfigMapProvider_Fetch(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name       string
		configMap  *corev1.ConfigMap
		key        string
		wantErr    bool
		wantMinLen int
	}{
		{
			name: "successful fetch with single CIDR",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"cidrs": "10.0.0.0/24",
				},
			},
			key:        "cidrs",
			wantErr:    false,
			wantMinLen: 1,
		},
		{
			name: "successful fetch with multiple CIDRs",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"cidrs": "10.0.0.0/24\n192.168.0.0/16\n172.16.0.0/12",
				},
			},
			key:        "cidrs",
			wantErr:    false,
			wantMinLen: 3,
		},
		{
			name: "successful fetch with comma-separated CIDRs",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"cidrs": "10.0.0.0/24,192.168.0.0/16,172.16.0.0/12",
				},
			},
			key:        "cidrs",
			wantErr:    false,
			wantMinLen: 3,
		},
		{
			name: "successful fetch with whitespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"cidrs": "  10.0.0.0/24  \n  192.168.0.0/16  ",
				},
			},
			key:        "cidrs",
			wantErr:    false,
			wantMinLen: 2,
		},
		{
			name: "missing key in configmap",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"other-key": "10.0.0.0/24",
				},
			},
			key:     "cidrs",
			wantErr: true,
		},
		{
			name: "empty data after sanitization",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"cidrs": "   \n   \n   ",
				},
			},
			key:     "cidrs",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.configMap).
				Build()

			provider := &configMapProvider{
				client:    kubeClient,
				namespace: tt.configMap.Namespace,
				name:      tt.configMap.Name,
				key:       tt.key,
			}

			got, err := provider.Fetch(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("configMapProvider.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) < tt.wantMinLen {
				t.Errorf("configMapProvider.Fetch() got %d CIDRs, want at least %d", len(got), tt.wantMinLen)
			}
		})
	}
}

func TestConfigMapProvider_FetchConfigMapNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create empty client without any configmaps
	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	provider := &configMapProvider{
		client:    kubeClient,
		namespace: "default",
		name:      "missing-config",
		key:       "cidrs",
	}

	_, err := provider.Fetch(context.Background())
	if err == nil {
		t.Error("expected error when configmap not found, got nil")
	}
}

func TestConfigMapProvider_FetchContextCancellation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"cidrs": "10.0.0.0/24",
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	provider := &configMapProvider{
		client:    kubeClient,
		namespace: "default",
		name:      "test-config",
		key:       "cidrs",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Note: The fake client doesn't actually respect context cancellation,
	// so this test verifies the API accepts a cancelled context without panicking
	_, err := provider.Fetch(ctx)
	// Either succeeds or fails with cancellation error, both are acceptable
	_ = err
}

func TestErrMissingKey(t *testing.T) {
	err := errMissingKey("test-key")
	expected := "configmap missing key: test-key"
	if err.Error() != expected {
		t.Errorf("errMissingKey.Error() = %v, want %v", err.Error(), expected)
	}
}
