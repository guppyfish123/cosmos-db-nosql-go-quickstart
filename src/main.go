package main

import (
    "log"
    "net/http"

    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()

    // Certification Endpoint 
    s := r.PathPrefix("/certifications/api").Subrouter()

    // List of paths for Certification Endpoint 
    s.HandleFunc("/cert/{Key}/{Value}", getCert).Methods("GET")
    s.HandleFunc("/certs", getCerts).Methods("GET")

    log.Fatal(http.ListenAndServe(":8000", r))
}


    