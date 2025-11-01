package controllers

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	botv1alpha1 "github.com/sugaf1204/botnetworkpolicy-operator/api/v1alpha1"
)

func TestBuildNetworkPolicy(t *testing.T) {
	ingress := true
	resource := &botv1alpha1.BotNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample",
			Namespace: "default",
		},
		Spec: botv1alpha1.BotNetworkPolicySpec{
			PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Ingress:     &ingress,
		},
	}

	cidrs := []string{"10.0.0.0/24"}
	np := buildNetworkPolicy(resource, cidrs)

	if np.Name != "sample-allow-bots" {
		t.Fatalf("unexpected name: %s", np.Name)
	}
	if len(np.Spec.Ingress) != 1 || len(np.Spec.Ingress[0].From) != 1 {
		t.Fatalf("unexpected ingress configuration: %#v", np.Spec.Ingress)
	}
	if np.Spec.Ingress[0].From[0].IPBlock == nil || np.Spec.Ingress[0].From[0].IPBlock.CIDR != "10.0.0.0/24" {
		t.Fatalf("missing IP block")
	}
	if len(np.Spec.PolicyTypes) == 0 || np.Spec.PolicyTypes[0] != networkingv1.PolicyTypeIngress {
		t.Fatalf("unexpected policy types: %#v", np.Spec.PolicyTypes)
	}
}
