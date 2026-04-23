package webhook_test

import (
	"encoding/json"
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/webhook"
)

func pod(name string, command, args []string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:latest", Command: command, Args: args},
			},
		},
	}
}

func TestBuildPatch_ContainsSidecar(t *testing.T) {
	p := pod("test", []string{"nginx"}, nil)
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	if !containsStr(string(b), "logcloak") {
		t.Error("patch should add logcloak sidecar container")
	}
}

func TestBuildPatch_ContainsVolumes(t *testing.T) {
	p := pod("test", []string{"app"}, nil)
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	if !containsStr(string(b), "masker-pipe") {
		t.Error("patch should add masker-pipe volume")
	}
}

func TestBuildPatch_WrapsEntrypoint(t *testing.T) {
	p := pod("test", []string{"java"}, []string{"-jar", "app.jar"})
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	if !containsStr(string(b), "masker-pipe/app.pipe") {
		t.Error("patch should redirect entrypoint to FIFO")
	}
}

func TestBuildPatch_SkipAnnotation(t *testing.T) {
	p := pod("test", []string{"app"}, nil)
	p.Annotations = map[string]string{"logcloak.io/skip": "true"}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 0 {
		t.Error("skip annotation should produce empty patch")
	}
}

func TestBuildPatch_WithRules(t *testing.T) {
	p := pod("test", []string{"app"}, nil)
	compiled := []masker.Rule{{
		Name:    "email",
		Pattern: regexp.MustCompile(`[a-z]+@[a-z]+\.[a-z]{2,}`),
		Replace: "[REDACTED]",
	}}
	ops, err := webhook.BuildPatch(p, compiled, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	if !containsStr(string(b), "LOGCLOAK_RULES") {
		t.Error("patch should inject LOGCLOAK_RULES env var")
	}
}

func TestBuildPatch_DoesNotWrapIstioProxy(t *testing.T) {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:latest", Command: []string{"app"}},
				{Name: "istio-proxy", Image: "istio/proxyv2:1.20", Command: []string{"/usr/local/bin/pilot-agent"}},
			},
		},
	}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	// app container must be wrapped
	if !containsStr(s, "masker-pipe/app.pipe") {
		t.Error("app container entrypoint should be redirected to FIFO")
	}
	// istio-proxy must NOT be wrapped (no replace op targeting containers/1)
	if containsStr(s, "containers/1/command") {
		t.Error("istio-proxy container should not have its entrypoint wrapped")
	}
}

func TestBuildPatch_ExcludeContainersAnnotation(t *testing.T) {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Annotations: map[string]string{
				"logcloak.io/exclude-containers": "custom-proxy, monitoring-agent",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:latest", Command: []string{"app"}},
				{Name: "custom-proxy", Image: "proxy:latest", Command: []string{"proxy"}},
				{Name: "monitoring-agent", Image: "agent:latest", Command: []string{"agent"}},
			},
		},
	}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	if !containsStr(s, "masker-pipe/app.pipe") {
		t.Error("app container should be wrapped")
	}
	// containers/1 and containers/2 (custom-proxy, monitoring-agent) must not have command replaced
	if containsStr(s, "containers/1/command") || containsStr(s, "containers/2/command") {
		t.Error("excluded containers should not have entrypoints wrapped")
	}
}

func containsStr(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
