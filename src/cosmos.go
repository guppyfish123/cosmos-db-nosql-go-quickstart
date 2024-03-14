package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

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


// getCert godoc
// @Summary      Show a Specified Cert
// @Description  Get a certification by a certain key & value
// @Tags         Certifications
// @Accept       json
// @Produce      json
// @Param        key	path	string	true	"Select Category for Search"	Enums(id, category, company)
// @Param        value	path	string	true	"Select Category for Search"	example(Microsoft)
// @Success      200  {object}  Certs
// @Router       /cert/{key}/{value} [get]
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

	//Input UEL Parameters 
    vars := mux.Vars(r)
    
    inputKey := vars["Key"]
    inputValue := vars["Value"]

	// inputKey validation to prevent code injection and ensure data santization 
	// Restricting input 
	validKeyInputs := [...]string{"id", "category", "company"}
	found := false

	// Loop to check if input paramter is valid 
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

	// Data sanatize as to prevent buffer overflow attacks and restrict the amount of data that can be injected 
	if len(inputKey)+len(inputValue) > 50 {
		http.Error(w, "Name exceeds maximum length", http.StatusBadRequest)
		return
	}

	partitionKey := azcosmos.NewPartitionKeyString("certification")

    // Construct the query dynamically
	// Validates and sanitizes user input to prevent security vulnerabilities such as SQL injection
    query := "SELECT * FROM certifications c WHERE c." + inputKey + " = @value"
    queryOptions := azcosmos.QueryOptions{
        QueryParameters: []azcosmos.QueryParameter{
            {Name: "@value", Value: inputValue},
        },
    }

	context := context.TODO()
	// Execute query to the cosmos container
	queryPager := container.NewQueryItemsPager(query, partitionKey, &queryOptions)

	var certs []Certs

	// https://learn.microsoft.com/en-us/azure/cosmos-db/nosql/query/pagination
	// How azure Csomos DB operates thru the use of pages which can span mulitple pages depending on certain conditions
	// each query made can be different can can contain output different number of pages and a different number of items on each page
	// This function loops thru each page and each item on those pages to passes them thru to a certs variable to output
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

	// Response output
	Responder(w, certs)
}


// getCert godoc
// @Summary      Show Certification Listings
// @Description  List all current Certifications by Patrick
// @Tags         Certifications
// @Accept       json
// @Produce      json
// @Param   top     query     int        false  "Top results"          example(3)
// @Success      200  {object}  Certs
// @Router       /certs [get]
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

	// Gets query from url 
	// This is not in route table as mux makes the query option mandatory if used there 
	values := r.URL.Query()
    topResults := values.Get("top")
	topNum := 0

	// If topResults is provided, validate it
	if topResults != "" {
		// Convert topResults to integer
		var err error
		topNum, err = strconv.Atoi(topResults)
		if err != nil || topNum <= 0 {
			http.Error(w, "Invalid 'top' value", http.StatusBadRequest)
			return
		}
	}

	// Define query and options depending of input of query or not
	query := "SELECT * FROM certifications c WHERE c.category = @category"
	if topNum > 0 {
		query = "SELECT TOP @topResults * FROM certifications c WHERE c.category = @category"
	}  


	// Seperating the value of @category is important for dynamic variable as it prevents sql injectin attacks
	queryOptions := azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@category", Value: "certification"},
			{Name: "@topResults", Value: topNum},
		},
	}

	// Execute query
	context := context.TODO()
	queryPager := container.NewQueryItemsPager(query, partitionKey, &queryOptions)

	var certs []Certs

	// https://learn.microsoft.com/en-us/azure/cosmos-db/nosql/query/pagination
	// How azure Csomos DB operates thru the use of pages which can span mulitple pages depending on certain conditions
	// each query made can be different can can contain output different number of pages and a different number of items on each page
	// This function loops thru each page and each item on those pages to passes them thru to a certs variable to output
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

	// Response output
	Responder(w, certs)
}

// This function is what cleans up the data and return it in a json format for the user to ready clearly 
func Responder(w http.ResponseWriter, certs interface{}) {
	// Write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(certs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
