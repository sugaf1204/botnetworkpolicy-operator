package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type jsonEndpointProvider struct {
	client        *http.Client
	kubeClient    client.Reader
	namespace     string
	url           string
	fieldPath     string
	headers       http.Header
	secretHeaders []secretHeaderRef
	filter        *jsonFilter
}

type jsonFilter struct {
	fieldConditions []fieldCondition
}

type fieldCondition struct {
	field  string
	values []string
}

func (p *jsonEndpointProvider) Fetch(ctx context.Context) ([]string, error) {
	headers, err := p.resolveHeaders(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, err
	}
	for k, values := range headers {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	value, err := navigateField(payload, p.fieldPath)
	if err != nil {
		return nil, err
	}

	cidrs, err := interpretCIDRs(value, p.filter)
	if err != nil {
		return nil, err
	}
	return sanitize(cidrs)
}

func (p *jsonEndpointProvider) resolveHeaders(ctx context.Context) (http.Header, error) {
	headers := http.Header{}
	for k, values := range p.headers {
		for _, value := range values {
			headers.Add(k, value)
		}
	}

	for _, secretHeader := range p.secretHeaders {
		value, err := p.resolveSecretHeader(ctx, secretHeader)
		if err != nil {
			return nil, err
		}
		headers.Add(secretHeader.name, value)
	}

	return headers, nil
}

func (p *jsonEndpointProvider) resolveSecretHeader(ctx context.Context, ref secretHeaderRef) (string, error) {
	if p.kubeClient == nil {
		return "", fmt.Errorf("kube client not configured for secret-backed headers")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Name: ref.selector.Name, Namespace: p.namespace}
	if err := p.kubeClient.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("fetching secret %s: %w", key.String(), err)
	}

	data, ok := secret.Data[ref.selector.Key]
	if !ok {
		return "", fmt.Errorf("secret %s missing key %s", key.String(), ref.selector.Key)
	}
	return string(data), nil
}

type secretHeaderRef struct {
	name     string
	selector corev1.SecretKeySelector
}

func navigateField(input any, path string) (any, error) {
	current := input
	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			continue
		}
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("segment %q not an object", segment)
		}
		next, ok := m[segment]
		if !ok {
			return nil, fmt.Errorf("missing segment %q", segment)
		}
		current = next
	}
	return current, nil
}

func interpretCIDRs(value any, filter *jsonFilter) ([]string, error) {
	switch v := value.(type) {
	case []any:
		cidrs := make([]string, 0, len(v))
		for _, item := range v {
			// If item is a string, use it directly
			if str, ok := item.(string); ok {
				cidrs = append(cidrs, str)
				continue
			}

			// If item is an object and we have a filter, apply filtering
			if obj, ok := item.(map[string]any); ok {
				if filter != nil && !matchesFilter(obj, filter) {
					continue
				}

				// Try to extract CIDR from common field names
				cidr := extractCIDRFromObject(obj)
				if cidr != "" {
					cidrs = append(cidrs, cidr)
				}
				continue
			}

			return nil, fmt.Errorf("array value %v is neither a string nor an object", item)
		}
		return cidrs, nil
	case string:
		return []string{v}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON field type %T", value)
	}
}

func matchesFilter(obj map[string]any, filter *jsonFilter) bool {
	// All field conditions must match
	for _, condition := range filter.fieldConditions {
		fieldValue, ok := obj[condition.field]
		if !ok {
			return false
		}

		fieldStr, ok := fieldValue.(string)
		if !ok {
			return false
		}

		// If no specific values specified, just check field exists
		if len(condition.values) == 0 {
			continue
		}

		// Check if field value matches any of the allowed values
		matched := false
		for _, allowedValue := range condition.values {
			if strings.EqualFold(strings.TrimSpace(fieldStr), strings.TrimSpace(allowedValue)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func extractCIDRFromObject(obj map[string]any) string {
	// Try common CIDR field names
	commonFields := []string{"ip_prefix", "ipv4Prefix", "ipv6Prefix", "cidr", "ipPrefix", "ip"}
	for _, field := range commonFields {
		if value, ok := obj[field].(string); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}
