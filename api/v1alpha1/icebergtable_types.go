// api/v1alpha1/icebergtable_types.go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

type PartitionField struct {
	Name      string `json:"name"`
	Transform string `json:"transform"`
}

type CatalogSpec struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Warehouse string `json:"warehouse,omitempty"`
}

type RetentionPolicy struct {
	SnapshotExpirationDays int `json:"snapshotExpirationDays"`
}

type IcebergTableSpec struct {
	DataProduct     string            `json:"dataProduct"`
	Catalog         CatalogSpec       `json:"catalog"`
	Database        string            `json:"database"`
	Table           string            `json:"table"`
	Schema          []Column          `json:"schema"`
	PartitionSpec   []PartitionField  `json:"partitionSpec,omitempty"`
	Properties      map[string]string `json:"properties,omitempty"`
	RetentionPolicy RetentionPolicy   `json:"retentionPolicy"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type IcebergTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IcebergTableSpec   `json:"spec,omitempty"`
	Status            IcebergTableStatus `json:"status,omitempty"`
}

type IcebergTableStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
type IcebergTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IcebergTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IcebergTable{}, &IcebergTableList{})
}
