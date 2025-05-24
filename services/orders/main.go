package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ObakengPhikiso/monorepo/libs/shared"
)

type Order struct {
	ID     string `json:"id"`
	Amount string `json:"amount"`
}

var orders = []Order{
	{ID: shared.GenerateID(), Amount: "$100"},
	{ID: shared.GenerateID(), Amount: "$200"},
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/orders/"):]
	for _, o := range orders {
		if o.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(o)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid body"))
		return
	}
	o.ID = shared.GenerateID()
	orders = append(orders, o)
	shared.Logger("Created order: %+v", o)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getOrders(w, r)
		case http.MethodPost:
			createOrder(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getOrder(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	dbURL := shared.GetEnv("ORDERS_DB_URL", "mongodb://localhost:27017/orders")
	shared.Logger("[orders] Using DB: %s", dbURL)
	fmt.Println("[orders] Service running on :8080")
	fmt.Println("Shared lib version:", shared.Version())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
