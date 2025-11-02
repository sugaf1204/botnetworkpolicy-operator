package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNavigateField(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		path    string
		want    any
		wantErr bool
	}{
		{
			name: "nested object navigation",
			payload: map[string]any{
				"outer": map[string]any{
					"inner": []any{"10.0.0.0/24", "2001:db8::/32"},
				},
			},
			path:    "outer.inner",
			want:    []any{"10.0.0.0/24", "2001:db8::/32"},
			wantErr: false,
		},
		{
			name: "single level navigation",
			payload: map[string]any{
				"cidrs": []any{"10.0.0.0/24"},
			},
			path:    "cidrs",
			want:    []any{"10.0.0.0/24"},
			wantErr: false,
		},
		{
			name: "deep nested navigation",
			payload: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": []any{"10.0.0.0/24"},
					},
				},
			},
			path:    "level1.level2.level3",
			want:    []any{"10.0.0.0/24"},
			wantErr: false,
		},
		{
			name: "missing segment",
			payload: map[string]any{
				"outer": map[string]any{},
			},
			path:    "outer.missing",
			wantErr: true,
		},
		{
			name: "non-object segment",
			payload: map[string]any{
				"outer": "not an object",
			},
			path:    "outer.inner",
			wantErr: true,
		},
		{
			name: "empty path",
			payload: map[string]any{
				"cidrs": []any{"10.0.0.0/24"},
			},
			path:    "",
			want:    map[string]any{"cidrs": []any{"10.0.0.0/24"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := navigateField(tt.payload, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("navigateField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				switch expected := tt.want.(type) {
				case []any:
					gotArr, ok := got.([]any)
					if !ok {
						t.Errorf("navigateField() got type %T, want []any", got)
						return
					}
					if len(gotArr) != len(expected) {
						t.Errorf("navigateField() got len %d, want %d", len(gotArr), len(expected))
					}
				}
			}
		})
	}
}

func TestInterpretCIDRs(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    []string
		wantErr bool
	}{
		{
			name:    "array of strings",
			value:   []any{"10.0.0.0/24", "10.0.1.0/24"},
			want:    []string{"10.0.0.0/24", "10.0.1.0/24"},
			wantErr: false,
		},
		{
			name:    "single string",
			value:   "10.0.0.0/24",
			want:    []string{"10.0.0.0/24"},
			wantErr: false,
		},
		{
			name:    "array with non-string",
			value:   []any{"10.0.0.0/24", 5},
			wantErr: true,
		},
		{
			name:    "array with nil",
			value:   []any{"10.0.0.0/24", nil},
			wantErr: true,
		},
		{
			name:    "empty array",
			value:   []any{},
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "unsupported type - integer",
			value:   123,
			wantErr: true,
		},
		{
			name:    "unsupported type - map",
			value:   map[string]any{"key": "value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpretCIDRs(tt.value, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("interpretCIDRs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("interpretCIDRs() got %d CIDRs, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("interpretCIDRs() got[%d] = %v, want %v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestJSONEndpointProvider_Fetch(t *testing.T) {
	tests := []struct {
		name         string
		responseBody map[string]any
		statusCode   int
		fieldPath    string
		headers      http.Header
		wantErr      bool
		wantMinLen   int
	}{
		{
			name: "successful fetch with nested field",
			responseBody: map[string]any{
				"data": map[string]any{
					"cidrs": []any{"10.0.0.0/24", "192.168.0.0/16"},
				},
			},
			statusCode: http.StatusOK,
			fieldPath:  "data.cidrs",
			headers:    http.Header{},
			wantErr:    false,
			wantMinLen: 2,
		},
		{
			name: "successful fetch with custom headers",
			responseBody: map[string]any{
				"cidrs": []any{"10.0.0.0/24"},
			},
			statusCode: http.StatusOK,
			fieldPath:  "cidrs",
			headers: http.Header{
				"X-Custom-Header": []string{"test-value"},
			},
			wantErr:    false,
			wantMinLen: 1,
		},
		{
			name: "successful fetch with single string value",
			responseBody: map[string]any{
				"cidr": "10.0.0.0/24",
			},
			statusCode: http.StatusOK,
			fieldPath:  "cidr",
			headers:    http.Header{},
			wantErr:    false,
			wantMinLen: 1,
		},
		{
			name:         "HTTP error status",
			responseBody: map[string]any{},
			statusCode:   http.StatusNotFound,
			fieldPath:    "cidrs",
			headers:      http.Header{},
			wantErr:      true,
		},
		{
			name:         "invalid JSON response",
			responseBody: nil,
			statusCode:   http.StatusOK,
			fieldPath:    "cidrs",
			headers:      http.Header{},
			wantErr:      true,
		},
		{
			name: "missing field path",
			responseBody: map[string]any{
				"other": []any{"10.0.0.0/24"},
			},
			statusCode: http.StatusOK,
			fieldPath:  "missing",
			headers:    http.Header{},
			wantErr:    true,
		},
		{
			name: "empty CIDRs after sanitization",
			responseBody: map[string]any{
				"cidrs": []any{"", "  "},
			},
			statusCode: http.StatusOK,
			fieldPath:  "cidrs",
			headers:    http.Header{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify custom headers are sent
				for k := range tt.headers {
					if r.Header.Get(k) != tt.headers.Get(k) {
						t.Errorf("header %s = %v, want %v", k, r.Header.Get(k), tt.headers.Get(k))
					}
				}

				w.WriteHeader(tt.statusCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				} else {
					w.Write([]byte("invalid json"))
				}
			}))
			defer server.Close()

			provider := &jsonEndpointProvider{
				client:    server.Client(),
				url:       server.URL,
				fieldPath: tt.fieldPath,
				headers:   tt.headers,
			}

			got, err := provider.Fetch(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonEndpointProvider.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) < tt.wantMinLen {
				t.Errorf("jsonEndpointProvider.Fetch() got %d CIDRs, want at least %d", len(got), tt.wantMinLen)
			}
		})
	}
}

func TestJSONEndpointProvider_FetchWithSecretHeaders(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("secret-token-value"),
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify secret header is sent
		authHeader := r.Header.Get("Authorization")
		if authHeader != "secret-token-value" {
			t.Errorf("Authorization header = %v, want secret-token-value", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"cidrs": []any{"10.0.0.0/24"},
		})
	}))
	defer server.Close()

	provider := &jsonEndpointProvider{
		client:     server.Client(),
		kubeClient: kubeClient,
		namespace:  "default",
		url:        server.URL,
		fieldPath:  "cidrs",
		headers:    http.Header{},
		secretHeaders: []secretHeaderRef{
			{
				name: "Authorization",
				selector: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "token",
				},
			},
		},
	}

	got, err := provider.Fetch(context.Background())
	if err != nil {
		t.Fatalf("jsonEndpointProvider.Fetch() error = %v", err)
	}
	if len(got) < 1 {
		t.Errorf("jsonEndpointProvider.Fetch() got %d CIDRs, want at least 1", len(got))
	}
}

func TestJSONEndpointProvider_ResolveSecretHeader_Errors(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name       string
		secret     *corev1.Secret
		secretName string
		secretKey  string
		wantErr    bool
	}{
		{
			name: "secret not found",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"token": []byte("value"),
				},
			},
			secretName: "missing-secret",
			secretKey:  "token",
			wantErr:    true,
		},
		{
			name: "secret key not found",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"token": []byte("value"),
				},
			},
			secretName: "test-secret",
			secretKey:  "missing-key",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.secret).
				Build()

			provider := &jsonEndpointProvider{
				kubeClient: kubeClient,
				namespace:  "default",
			}

			ref := secretHeaderRef{
				name: "Authorization",
				selector: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tt.secretName,
					},
					Key: tt.secretKey,
				},
			}

			_, err := provider.resolveSecretHeader(context.Background(), ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveSecretHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONEndpointProvider_FetchContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when context is cancelled")
	}))
	defer server.Close()

	provider := &jsonEndpointProvider{
		client:    server.Client(),
		url:       server.URL,
		fieldPath: "cidrs",
		headers:   http.Header{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.Fetch(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
}
