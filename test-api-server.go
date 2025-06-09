package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type IcebergTableRequest struct {
	DataProduct     string                   `json:"dataProduct"`
	Catalog         map[string]interface{}   `json:"catalog"`
	Database        string                   `json:"database"`
	Table           string                   `json:"table"`
	Schema          []map[string]interface{} `json:"schema"`
	PartitionSpec   []map[string]interface{} `json:"partitionSpec,omitempty"`
	Properties      map[string]string        `json:"properties,omitempty"`
	RetentionPolicy map[string]interface{}   `json:"retentionPolicy"`
}

func provisionIcebergTable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var req IcebergTableRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// Log the received request
	fmt.Printf("🎉 Received Iceberg Table Provision Request:\n")
	fmt.Printf("   Data Product: %s\n", req.DataProduct)
	fmt.Printf("   Catalog: %s\n", req.Catalog["name"])
	fmt.Printf("   Database: %s\n", req.Database)
	fmt.Printf("   Table: %s\n", req.Table)
	fmt.Printf("   Schema Fields: %d\n", len(req.Schema))
	fmt.Printf("   Partition Fields: %d\n", len(req.PartitionSpec))
	fmt.Printf("   Retention Days: %v\n", req.RetentionPolicy["snapshotExpirationDays"])
	fmt.Printf("   Raw JSON: %s\n", string(body))
	fmt.Println("---")

	// Simulate successful provisioning
	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Successfully provisioned table %s.%s", req.Database, req.Table),
		"table":   fmt.Sprintf("%s.%s", req.Database, req.Table),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/provision-iceberg-table", provisionIcebergTable)

	fmt.Println("🚀 Test API Server starting on http://localhost:8081")
	fmt.Println("📋 Endpoint: /provision-iceberg-table")
	fmt.Println("🎯 Ready to receive operator HTTP calls...")

	log.Fatal(http.ListenAndServe(":8081", nil))
}
