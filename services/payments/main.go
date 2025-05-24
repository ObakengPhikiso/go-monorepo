package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ObakengPhikiso/monorepo/libs/shared"
)

type Payment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var payments = []Payment{
	{ID: shared.GenerateID(), Status: "pending"},
	{ID: shared.GenerateID(), Status: "completed"},
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getPayments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/payments/"):]
	for _, p := range payments {
		if p.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func createPayment(w http.ResponseWriter, r *http.Request) {
	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid body"))
		return
	}
	p.ID = shared.GenerateID()
	payments = append(payments, p)
	shared.Logger("Created payment: %+v", p)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getPayments(w, r)
		case http.MethodPost:
			createPayment(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/payments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getPayment(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	dbURL := shared.GetEnv("PAYMENTS_DB_URL", "mongodb://localhost:27017/payments")
	shared.Logger("[payments] Using DB: %s", dbURL)
	fmt.Println("[payments] Service running on :8080")
	fmt.Println("Shared lib version:", shared.Version())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
