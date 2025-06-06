/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/abhilashkp/table-definition-operator/api/v1alpha1"
	tablesv1alpha1 "github.com/abhilashkp/table-definition-operator/api/v1alpha1"
)

// IcebergTableReconciler reconciles a IcebergTable object
type IcebergTableReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tables.example.com,resources=icebergtables,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tables.example.com,resources=icebergtables/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tables.example.com,resources=icebergtables/finalizers,verbs=update

// provisionIcebergTable makes an HTTP POST call to provision the Iceberg table
// It reads the submitted custom resource and generates JSON payload based on it
func (r *IcebergTableReconciler) provisionIcebergTable(ctx context.Context, icebergTable *v1alpha1.IcebergTable) error {
	log := log.FromContext(ctx)

	// Extract spec from the submitted custom resource
	spec := icebergTable.Spec

	log.Info("Processing custom resource to generate JSON payload",
		"resourceName", icebergTable.Name,
		"namespace", icebergTable.Namespace,
		"dataProduct", spec.DataProduct,
		"catalog", spec.Catalog.Name,
		"database", spec.Database,
		"table", spec.Table,
		"schemaFields", len(spec.Schema),
		"partitionFields", len(spec.PartitionSpec))

	// Create the JSON payload based on the submitted IcebergTable custom resource
	// This directly maps the CR fields to the JSON structure
	payload := struct {
		DataProduct     string                    `json:"dataProduct"`
		Catalog         v1alpha1.CatalogSpec      `json:"catalog"`
		Database        string                    `json:"database"`
		Table           string                    `json:"table"`
		Schema          []v1alpha1.Column         `json:"schema"`
		PartitionSpec   []v1alpha1.PartitionField `json:"partitionSpec,omitempty"`
		Properties      map[string]string         `json:"properties,omitempty"`
		RetentionPolicy v1alpha1.RetentionPolicy  `json:"retentionPolicy"`
	}{
		DataProduct:     spec.DataProduct,     // From CR: dataProduct: esg
		Catalog:         spec.Catalog,         // From CR: catalog section (name: polaris, type: rest, warehouse: abfs://...)
		Database:        spec.Database,        // From CR: database: customer
		Table:           spec.Table,           // From CR: table: transactions-2
		Schema:          spec.Schema,          // From CR: schema array (id: long, amount: double, etc.)
		PartitionSpec:   spec.PartitionSpec,   // From CR: partitionSpec (date with day transform)
		Properties:      spec.Properties,      // From CR: properties (write.format.default: parquet)
		RetentionPolicy: spec.RetentionPolicy, // From CR: retentionPolicy (snapshotExpirationDays: 21)
	}

	// Marshal the custom resource data to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload from custom resource %s: %w", icebergTable.Name, err)
	}

	log.Info("Generated JSON payload from submitted custom resource",
		"resourceName", icebergTable.Name,
		"jsonPayload", string(jsonData))

	log.Info("Making HTTP POST call to provision Iceberg table based on custom resource",
		"url", "http://localhost:8081/provision-iceberg-table",
		"resourceName", icebergTable.Name,
		"tableIdentifier", fmt.Sprintf("%s.%s", spec.Database, spec.Table))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create the POST request with the CR-generated payload
	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:8081/provision-iceberg-table", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for custom resource %s: %w", icebergTable.Name, err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "table-definition-operator/v1alpha1")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request for custom resource %s: %w", icebergTable.Name, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed for custom resource %s with status code: %d", icebergTable.Name, resp.StatusCode)
	}

	log.Info("Successfully provisioned Iceberg table via HTTP API based on custom resource",
		"resourceName", icebergTable.Name,
		"table", fmt.Sprintf("%s.%s", spec.Database, spec.Table),
		"statusCode", resp.StatusCode)
	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IcebergTableReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the IcebergTable instance
	var icebergTable v1alpha1.IcebergTable
	if err := r.Get(ctx, req.NamespacedName, &icebergTable); err != nil {
		if errors.IsNotFound(err) {
			// Resource was deleted, nothing to do
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get IcebergTable")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling IcebergTable custom resource - received from Kubernetes cluster",
		"resourceName", icebergTable.Name,
		"namespace", icebergTable.Namespace,
		"dataProduct", icebergTable.Spec.DataProduct,
		"catalog", icebergTable.Spec.Catalog.Name,
		"database", icebergTable.Spec.Database,
		"table", icebergTable.Spec.Table,
		"snapshotExpirationDays", icebergTable.Spec.RetentionPolicy.SnapshotExpirationDays,
		"schemaFieldCount", len(icebergTable.Spec.Schema),
		"partitionFieldCount", len(icebergTable.Spec.PartitionSpec))

	// Check if the table is already provisioned (check status)
	if icebergTable.Status.State == "Provisioned" {
		log.Info("Table already provisioned, skipping")
		return ctrl.Result{}, nil
	}

	// Update status to "Provisioning"
	icebergTable.Status.State = "Provisioning"
	icebergTable.Status.Message = "Provisioning Iceberg table..."
	if err := r.Status().Update(ctx, &icebergTable); err != nil {
		log.Error(err, "Failed to update status to Provisioning")
		return ctrl.Result{}, err
	}

	// Make the HTTP call to provision the table based on the custom resource
	if err := r.provisionIcebergTable(ctx, &icebergTable); err != nil {
		log.Error(err, "Failed to provision Iceberg table")

		// Update status to "Failed"
		icebergTable.Status.State = "Failed"
		icebergTable.Status.Message = fmt.Sprintf("Failed to provision table: %v", err)
		if statusErr := r.Status().Update(ctx, &icebergTable); statusErr != nil {
			log.Error(statusErr, "Failed to update status to Failed")
		}

		// Requeue after 5 minutes to retry
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}

	// Update status to "Provisioned"
	icebergTable.Status.State = "Provisioned"
	icebergTable.Status.Message = "Iceberg table successfully provisioned"
	if err := r.Status().Update(ctx, &icebergTable); err != nil {
		log.Error(err, "Failed to update status to Provisioned")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled IcebergTable")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IcebergTableReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tablesv1alpha1.IcebergTable{}).
		Named("icebergtable").
		Complete(r)
}
