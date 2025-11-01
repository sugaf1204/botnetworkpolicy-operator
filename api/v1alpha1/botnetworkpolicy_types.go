// +kubebuilder:object:generate=true
// +groupName=bot.networking.dev
package v1alpha1

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion defines the group version for the BotNetworkPolicy API.
	GroupVersion = schema.GroupVersion{Group: "bot.networking.dev", Version: "v1alpha1"}
	// SchemeBuilder registers the API types with a scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds this API to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&BotNetworkPolicy{},
		&BotNetworkPolicyList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

// BotNetworkPolicySpec defines the desired state of BotNetworkPolicy.
type BotNetworkPolicySpec struct {
	// PodSelector selects the pods to which the NetworkPolicy will apply. If omitted, it targets all pods in the namespace.
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`

	// NamespaceSelector optionally restricts target namespaces. Currently informational.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// PolicyTypes explicitly sets the policy types. If empty, they are derived from ingress/egress flags.
	// +optional
	PolicyTypes []networkingv1.PolicyType `json:"policyTypes,omitempty"`

	// Ingress controls whether ingress rules should be managed. Defaults to true.
	// +optional
	Ingress *bool `json:"ingress,omitempty"`

	// Egress controls whether egress rules should be managed.
	// +optional
	Egress *bool `json:"egress,omitempty"`

	// Providers declares the providers that should be consulted for IP ranges.
	Providers []ProviderSpec `json:"providers"`

	// CustomCIDRs adds additional CIDRs that should be included in the generated NetworkPolicy.
	// +optional
	CustomCIDRs []string `json:"customCidrs,omitempty"`

	// SyncPeriod defines how frequently the controller should refresh the provider data.
	// +optional
	SyncPeriod metav1.Duration `json:"syncPeriod,omitempty"`
}

// ProviderSpec describes a single provider.
type ProviderSpec struct {
	// Name identifies the provider type. Supported values: google, aws, github, configMap, jsonEndpoint.
	Name string `json:"name"`

	// ConfigMap configures the built-in config map provider.
	// +optional
	ConfigMap *ConfigMapProviderSpec `json:"configMap,omitempty"`

	// JSONEndpoint configures the JSON endpoint provider that extracts CIDRs from a JSON response body.
	// +optional
	JSONEndpoint *JSONEndpointProviderSpec `json:"jsonEndpoint,omitempty"`
}

// ConfigMapProviderSpec fetches CIDRs from a ConfigMap key.
type ConfigMapProviderSpec struct {
	// Name is the name of the ConfigMap.
	Name string `json:"name"`

	// Namespace is the namespace containing the ConfigMap. Defaults to the namespace of the BotNetworkPolicy.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key selects the data key within the ConfigMap that contains newline or comma-separated CIDRs.
	Key string `json:"key"`
}

// JSONEndpointProviderSpec fetches CIDRs from a JSON REST endpoint.
type JSONEndpointProviderSpec struct {
	// URL is the HTTP endpoint to query.
	URL string `json:"url"`

	// FieldPath selects the JSON path (dot-separated) that contains the CIDR list.
	FieldPath string `json:"fieldPath"`

	// Headers optionally adds headers to the HTTP request.
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// HeaderSecretRefs composes request headers from Kubernetes Secrets.
	// +optional
	HeaderSecretRefs []HTTPHeaderSecretRef `json:"headerSecretRefs,omitempty"`
}

// HTTPHeaderSecretRef configures an HTTP header sourced from a Secret key.
type HTTPHeaderSecretRef struct {
	// Name is the HTTP header name.
	Name string `json:"name"`

	// SecretKeyRef identifies the Secret key that contains the header value.
	SecretKeyRef corev1.SecretKeySelector `json:"secretKeyRef"`
}

// BotNetworkPolicyStatus defines the observed state of BotNetworkPolicy.
type BotNetworkPolicyStatus struct {
	// LastSyncTime records the last time the providers were synchronised.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// ProviderCount records how many providers were processed successfully.
	// +optional
	ProviderCount int `json:"providerCount,omitempty"`
}

// +kubebuilder:object:root=true

// BotNetworkPolicy is the Schema for the botnetworkpolicies API.
type BotNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BotNetworkPolicySpec   `json:"spec,omitempty"`
	Status BotNetworkPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BotNetworkPolicyList contains a list of BotNetworkPolicy.
type BotNetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BotNetworkPolicy `json:"items"`
}

// DeepCopyObject implements runtime.Object.
func (in *BotNetworkPolicy) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BotNetworkPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver.
func (in *BotNetworkPolicy) DeepCopyInto(out *BotNetworkPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BotNetworkPolicy.
func (in *BotNetworkPolicy) DeepCopy() *BotNetworkPolicy {
	if in == nil {
		return nil
	}
	out := new(BotNetworkPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver.
func (in *BotNetworkPolicySpec) DeepCopyInto(out *BotNetworkPolicySpec) {
	*out = *in
	if in.PodSelector != nil {
		out.PodSelector = new(metav1.LabelSelector)
		in.PodSelector.DeepCopyInto(out.PodSelector)
	}
	if in.NamespaceSelector != nil {
		out.NamespaceSelector = new(metav1.LabelSelector)
		in.NamespaceSelector.DeepCopyInto(out.NamespaceSelector)
	}
	if in.PolicyTypes != nil {
		out.PolicyTypes = append([]networkingv1.PolicyType{}, in.PolicyTypes...)
	}
	if in.Ingress != nil {
		out.Ingress = new(bool)
		*out.Ingress = *in.Ingress
	}
	if in.Egress != nil {
		out.Egress = new(bool)
		*out.Egress = *in.Egress
	}
	if in.Providers != nil {
		out.Providers = make([]ProviderSpec, len(in.Providers))
		for i := range in.Providers {
			in.Providers[i].DeepCopyInto(&out.Providers[i])
		}
	}
	if in.CustomCIDRs != nil {
		out.CustomCIDRs = append([]string{}, in.CustomCIDRs...)
	}
}

// DeepCopyInto copies the receiver.
func (in *ProviderSpec) DeepCopyInto(out *ProviderSpec) {
	*out = *in
	if in.ConfigMap != nil {
		out.ConfigMap = new(ConfigMapProviderSpec)
		*out.ConfigMap = *in.ConfigMap
	}
	if in.JSONEndpoint != nil {
		out.JSONEndpoint = new(JSONEndpointProviderSpec)
		in.JSONEndpoint.DeepCopyInto(out.JSONEndpoint)
	}
}

// DeepCopyInto copies the receiver.
func (in *JSONEndpointProviderSpec) DeepCopyInto(out *JSONEndpointProviderSpec) {
	*out = *in
	if in.Headers != nil {
		out.Headers = make(map[string]string, len(in.Headers))
		for k, v := range in.Headers {
			out.Headers[k] = v
		}
	}
	if in.HeaderSecretRefs != nil {
		out.HeaderSecretRefs = make([]HTTPHeaderSecretRef, len(in.HeaderSecretRefs))
		for i := range in.HeaderSecretRefs {
			in.HeaderSecretRefs[i].DeepCopyInto(&out.HeaderSecretRefs[i])
		}
	}
}

// DeepCopyInto copies the receiver.
func (in *HTTPHeaderSecretRef) DeepCopyInto(out *HTTPHeaderSecretRef) {
	*out = *in
	in.SecretKeyRef.DeepCopyInto(&out.SecretKeyRef)
}

// DeepCopyInto copies the receiver.
func (in *BotNetworkPolicyStatus) DeepCopyInto(out *BotNetworkPolicyStatus) {
	*out = *in
	if in.LastSyncTime != nil {
		out.LastSyncTime = in.LastSyncTime.DeepCopy()
	}
}

// DeepCopyObject implements runtime.Object.
func (in *BotNetworkPolicyList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BotNetworkPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver.
func (in *BotNetworkPolicyList) DeepCopyInto(out *BotNetworkPolicyList) {
	*out = *in
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]BotNetworkPolicy, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BotNetworkPolicyList.
func (in *BotNetworkPolicyList) DeepCopy() *BotNetworkPolicyList {
	if in == nil {
		return nil
	}
	out := new(BotNetworkPolicyList)
	in.DeepCopyInto(out)
	return out
}

// IngressEnabled returns true when ingress is enabled.
func (s *BotNetworkPolicySpec) IngressEnabled() bool {
	if s.Ingress == nil {
		return true
	}
	return *s.Ingress
}

// EgressEnabled returns true when egress is enabled.
func (s *BotNetworkPolicySpec) EgressEnabled() bool {
	if s.Egress == nil {
		return false
	}
	return *s.Egress
}

// NetworkPolicyName returns the derived NetworkPolicy name.
func (b *BotNetworkPolicy) NetworkPolicyName() string {
	if name := strings.TrimSpace(b.Annotations["bot.networking.dev/networkpolicy-name"]); name != "" {
		return name
	}
	return b.Name + "-allow-bots"
}

// Validate performs basic validation on provider spec.
func (p *ProviderSpec) Validate() error {
	switch strings.ToLower(p.Name) {
	case "google", "aws", "github":
		return nil
	case "configmap":
		if p.ConfigMap == nil {
			return fmt.Errorf("configMap provider requires configMap configuration")
		}
		if p.ConfigMap.Name == "" || p.ConfigMap.Key == "" {
			return fmt.Errorf("configMap provider requires name and key")
		}
		return nil
	case "jsonendpoint":
		if p.JSONEndpoint == nil {
			return fmt.Errorf("jsonEndpoint provider requires jsonEndpoint configuration")
		}
		if p.JSONEndpoint.URL == "" || p.JSONEndpoint.FieldPath == "" {
			return fmt.Errorf("jsonEndpoint provider requires url and fieldPath")
		}
		for _, headerRef := range p.JSONEndpoint.HeaderSecretRefs {
			if strings.TrimSpace(headerRef.Name) == "" {
				return fmt.Errorf("jsonEndpoint headerSecretRefs requires name")
			}
			if headerRef.SecretKeyRef.Name == "" || headerRef.SecretKeyRef.Key == "" {
				return fmt.Errorf("jsonEndpoint headerSecretRefs requires secret name and key")
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported provider: %s", p.Name)
	}
}

// Validate performs validation for the BotNetworkPolicy resource.
func (b *BotNetworkPolicy) Validate() error {
	for i := range b.Spec.Providers {
		if err := b.Spec.Providers[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ExtractCIDRs parses newline/comma separated values.
func ExtractCIDRs(payload string) []string {
	fields := strings.FieldsFunc(payload, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	results := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			results = append(results, trimmed)
		}
	}
	return results
}
