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

func containsStr(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
