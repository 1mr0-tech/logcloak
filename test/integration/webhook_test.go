package integration_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/1mr0-tech/logcloak/pkg/rules"
)

// TestWebhook_BasicInjection verifies that a plain pod in the injected
// namespace receives the logcloak sidecar, init container, and FIFO volume.
func TestWebhook_BasicInjection(t *testing.T) {
	mustCreateNamespace(t, "inject-basic")
	h := newHandler()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "basic", Namespace: "inject-basic"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36", Command: []string{"sh"}},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if !resp.Allowed {
		t.Fatalf("expected allowed, got denied: %v", resp.Result)
	}
	if resp.Patch == nil {
		t.Fatal("expected non-empty patch")
	}
	patch := string(resp.Patch)
	assertContains(t, patch, "logcloak", "sidecar container")
	assertContains(t, patch, "masker-pipe", "FIFO volume")
	assertContains(t, patch, "logcloak-init", "init container")
}

// TestWebhook_SkipAnnotation verifies that logcloak.io/skip=true produces
// an empty patch (no injection).
func TestWebhook_SkipAnnotation(t *testing.T) {
	mustCreateNamespace(t, "inject-skip")
	h := newHandler()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "skip-me",
			Namespace:   "inject-skip",
			Annotations: map[string]string{"logcloak.io/skip": "true"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36"},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if !resp.Allowed {
		t.Fatalf("expected allowed, got denied: %v", resp.Result)
	}
	if resp.Patch != nil {
		t.Errorf("skip annotation should produce nil patch, got: %s", resp.Patch)
	}
}

// TestWebhook_InvalidRegexRejected verifies that a pod with an unsafe regex
// annotation is denied at admission time.
func TestWebhook_InvalidRegexRejected(t *testing.T) {
	mustCreateNamespace(t, "inject-regex")
	h := newHandler()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bad-regex",
			Namespace: "inject-regex",
			Annotations: map[string]string{
				"logcloak.io/regex-bad": "(?=lookahead)",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36"},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if resp.Allowed {
		t.Error("pod with lookahead regex should be denied")
	}
	if resp.Result == nil || !strings.Contains(resp.Result.Message, "unsafe") {
		t.Errorf("denial message should mention 'unsafe', got: %v", resp.Result)
	}
}

// TestWebhook_MaskingPolicyMerged verifies that patterns from a MaskingPolicy
// in the namespace are serialised into the sidecar's LOGCLOAK_RULES env var.
func TestWebhook_MaskingPolicyMerged(t *testing.T) {
	mustCreateNamespace(t, "inject-policy")
	h := newHandler()

	policy := &rules.MaskingPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "logcloak.io/v1alpha1",
			Kind:       "MaskingPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "baseline",
			Namespace: "inject-policy",
		},
		Spec: rules.MaskingPolicySpec{
			Patterns: []rules.PatternSpec{
				{Name: "email", Builtin: "email"},
				{Name: "otp", Builtin: "otp-6digit"},
			},
			RedactWith: "[REDACTED]",
		},
	}
	if err := k8sClient.Create(context.Background(), policy); err != nil {
		t.Fatalf("create MaskingPolicy: %v", err)
	}
	t.Cleanup(func() {
		_ = k8sClient.Delete(context.Background(), policy)
	})

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "inject-policy"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36", Command: []string{"sh"}},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if !resp.Allowed {
		t.Fatalf("expected allowed, got denied: %v", resp.Result)
	}
	patch := string(resp.Patch)
	// LOGCLOAK_RULES env var must be present and contain the email pattern
	assertContains(t, patch, "LOGCLOAK_RULES", "rules env var")
	assertContains(t, patch, "email", "email rule from MaskingPolicy")
}

// TestWebhook_MultiContainerOnlyWrapsApp verifies that when multiple containers
// are present, only the non-excluded ones get their entrypoints redirected.
func TestWebhook_MultiContainerOnlyWrapsApp(t *testing.T) {
	mustCreateNamespace(t, "inject-multi")
	h := newHandler()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi",
			Namespace: "inject-multi",
			Annotations: map[string]string{
				"logcloak.io/exclude-containers": "sidecar-db",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:latest", Command: []string{"app"}},
				{Name: "sidecar-db", Image: "postgres:16", Command: []string{"postgres"}},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if !resp.Allowed {
		t.Fatalf("expected allowed: %v", resp.Result)
	}
	patch := string(resp.Patch)
	assertContains(t, patch, "containers/0/command", "app container wrapped")
	if strings.Contains(patch, "containers/1/command") {
		t.Error("excluded sidecar-db must not have its entrypoint wrapped")
	}
}

// TestWebhook_MaskingPolicySelectorFiltering verifies that a MaskingPolicy
// with a label selector only applies to matching pods.
func TestWebhook_MaskingPolicySelectorFiltering(t *testing.T) {
	mustCreateNamespace(t, "inject-selector")
	h := newHandler()

	policy := &rules.MaskingPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "logcloak.io/v1alpha1",
			Kind:       "MaskingPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "targeted",
			Namespace: "inject-selector",
		},
		Spec: rules.MaskingPolicySpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "payment-service"},
			},
			Patterns: []rules.PatternSpec{
				{Name: "credit-card", Builtin: "credit-card"},
			},
			RedactWith: "[REDACTED]",
		},
	}
	if err := k8sClient.Create(context.Background(), policy); err != nil {
		t.Fatalf("create MaskingPolicy: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), policy) })

	// Pod that does NOT match the selector — LOGCLOAK_RULES should be empty or null
	unmatchedPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unmatched",
			Namespace: "inject-selector",
			Labels:    map[string]string{"app": "other-service"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36", Command: []string{"sh"}},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(unmatchedPod))
	if !resp.Allowed {
		t.Fatalf("expected allowed: %v", resp.Result)
	}
	unmatchedPatch := string(resp.Patch)
	if strings.Contains(unmatchedPatch, "credit-card") {
		t.Error("unmatched pod should not receive credit-card rule from targeted MaskingPolicy")
	}

	// Pod that DOES match the selector
	matchedPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "matched",
			Namespace: "inject-selector",
			Labels:    map[string]string{"app": "payment-service"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36", Command: []string{"sh"}},
			},
		},
	}
	resp2 := h.MutateForTest(context.Background(), buildReview(matchedPod))
	if !resp2.Allowed {
		t.Fatalf("expected allowed: %v", resp2.Result)
	}
	matchedPatch := string(resp2.Patch)
	assertContains(t, matchedPatch, "credit-card", "matched pod receives credit-card rule")
}

// TestWebhook_AnnotationExtendsPolicy verifies that pod annotations add rules
// on top of MaskingPolicy rules without replacing them.
func TestWebhook_AnnotationExtendsPolicy(t *testing.T) {
	mustCreateNamespace(t, "inject-extend")
	h := newHandler()

	policy := &rules.MaskingPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "logcloak.io/v1alpha1",
			Kind:       "MaskingPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "base",
			Namespace: "inject-extend",
		},
		Spec: rules.MaskingPolicySpec{
			Patterns: []rules.PatternSpec{
				{Name: "email", Builtin: "email"},
			},
		},
	}
	if err := k8sClient.Create(context.Background(), policy); err != nil {
		t.Fatalf("create MaskingPolicy: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), policy) })

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "extended",
			Namespace: "inject-extend",
			Annotations: map[string]string{
				"logcloak.io/regex-order-id": `ORD-[0-9]{8}`,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36", Command: []string{"sh"}},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if !resp.Allowed {
		t.Fatalf("expected allowed: %v", resp.Result)
	}
	patch := string(resp.Patch)

	// Deserialise LOGCLOAK_RULES from the patch to verify both rules are present
	rulesJSON := extractLogcloakRules(t, patch)
	var serialised []rules.SerializedRule
	if err := json.Unmarshal([]byte(rulesJSON), &serialised); err != nil {
		t.Fatalf("parse LOGCLOAK_RULES: %v", err)
	}

	names := make(map[string]bool)
	for _, r := range serialised {
		names[r.Name] = true
	}
	if !names["email"] {
		t.Error("email rule from MaskingPolicy must be present after annotation extension")
	}
	if !names["order-id"] {
		t.Error("order-id rule from annotation must be present")
	}
}

// TestWebhook_MaskingPolicyDeleted verifies that after a MaskingPolicy is
// deleted, a new pod receives no rules from it.
func TestWebhook_MaskingPolicyDeleted(t *testing.T) {
	mustCreateNamespace(t, "inject-delete")
	h := newHandler()

	policy := &rules.MaskingPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "logcloak.io/v1alpha1",
			Kind:       "MaskingPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transient",
			Namespace: "inject-delete",
		},
		Spec: rules.MaskingPolicySpec{
			Patterns: []rules.PatternSpec{
				{Name: "jwt", Builtin: "jwt"},
			},
		},
	}
	if err := k8sClient.Create(context.Background(), policy); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := k8sClient.Delete(context.Background(), policy, &client.DeleteOptions{}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "after-delete", Namespace: "inject-delete"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "busybox:1.36", Command: []string{"sh"}},
			},
		},
	}
	resp := h.MutateForTest(context.Background(), buildReview(pod))
	if !resp.Allowed {
		t.Fatalf("expected allowed: %v", resp.Result)
	}
	if strings.Contains(string(resp.Patch), "jwt") {
		t.Error("deleted MaskingPolicy must not contribute rules to new pods")
	}
}

// extractLogcloakRules pulls the LOGCLOAK_RULES value out of a JSON Patch string.
func extractLogcloakRules(t *testing.T, patch string) string {
	t.Helper()
	type op struct {
		Op    string          `json:"op"`
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	var ops []op
	if err := json.Unmarshal([]byte(patch), &ops); err != nil {
		t.Fatalf("parse patch: %v", err)
	}
	for _, o := range ops {
		// The sidecar container is added as a single object to /spec/containers/-
		if o.Path != "/spec/containers/-" {
			continue
		}
		var ctr corev1.Container
		if err := json.Unmarshal(o.Value, &ctr); err != nil {
			continue
		}
		for _, env := range ctr.Env {
			if env.Name == "LOGCLOAK_RULES" {
				return env.Value
			}
		}
	}
	t.Fatal("LOGCLOAK_RULES not found in patch")
	return ""
}

func assertContains(t *testing.T, s, needle, desc string) {
	t.Helper()
	if !strings.Contains(s, needle) {
		t.Errorf("expected %s to contain %q\ngot: %s", desc, needle, s)
	}
}
