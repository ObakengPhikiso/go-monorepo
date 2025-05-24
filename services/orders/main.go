package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ObakengPhikiso/monorepo/libs/shared"
)

type Order struct {
	ID     string `json:"id"`
	Amount string `json:"amount"`
}

var ordersCollection *mongo.Collection

func getMongoCollection() *mongo.Collection {
	dbURL := shared.GetEnv("ORDERS_DB_URL", "mongodb://localhost:27017/orders")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	return client.Database("orders").Collection("orders")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := ordersCollection.Find(ctx, bson.M{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	var orders []Order
	if err := cursor.All(ctx, &orders); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/orders/"):]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var o Order
	err := ordersCollection.FindOne(ctx, bson.M{"id": id}).Decode(&o)
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
	json.NewEncoder(w).Encode(o)
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid body"))
		return
	}
	o.ID = shared.GenerateID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ordersCollection.InsertOne(ctx, o)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("db error"))
		return
	}
	shared.Logger("Created order: %+v", o)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func main() {
	ordersCollection = getMongoCollection()
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
