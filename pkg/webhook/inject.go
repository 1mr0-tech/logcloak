package webhook

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/rules"
)

const (
	sidecarName  = "logcloak"
	pipePath     = "/masker-pipe/app.pipe"
	pipeVolume   = "masker-pipe"
	pipeMountDir = "/masker-pipe"
)

// BuildPatch returns JSON Patch ops that mutate a pod to inject the masker sidecar.
// Returns an empty slice (no-op) if the pod has logcloak.io/skip=true.
func BuildPatch(pod corev1.Pod, compiled []masker.Rule, sidecarImage string) ([]Op, error) {
	if pod.Annotations["logcloak.io/skip"] == "true" {
		return nil, nil
	}

	rulesJSON, err := rules.Serialize(compiled)
	if err != nil {
		return nil, fmt.Errorf("serialize rules: %w", err)
	}

	var ops []Op

	if pod.Annotations == nil {
		ops = append(ops, addOp("/metadata/annotations", map[string]string{}))
	}

	ops = append(ops, addOp(
		"/metadata/annotations/"+escapePath("kubectl.kubernetes.io/default-container"),
		sidecarName,
	))

	if len(pod.Spec.Volumes) == 0 {
		ops = append(ops, addOp("/spec/volumes", []corev1.Volume{}))
	}
	ops = append(ops, addOp("/spec/volumes/-", corev1.Volume{
		Name: pipeVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumMemory,
				SizeLimit: resourcePtr("8Mi"),
			},
		},
	}))

	initCtr := corev1.Container{
		Name:    "logcloak-init",
		Image:   "busybox:1.36",
		Command: []string{"mkfifo", pipePath},
		VolumeMounts: []corev1.VolumeMount{
			{Name: pipeVolume, MountPath: pipeMountDir},
		},
		SecurityContext: &corev1.SecurityContext{
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			RunAsNonRoot:             boolPtr(true),
			RunAsUser:                int64Ptr(65534),
		},
	}
	if len(pod.Spec.InitContainers) == 0 {
		ops = append(ops, addOp("/spec/initContainers", []corev1.Container{initCtr}))
	} else {
		ops = append(ops, addOp("/spec/initContainers/-", initCtr))
	}

	for i, c := range pod.Spec.Containers {
		if c.Name == sidecarName {
			continue
		}
		ops = append(ops, wrapEntrypoint(i, c)...)
		if len(c.VolumeMounts) == 0 {
			ops = append(ops, replaceOp(
				fmt.Sprintf("/spec/containers/%d/volumeMounts", i),
				[]corev1.VolumeMount{{Name: pipeVolume, MountPath: pipeMountDir}},
			))
		} else {
			ops = append(ops, addOp(
				fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
				corev1.VolumeMount{Name: pipeVolume, MountPath: pipeMountDir},
			))
		}
	}

	ops = append(ops, addOp("/spec/containers/-", sidecarContainer(sidecarImage, rulesJSON)))

	return ops, nil
}

func wrapEntrypoint(idx int, c corev1.Container) []Op {
	original := buildOriginalCmd(c)
	wrapped := fmt.Sprintf("%s >%s 2>&1", original, pipePath)
	return []Op{
		replaceOp(fmt.Sprintf("/spec/containers/%d/command", idx), []string{"sh", "-c"}),
		replaceOp(fmt.Sprintf("/spec/containers/%d/args", idx), []string{wrapped}),
	}
}

func buildOriginalCmd(c corev1.Container) string {
	if len(c.Command) == 0 && len(c.Args) == 0 {
		return `exec "$0" "$@"`
	}
	parts := make([]string, 0, len(c.Command)+len(c.Args))
	for _, p := range append(c.Command, c.Args...) {
		parts = append(parts, shellQuote(p))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t\n\"'\\$`") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func sidecarContainer(image, rulesJSON string) corev1.Container {
	return corev1.Container{
		Name:  sidecarName,
		Image: image,
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
				},
			},
			{Name: "LOGCLOAK_RULES", Value: rulesJSON},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("5m"),
				corev1.ResourceMemory: resource.MustParse("20Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: pipeVolume, MountPath: pipeMountDir},
		},
		SecurityContext: &corev1.SecurityContext{
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			RunAsNonRoot:             boolPtr(true),
			RunAsUser:                int64Ptr(65534),
		},
	}
}

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }
func resourcePtr(s string) *resource.Quantity {
	q := resource.MustParse(s)
	return &q
}
