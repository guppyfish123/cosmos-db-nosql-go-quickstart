package main

import (
	"context"
	"encoding/json"
	"os"
    "net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/gorilla/mux"
)

func authenticateCosmosDB() (*azcosmos.Client, error) {
	endpoint := os.Getenv("COSMOS_DB_ENDPOINT")

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	clientOptions := azcosmos.ClientOptions{
		EnableContentResponseOnWrite: true,
	}

	client, err := azcosmos.NewClient(endpoint, credential, &clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getCert(w http.ResponseWriter, r *http.Request) {

	///////////////////////////////////////////
	//DB Authentication
	client, err := authenticateCosmosDB()
	if err != nil {
		http.Error(w, "DB Auth Error", http.StatusInternalServerError)
		return
	}

	// <get_database>
	database, err := client.NewDatabase("cosmicworks")
	if err != nil {
		http.Error(w, "DB Auth Error", http.StatusInternalServerError)
		return
	}

	// <get_container>
	container, err := database.NewContainer("certifications")
	if err != nil {
		http.Error(w, "DB Auth Error", http.StatusInternalServerError)
		return
	}

	///////////////////////////////////////////

	//Input Parameters
    vars := mux.Vars(r)
    
    inputKey := vars["key"]
    inputValue := vars["value"]

	// inputKey validation
	validKeyInputs := [...]string{"id", "category", "company"}

	found := false

	for i := 0; i < 5; i++ {
		if inputKey == validKeyInputs[i] {
			found = true
			break
		} 
	}
	if !found {
		http.Error(w, "Error: Input key is not valid.", http.StatusBadRequest)
		return
	}

	// Validate Name length and pattern
	if len(inputKey)+len(inputValue) > 30 {
		http.Error(w, "Name exceeds maximum length", http.StatusBadRequest)
		return
	}

	partitionKey := azcosmos.NewPartitionKeyString("certification")

    // Construct the query dynamically
    query := "SELECT * FROM c WHERE " + inputKey + " = @value"
    queryOptions := azcosmos.QueryOptions{
        QueryParameters: []azcosmos.QueryParameter{
            {Name: "@value", Value: inputValue},
        },
    }

	// Execute query
	context := context.TODO()
	queryPager := container.NewQueryItemsPager(query, partitionKey, &queryOptions)

	var certs []Certs

	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, item := range queryResponse.Items {
			var cert Certs
			if err := json.Unmarshal(item, &cert); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			certs = append(certs, cert)
		}
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(certs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getCerts(w http.ResponseWriter, r *http.Request) {
	// Authenticate Cosmos DB
	client, err := authenticateCosmosDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get database and container
	database, err := client.NewDatabase("cosmicworks")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	container, err := database.NewContainer("certifications")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	partitionKey := azcosmos.NewPartitionKeyString("certification")

	// Define query and options
	query := "SELECT * FROM certifications c WHERE c.category = @category"

	// Seperating the value of @category is important for dynamic variable as it prevents sql injectin attacks
	queryOptions := azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@category", Value: "certification"},
		},
	}

	// Execute query
	context := context.TODO()
	queryPager := container.NewQueryItemsPager(query, partitionKey, &queryOptions)

	var certs []Certs

	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, item := range queryResponse.Items {
			var cert Certs
			if err := json.Unmarshal(item, &cert); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			certs = append(certs, cert)
		}
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(certs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}