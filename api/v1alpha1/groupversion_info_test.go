package v1alpha1

import (
	"testing"
)

func TestGroupVersion(t *testing.T) {
	expectedGroup := "external-haproxy-operator.ullberg.us"
	expectedVersion := "v1alpha1"

	if GroupVersion.Group != expectedGroup {
		t.Errorf("GroupVersion.Group = %q, want %q", GroupVersion.Group, expectedGroup)
	}
	if GroupVersion.Version != expectedVersion {
		t.Errorf("GroupVersion.Version = %q, want %q", GroupVersion.Version, expectedVersion)
	}
}

func TestSchemeBuilderGroupVersion(t *testing.T) {
	if SchemeBuilder.GroupVersion != GroupVersion {
		t.Errorf("SchemeBuilder.GroupVersion = %v, want %v", SchemeBuilder.GroupVersion, GroupVersion)
	}
}

func TestAddToSchemeIsNotNil(t *testing.T) {
	if AddToScheme == nil {
		t.Error("AddToScheme is nil, want non-nil function")
	}
}
