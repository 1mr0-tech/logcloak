package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/metrics"
	"github.com/1mr0-tech/logcloak/pkg/regex"
	"github.com/1mr0-tech/logcloak/pkg/rules"
)

// Handler serves the mutating admission webhook endpoint.
type Handler struct {
	Client       client.Client
	SidecarImage string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		metrics.WebhookErrors.WithLabelValues("read_body").Inc()
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var review admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &review); err != nil {
		metrics.WebhookErrors.WithLabelValues("parse_review").Inc()
		http.Error(w, "parse review", http.StatusBadRequest)
		return
	}

	review.Response = h.mutate(r.Context(), review.Request)
	review.Response.UID = review.Request.UID

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(review); err != nil {
		metrics.WebhookErrors.WithLabelValues("write_response").Inc()
	}
}

func (h *Handler) mutate(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		metrics.WebhookErrors.WithLabelValues("parse_pod").Inc()
		return deny("failed to parse pod spec")
	}

	if pod.Annotations["logcloak.io/skip"] == "true" {
		metrics.WebhookAdmissions.WithLabelValues("skipped").Inc()
		return allow(nil)
	}

	const regexPrefix = "logcloak.io/regex-"
	for k, v := range pod.Annotations {
		if len(k) > len(regexPrefix) && k[:len(regexPrefix)] == regexPrefix {
			if err := regex.Validate(v); err != nil {
				metrics.WebhookAdmissions.WithLabelValues("rejected").Inc()
				return deny(fmt.Sprintf("invalid regex in annotation %q: %v", k, err))
			}
		}
	}

	compiled, err := h.resolveRules(ctx, req.Namespace, pod)
	if err != nil {
		metrics.WebhookErrors.WithLabelValues("resolve_rules").Inc()
		return deny(fmt.Sprintf("resolve rules: %v", err))
	}

	patch, err := BuildPatch(pod, compiled, h.SidecarImage)
	if err != nil {
		metrics.WebhookErrors.WithLabelValues("build_patch").Inc()
		return deny(fmt.Sprintf("build patch: %v", err))
	}

	metrics.WebhookAdmissions.WithLabelValues("injected").Inc()
	return allow(patch)
}

func (h *Handler) resolveRules(ctx context.Context, namespace string, pod corev1.Pod) ([]masker.Rule, error) {
	var policyList rules.MaskingPolicyList
	if err := h.Client.List(ctx, &policyList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("list MaskingPolicies: %w", err)
	}

	podLabels := labels.Set(pod.Labels)
	var matching []rules.MaskingPolicy
	for _, p := range policyList.Items {
		if p.Spec.Selector == nil {
			matching = append(matching, p)
			continue
		}
		sel, err := metav1.LabelSelectorAsSelector(p.Spec.Selector)
		if err != nil {
			continue
		}
		if sel.Matches(podLabels) {
			matching = append(matching, p)
		}
	}

	annotationSpecs := rules.ParseAnnotations(pod.Annotations)
	return rules.Merge(matching, annotationSpecs)
}

func allow(patch []Op) *admissionv1.AdmissionResponse {
	resp := &admissionv1.AdmissionResponse{Allowed: true}
	if len(patch) > 0 {
		b, _ := json.Marshal(patch)
		pt := admissionv1.PatchTypeJSONPatch
		resp.Patch = b
		resp.PatchType = &pt
	}
	return resp
}

func deny(msg string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: false,
		Result:  &metav1.Status{Message: msg},
	}
}
