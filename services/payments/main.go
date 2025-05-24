package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ObakengPhikiso/monorepo/libs/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Payment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var paymentsCollection *mongo.Collection

func getMongoCollection() *mongo.Collection {
	dbURL := shared.GetEnv("PAYMENTS_DB_URL", "mongodb://localhost:27017/payments")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	return client.Database("payments").Collection("payments")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getPayments(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := paymentsCollection.Find(ctx, bson.M{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	var payments []Payment
	if err := cursor.All(ctx, &payments); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/payments/"):]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p Payment
	err := paymentsCollection.FindOne(ctx, bson.M{"id": id}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func createPayment(w http.ResponseWriter, r *http.Request) {
	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid body"))
		return
	}
	p.ID = shared.GenerateID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := paymentsCollection.InsertOne(ctx, p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	shared.Logger("Created payment: %+v", p)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func main() {
	paymentsCollection = getMongoCollection()
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
