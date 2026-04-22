package rules

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var GroupVersion = schema.GroupVersion{Group: "logcloak.io", Version: "v1alpha1"}

func AddToScheme(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &MaskingPolicy{}, &MaskingPolicyList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}

type MaskingPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MaskingPolicySpec `json:"spec"`
}

type MaskingPolicySpec struct {
	Selector   *metav1.LabelSelector `json:"selector,omitempty"`
	Patterns   []PatternSpec         `json:"patterns"`
	RedactWith string                `json:"redactWith,omitempty"`
}

type PatternSpec struct {
	Name    string `json:"name"`
	Builtin string `json:"builtin,omitempty"`
	Regex   string `json:"regex,omitempty"`
}

type MaskingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaskingPolicy `json:"items"`
}

func (in *MaskingPolicy) DeepCopyObject() runtime.Object {
	out := &MaskingPolicy{}
	*out = *in
	out.Spec.Patterns = make([]PatternSpec, len(in.Spec.Patterns))
	copy(out.Spec.Patterns, in.Spec.Patterns)
	if in.Spec.Selector != nil {
		sel := *in.Spec.Selector
		out.Spec.Selector = &sel
	}
	return out
}

func (in *MaskingPolicyList) DeepCopyObject() runtime.Object {
	out := &MaskingPolicyList{}
	*out = *in
	out.Items = make([]MaskingPolicy, len(in.Items))
	for i, item := range in.Items {
		out.Items[i] = *item.DeepCopyObject().(*MaskingPolicy)
	}
	return out
}
