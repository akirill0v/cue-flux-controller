// Package v1alpha1 contains API Schema definitions for the cue v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=cue.contrib.flux.io
package v1alpha1

// ResourceInventory contains a list of Kubernetes resource object references that have been applied by a CueInstance.
type ResourceInventory struct {
	// Entries of Kubernetes resource object references.
	Entries []ResourceRef `json:"entries"`
}

// ResourceRef contains the information necessary to locate a resource within a cluster.
type ResourceRef struct {
	// ID is the string representation of the Kubernetes resource object's metadata,
	// in the format '<namespace>_<name>_<group>_<kind>'.
	ID string `json:"id"`

	// Version is the API version of the Kubernetes resource object's kind.
	Version string `json:"v"`
}
