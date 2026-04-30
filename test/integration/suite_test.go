// Package integration runs envtest-based tests that exercise the mutating
// admission webhook against a real (fake) Kubernetes API server.
// Run with: KUBEBUILDER_ASSETS=/tmp/envtest-bins/k8s/1.30.3-darwin-arm64 go test ./test/integration/... -v
package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/1mr0-tech/logcloak/pkg/rules"
	"github.com/1mr0-tech/logcloak/pkg/webhook"
)

var (
	testEnv   *envtest.Environment
	k8sClient client.Client
)

func TestMain(m *testing.M) {
	assets := os.Getenv("KUBEBUILDER_ASSETS")
	if assets == "" {
		fmt.Fprintln(os.Stderr, "KUBEBUILDER_ASSETS not set — skipping integration tests")
		fmt.Fprintln(os.Stderr, "Run: setup-envtest use 1.30 --bin-dir /tmp/envtest-bins && export KUBEBUILDER_ASSETS=$(setup-envtest use 1.30 --bin-dir /tmp/envtest-bins -p path)")
		os.Exit(0)
	}

	crdPath := filepath.Join("..", "..", "charts", "logcloak", "crds")
	testEnv = &envtest.Environment{
		BinaryAssetsDirectory: assets,
		CRDDirectoryPaths:     []string{crdPath},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "start envtest: %v\n", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = rules.AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "logcloak.io", Version: "v1alpha1"},
		&rules.MaskingPolicy{}, &rules.MaskingPolicyList{})

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Fprintf(os.Stderr, "new client: %v\n", err)
		testEnv.Stop() //nolint:errcheck
		os.Exit(1)
	}

	code := m.Run()
	testEnv.Stop() //nolint:errcheck
	os.Exit(code)
}

// newHandler returns a webhook.Handler backed by the envtest API server.
func newHandler() *webhook.Handler {
	return &webhook.Handler{Client: k8sClient, SidecarImage: "logcloak-sidecar:dev"}
}

// buildReview wraps a pod into an AdmissionReview request.
func buildReview(pod corev1.Pod) admissionv1.AdmissionRequest {
	raw, _ := json.Marshal(pod)
	return admissionv1.AdmissionRequest{
		UID:       "test-uid",
		Namespace: pod.Namespace,
		Object:    runtime.RawExtension{Raw: raw},
	}
}

// mustCreateNamespace creates a namespace; if it already exists that is fine.
func mustCreateNamespace(t *testing.T, name string) {
	t.Helper()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := k8sClient.Create(context.Background(), ns); err != nil {
		if client.IgnoreNotFound(err) != nil {
			t.Fatalf("create namespace %s: %v", name, err)
		}
	}
}
