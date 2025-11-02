package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type staticHTTPProvider struct {
	client   *http.Client
	url      string
	selector func(map[string]any) ([]string, error)
}

func (p *staticHTTPProvider) Fetch(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	cidrs, err := p.selector(payload)
	if err != nil {
		return nil, err
	}
	return sanitize(cidrs)
}

const (
	defaultGoogleEndpoint = "https://www.gstatic.com/ipranges/goog.json"
	defaultAWSEndpoint    = "https://ip-ranges.amazonaws.com/ip-ranges.json"
	defaultGitHubEndpoint = "https://api.github.com/meta"
)

func googleSelectorWithScope(data map[string]any, scopes []string) ([]string, error) {
	prefixesRaw, ok := data["prefixes"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing prefixes")
	}

	// Convert scopes to a map for efficient lookup
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[strings.ToLower(strings.TrimSpace(scope))] = true
	}
	filterByScope := len(scopeMap) > 0

	results := make([]string, 0)
	for _, prefix := range prefixesRaw {
		item, _ := prefix.(map[string]any)
		if item == nil {
			continue
		}

		// Apply scope filter if specified
		if filterByScope {
			scope, _ := item["scope"].(string)
			if !scopeMap[strings.ToLower(strings.TrimSpace(scope))] {
				continue
			}
		}

		if ipv4, ok := item["ipv4Prefix"].(string); ok {
			if value := strings.TrimSpace(ipv4); value != "" {
				results = append(results, value)
			}
		}
		if ipv6, ok := item["ipv6Prefix"].(string); ok {
			if value := strings.TrimSpace(ipv6); value != "" {
				results = append(results, value)
			}
		}
	}
	return results, nil
}

func googleSelector(data map[string]any) ([]string, error) {
	return googleSelectorWithScope(data, nil)
}

func awsSelectorWithFilter(data map[string]any, services, regions, networkBorderGroups []string) ([]string, error) {
	prefixesRaw, ok := data["prefixes"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing prefixes")
	}

	// Convert filters to maps for efficient lookup
	serviceMap := make(map[string]bool)
	for _, svc := range services {
		serviceMap[strings.ToUpper(strings.TrimSpace(svc))] = true
	}
	filterByService := len(serviceMap) > 0

	regionMap := make(map[string]bool)
	for _, reg := range regions {
		regionMap[strings.ToLower(strings.TrimSpace(reg))] = true
	}
	filterByRegion := len(regionMap) > 0

	nbgMap := make(map[string]bool)
	for _, nbg := range networkBorderGroups {
		nbgMap[strings.ToLower(strings.TrimSpace(nbg))] = true
	}
	filterByNBG := len(nbgMap) > 0

	results := make([]string, 0)
	for _, prefix := range prefixesRaw {
		item, _ := prefix.(map[string]any)
		if item == nil {
			continue
		}

		// Apply service filter
		if filterByService {
			service, _ := item["service"].(string)
			if !serviceMap[strings.ToUpper(strings.TrimSpace(service))] {
				continue
			}
		}

		// Apply region filter
		if filterByRegion {
			region, _ := item["region"].(string)
			if !regionMap[strings.ToLower(strings.TrimSpace(region))] {
				continue
			}
		}

		// Apply network border group filter
		if filterByNBG {
			nbg, _ := item["network_border_group"].(string)
			if !nbgMap[strings.ToLower(strings.TrimSpace(nbg))] {
				continue
			}
		}

		if cidr, ok := item["ip_prefix"].(string); ok {
			if value := strings.TrimSpace(cidr); value != "" {
				results = append(results, value)
			}
		}
	}
	return results, nil
}

func awsSelector(data map[string]any) ([]string, error) {
	defaultServices := []string{"AMAZON", "AMAZON_CONNECT"}
	defaultRegions := []string{"GLOBAL", "us-east-1"}
	return awsSelectorWithFilter(data, defaultServices, defaultRegions, nil)
}

func githubSelectorWithRoles(data map[string]any, roles []string) ([]string, error) {
	// If no roles specified, default to hooks
	if len(roles) == 0 {
		roles = []string{"hooks"}
	}

	results := make([]string, 0)
	for _, role := range roles {
		roleKey := strings.ToLower(strings.TrimSpace(role))
		roleData, ok := data[roleKey].([]any)
		if !ok {
			// Role field doesn't exist or is not an array, skip
			continue
		}

		for _, item := range roleData {
			if cidr, ok := item.(string); ok {
				if value := strings.TrimSpace(cidr); value != "" {
					results = append(results, value)
				}
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no CIDRs found for roles: %v", roles)
	}
	return results, nil
}

func githubSelector(data map[string]any) ([]string, error) {
	hooks, ok := data["hooks"]
	emptyHooksArray := false
	if ok {
		if arr, ok := hooks.([]any); ok {
			if len(arr) == 0 {
				emptyHooksArray = true
			}
		}
	}
	results, err := githubSelectorWithRoles(data, nil)
	if err != nil {
		if emptyHooksArray && strings.Contains(err.Error(), "no CIDRs found") {
			return []string{}, nil
		}
		return nil, err
	}
	return results, nil
}
