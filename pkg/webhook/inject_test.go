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

func TestBuildPatch_ExistingVolumes(t *testing.T) {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "existing-vol"},
			},
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:latest", Command: []string{"app"}},
			},
		},
	}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	// Should append to existing volumes with /spec/volumes/-, not reset them
	if containsStr(s, `"path":"/spec/volumes"`) && !containsStr(s, `"path":"/spec/volumes/-"`) {
		t.Error("with existing volumes, patch should use /spec/volumes/- to append, not replace")
	}
	if !containsStr(s, "masker-pipe") {
		t.Error("masker-pipe volume must be added")
	}
}

func TestBuildPatch_ExistingInitContainers(t *testing.T) {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "migrations", Image: "migrate:latest", Command: []string{"migrate"}},
			},
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:latest", Command: []string{"app"}},
			},
		},
	}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	// logcloak-init must be appended with /spec/initContainers/-, not overwrite
	if containsStr(s, `"path":"/spec/initContainers"`) && !containsStr(s, `"path":"/spec/initContainers/-"`) {
		t.Error("with existing initContainers, patch must use /spec/initContainers/- to append")
	}
	if !containsStr(s, "logcloak-init") {
		t.Error("logcloak-init container must appear in patch")
	}
}

func TestBuildPatch_ContainerWithExistingVolumeMounts(t *testing.T) {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "app",
					Image:   "myapp:latest",
					Command: []string{"app"},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "config", MountPath: "/etc/config"},
					},
				},
			},
		},
	}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	// Must append mount with /spec/containers/0/volumeMounts/-, not replace
	if !containsStr(s, `containers/0/volumeMounts/-`) {
		t.Error("with existing mounts, patch must append with volumeMounts/-")
	}
}

func TestBuildPatch_NoCommandNoArgs(t *testing.T) {
	// Distroless/entrypoint-only image: no Command, no Args in the pod spec.
	// buildOriginalCmd falls back to exec "$0" "$@" which in JSON has escaped quotes.
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:latest"},
			},
		},
	}
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	// JSON escapes quotes, so check for the escaped forms of $0 and $@
	if !containsStr(s, `$0`) || !containsStr(s, `$@`) {
		t.Errorf("entrypoint-only container should use exec $0 $@ fallback, got patch: %s", s)
	}
}

func TestBuildPatch_DefaultContainerAnnotation(t *testing.T) {
	p := pod("test", []string{"app"}, nil)
	ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(ops)
	s := string(b)
	// JSON Pointer encoding replaces / with ~1, so the annotation key appears as:
	// kubectl.kubernetes.io~1default-container
	if !containsStr(s, "default-container") {
		t.Error("patch must set default-container annotation")
	}
	if !containsStr(s, `"value":"logcloak"`) {
		t.Error("default-container annotation value must be 'logcloak'")
	}
}

func TestBuildPatch_AllKnownProxiesSkipped(t *testing.T) {
	proxies := []string{"istio-proxy", "linkerd-proxy", "envoy", "envoy-sidecar",
		"kuma-sidecar", "consul-sidecar", "vault-agent", "config-reloader"}
	for _, proxy := range proxies {
		p := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "myapp:latest", Command: []string{"app"}},
					{Name: proxy, Image: "proxy:latest", Command: []string{proxy}},
				},
			},
		}
		ops, err := webhook.BuildPatch(p, nil, "ghcr.io/1mr0-tech/logcloak-sidecar:latest")
		if err != nil {
			t.Fatalf("proxy=%s: %v", proxy, err)
		}
		b, _ := json.Marshal(ops)
		if containsStr(string(b), "containers/1/command") {
			t.Errorf("proxy container %q must not have its entrypoint wrapped", proxy)
		}
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
