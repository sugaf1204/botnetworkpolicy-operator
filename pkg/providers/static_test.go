package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleSelector(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantLen int
		wantErr bool
	}{
		{
			name: "valid google response with IPv4 and IPv6",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ipv4Prefix": "8.8.8.0/24"},
					map[string]any{"ipv6Prefix": "2001:4860::/32"},
					map[string]any{"ipv4Prefix": "8.8.4.0/24"},
				},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "google response with empty prefix",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ipv4Prefix": "8.8.8.0/24"},
					map[string]any{"ipv4Prefix": "  "},
					map[string]any{"ipv6Prefix": "2001:4860::/32"},
				},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "google response with invalid prefix type",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ipv4Prefix": "8.8.8.0/24"},
					"invalid",
					nil,
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "missing prefixes field",
			data:    map[string]any{},
			wantErr: true,
		},
		{
			name: "invalid prefixes type",
			data: map[string]any{
				"prefixes": "not an array",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := googleSelector(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("googleSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("googleSelector() got %d CIDRs, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestAWSSelector(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantLen int
		wantErr bool
	}{
		{
			name: "valid AWS response with GLOBAL and us-east-1",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ip_prefix": "52.94.76.0/24", "service": "AMAZON", "region": "GLOBAL", "network_border_group": "GLOBAL"},
					map[string]any{"ip_prefix": "54.239.0.0/16", "service": "AMAZON", "region": "us-east-1", "network_border_group": "us-east-1"},
					map[string]any{"ip_prefix": "52.119.224.0/20", "service": "AMAZON_CONNECT", "region": "GLOBAL", "network_border_group": "GLOBAL"},
				},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "filter out non-AMAZON services",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ip_prefix": "52.94.76.0/24", "service": "AMAZON", "region": "GLOBAL"},
					map[string]any{"ip_prefix": "54.239.0.0/16", "service": "EC2", "region": "us-east-1"},
					map[string]any{"ip_prefix": "52.119.224.0/20", "service": "S3", "region": "GLOBAL"},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "filter out non-GLOBAL and non-us-east-1 regions",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ip_prefix": "52.94.76.0/24", "service": "AMAZON", "region": "GLOBAL"},
					map[string]any{"ip_prefix": "54.239.0.0/16", "service": "AMAZON", "region": "us-west-2"},
					map[string]any{"ip_prefix": "13.248.0.0/16", "service": "AMAZON", "region": "eu-west-1"},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "handle empty and whitespace prefixes",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ip_prefix": "52.94.76.0/24", "service": "AMAZON", "region": "GLOBAL"},
					map[string]any{"ip_prefix": "  ", "service": "AMAZON", "region": "us-east-1"},
					map[string]any{"ip_prefix": "", "service": "AMAZON_CONNECT", "region": "GLOBAL"},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "with network_border_group field",
			data: map[string]any{
				"prefixes": []any{
					map[string]any{"ip_prefix": "52.94.76.0/24", "service": "AMAZON", "region": "GLOBAL", "network_border_group": "GLOBAL"},
					map[string]any{"ip_prefix": "54.239.0.0/16", "service": "AMAZON", "region": "us-east-1", "network_border_group": "us-east-1-mia-1"},
					map[string]any{"ip_prefix": "3.5.0.0/18", "service": "AMAZON", "region": "us-east-1", "network_border_group": "us-east-1"},
				},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "missing prefixes field",
			data:    map[string]any{},
			wantErr: true,
		},
		{
			name: "invalid prefixes type",
			data: map[string]any{
				"prefixes": "not an array",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := awsSelector(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("awsSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("awsSelector() got %d CIDRs, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestGitHubSelector(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantLen int
		wantErr bool
	}{
		{
			name: "valid GitHub response",
			data: map[string]any{
				"hooks": []any{
					"192.30.252.0/22",
					"185.199.108.0/22",
					"140.82.112.0/20",
				},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "GitHub response with empty strings",
			data: map[string]any{
				"hooks": []any{
					"192.30.252.0/22",
					"  ",
					"",
					"185.199.108.0/22",
				},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "GitHub response with invalid types",
			data: map[string]any{
				"hooks": []any{
					"192.30.252.0/22",
					123,
					nil,
					"185.199.108.0/22",
				},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "missing hooks field",
			data:    map[string]any{},
			wantErr: true,
		},
		{
			name: "invalid hooks type",
			data: map[string]any{
				"hooks": "not an array",
			},
			wantErr: true,
		},
		{
			name: "empty hooks array",
			data: map[string]any{
				"hooks": []any{},
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubSelector(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("githubSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("githubSelector() got %d CIDRs, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestStaticHTTPProvider_Fetch(t *testing.T) {
	tests := []struct {
		name         string
		responseBody map[string]any
		statusCode   int
		selector     func(map[string]any) ([]string, error)
		wantErr      bool
		wantMinLen   int
	}{
		{
			name: "successful fetch with Google selector",
			responseBody: map[string]any{
				"prefixes": []any{
					map[string]any{"ipv4Prefix": "8.8.8.0/24"},
					map[string]any{"ipv6Prefix": "2001:4860::/32"},
				},
			},
			statusCode: http.StatusOK,
			selector:   googleSelector,
			wantErr:    false,
			wantMinLen: 2,
		},
		{
			name: "successful fetch with AWS selector",
			responseBody: map[string]any{
				"prefixes": []any{
					map[string]any{"ip_prefix": "52.94.76.0/24", "service": "AMAZON", "region": "GLOBAL"},
					map[string]any{"ip_prefix": "54.239.0.0/16", "service": "AMAZON", "region": "us-east-1"},
				},
			},
			statusCode: http.StatusOK,
			selector:   awsSelector,
			wantErr:    false,
			wantMinLen: 2,
		},
		{
			name: "successful fetch with GitHub selector",
			responseBody: map[string]any{
				"hooks": []any{
					"192.30.252.0/22",
					"185.199.108.0/22",
				},
			},
			statusCode: http.StatusOK,
			selector:   githubSelector,
			wantErr:    false,
			wantMinLen: 2,
		},
		{
			name:         "HTTP error status",
			responseBody: map[string]any{},
			statusCode:   http.StatusInternalServerError,
			selector:     googleSelector,
			wantErr:      true,
		},
		{
			name:         "invalid JSON response",
			responseBody: nil,
			statusCode:   http.StatusOK,
			selector:     googleSelector,
			wantErr:      true,
		},
		{
			name: "selector returns empty CIDRs",
			responseBody: map[string]any{
				"prefixes": []any{},
			},
			statusCode: http.StatusOK,
			selector:   googleSelector,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				} else {
					w.Write([]byte("invalid json"))
				}
			}))
			defer server.Close()

			provider := &staticHTTPProvider{
				client:   server.Client(),
				url:      server.URL,
				selector: tt.selector,
			}

			got, err := provider.Fetch(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("staticHTTPProvider.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) < tt.wantMinLen {
				t.Errorf("staticHTTPProvider.Fetch() got %d CIDRs, want at least %d", len(got), tt.wantMinLen)
			}
		})
	}
}

func TestStaticHTTPProvider_FetchContextCancellation(t *testing.T) {
	// Create mock server that checks for context cancellation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler won't be reached because context is already cancelled
		t.Error("handler should not be called when context is cancelled")
	}))
	defer server.Close()

	provider := &staticHTTPProvider{
		client:   server.Client(),
		url:      server.URL,
		selector: googleSelector,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.Fetch(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
}
