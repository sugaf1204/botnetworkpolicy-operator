package controllers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	botv1alpha1 "github.com/sugaf1204/botnetworkpolicy-operator/api/v1alpha1"
	"github.com/sugaf1204/botnetworkpolicy-operator/pkg/providers"
)

// BotNetworkPolicyReconciler reconciles a BotNetworkPolicy object
//+kubebuilder:rbac:groups=bot.networking.dev,resources=botnetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bot.networking.dev,resources=botnetworkpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bot.networking.dev,resources=botnetworkpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

type BotNetworkPolicyReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	HTTPClient *http.Client
}

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func (r *BotNetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var resource botv1alpha1.BotNetworkPolicy
	if err := r.Get(ctx, req.NamespacedName, &resource); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := resource.Validate(); err != nil {
		logger.Error(err, "invalid specification")
		r.Recorder.Event(&resource, corev1.EventTypeWarning, "InvalidSpec", err.Error())
		return ctrl.Result{}, nil
	}

	cidrs, warnings, err := r.collectCIDRs(ctx, &resource, logger)
	if err != nil {
		logger.Error(err, "failed to collect CIDRs")
		return ctrl.Result{}, err
	}

	for _, warning := range warnings {
		r.Recorder.Event(&resource, corev1.EventTypeWarning, "ProviderWarning", warning)
	}

	if err := r.ensureNetworkPolicy(ctx, &resource, cidrs, logger); err != nil {
		logger.Error(err, "failed to ensure network policy")
		return ctrl.Result{}, err
	}

	syncAfter := resource.Spec.SyncPeriod.Duration
	if syncAfter == 0 {
		syncAfter = providers.DefaultSyncPeriod
	}

	logger.Info("reconciliation complete", "requeueAfter", syncAfter)
	return ctrl.Result{RequeueAfter: syncAfter}, nil
}

func (r *BotNetworkPolicyReconciler) ensureNetworkPolicy(ctx context.Context, resource *botv1alpha1.BotNetworkPolicy, cidrs []string, logger logr.Logger) error {
	desired := buildNetworkPolicy(resource, cidrs)

	var existing networkingv1.NetworkPolicy
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, &existing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(resource, desired, r.Scheme); err != nil {
			return err
		}
		logger.Info("creating networkpolicy", "name", desired.Name)
		return r.Create(ctx, desired)
	}

	if metav1.IsControlledBy(&existing, resource) {
		if networkPoliciesEqual(&existing, desired) {
			return nil
		}
		existing.Spec = desired.Spec
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations
		logger.Info("updating networkpolicy", "name", desired.Name)
		return r.Update(ctx, &existing)
	}

	return fmt.Errorf("networkpolicy %s/%s exists and is not controlled by BotNetworkPolicy", desired.Namespace, desired.Name)
}

func buildNetworkPolicy(resource *botv1alpha1.BotNetworkPolicy, cidrs []string) *networkingv1.NetworkPolicy {
	labels := map[string]string{
		"botnetworkpolicy.bot.networking.dev/owner": resource.Name,
	}

	podSelector := metav1.LabelSelector{}
	if resource.Spec.PodSelector != nil {
		podSelector = *resource.Spec.PodSelector
	}

	policyTypes := determinePolicyTypes(resource.Spec.PolicyTypes, resource.Spec.Ingress, resource.Spec.Egress)

	ingressRules := []networkingv1.NetworkPolicyIngressRule{}
	egressRules := []networkingv1.NetworkPolicyEgressRule{}

	if len(cidrs) > 0 {
		peers := make([]networkingv1.NetworkPolicyPeer, 0, len(cidrs))
		for _, cidr := range cidrs {
			peers = append(peers, networkingv1.NetworkPolicyPeer{IPBlock: &networkingv1.IPBlock{CIDR: cidr}})
		}
		if resource.Spec.IngressEnabled() {
			ingressRules = append(ingressRules, networkingv1.NetworkPolicyIngressRule{From: peers})
		}
		if resource.Spec.EgressEnabled() {
			egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{To: peers})
		}
	}

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.NetworkPolicyName(),
			Namespace: resource.Namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: podSelector,
			PolicyTypes: policyTypes,
			Ingress:     ingressRules,
			Egress:      egressRules,
		},
	}
}

func networkPoliciesEqual(existing *networkingv1.NetworkPolicy, desired *networkingv1.NetworkPolicy) bool {
	if len(existing.Spec.PolicyTypes) != len(desired.Spec.PolicyTypes) {
		return false
	}
	existingTypes := append([]networkingv1.PolicyType{}, existing.Spec.PolicyTypes...)
	desiredTypes := append([]networkingv1.PolicyType{}, desired.Spec.PolicyTypes...)
	sort.Slice(existingTypes, func(i, j int) bool { return existingTypes[i] < existingTypes[j] })
	sort.Slice(desiredTypes, func(i, j int) bool { return desiredTypes[i] < desiredTypes[j] })
	for i := range existingTypes {
		if existingTypes[i] != desiredTypes[i] {
			return false
		}
	}
	if !networkPolicyIngressEqual(existing.Spec.Ingress, desired.Spec.Ingress) {
		return false
	}
	if !networkPolicyEgressEqual(existing.Spec.Egress, desired.Spec.Egress) {
		return false
	}
	if !selectorsEqual(existing.Spec.PodSelector, desired.Spec.PodSelector) {
		return false
	}
	return true
}

func selectorsEqual(a, b metav1.LabelSelector) bool {
	if len(a.MatchLabels) != len(b.MatchLabels) {
		return false
	}
	for k, v := range a.MatchLabels {
		if b.MatchLabels[k] != v {
			return false
		}
	}
	if len(a.MatchExpressions) != len(b.MatchExpressions) {
		return false
	}
	for i := range a.MatchExpressions {
		ae := a.MatchExpressions[i]
		be := b.MatchExpressions[i]
		if ae.Key != be.Key || ae.Operator != be.Operator || !equalStringSlices(ae.Values, be.Values) {
			return false
		}
	}
	return true
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func networkPolicyIngressEqual(a, b []networkingv1.NetworkPolicyIngressRule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !networkPolicyPeersEqual(a[i].From, b[i].From) {
			return false
		}
	}
	return true
}

func networkPolicyEgressEqual(a, b []networkingv1.NetworkPolicyEgressRule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !networkPolicyPeersEqual(a[i].To, b[i].To) {
			return false
		}
	}
	return true
}

func networkPolicyPeersEqual(a, b []networkingv1.NetworkPolicyPeer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i].IPBlock == nil) != (b[i].IPBlock == nil) {
			return false
		}
		if a[i].IPBlock != nil {
			if a[i].IPBlock.CIDR != b[i].IPBlock.CIDR || !equalStringSlices(a[i].IPBlock.Except, b[i].IPBlock.Except) {
				return false
			}
		}
	}
	return true
}

func determinePolicyTypes(requested []networkingv1.PolicyType, ingress, egress *bool) []networkingv1.PolicyType {
	if len(requested) > 0 {
		return append([]networkingv1.PolicyType{}, requested...)
	}
	enabled := sets.New[networkingv1.PolicyType]()
	if ingress == nil || *ingress {
		enabled.Insert(networkingv1.PolicyTypeIngress)
	}
	if egress != nil && *egress {
		enabled.Insert(networkingv1.PolicyTypeEgress)
	}
	if enabled.Len() == 0 {
		enabled.Insert(networkingv1.PolicyTypeIngress)
	}
	return sets.List(enabled)
}

func (r *BotNetworkPolicyReconciler) collectCIDRs(ctx context.Context, resource *botv1alpha1.BotNetworkPolicy, logger logr.Logger) ([]string, []string, error) {
	factory := providers.NewFactory(r.Client, r.HTTPClient)

	providerCIDRs := sets.NewString()
	warnings := make([]string, 0)

	for _, providerSpec := range resource.Spec.Providers {
		provider, err := factory.FromSpec(resource.Namespace, providerSpec)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("provider %s skipped: %v", providerSpec.Name, err))
			continue
		}

		cidrs, err := provider.Fetch(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("provider %s fetch error: %v", providerSpec.Name, err))
			continue
		}

		for _, cidr := range cidrs {
			normalized := strings.TrimSpace(cidr)
			if normalized == "" {
				continue
			}
			providerCIDRs.Insert(normalized)
		}
	}

	for _, cidr := range resource.Spec.CustomCIDRs {
		providerCIDRs.Insert(strings.TrimSpace(cidr))
	}

	result := providerCIDRs.List()
	sort.Strings(result)
	logger.Info("collected CIDRs", "count", len(result))
	return result, warnings, nil
}

func (r *BotNetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&botv1alpha1.BotNetworkPolicy{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
