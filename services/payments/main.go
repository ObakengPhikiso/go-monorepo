package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/obakengphikiso/go-monorepo/libs/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Payment struct {
	ID        string    `json:"id" bson:"id"`
	Amount    float64   `json:"amount" bson:"amount"`
	Currency  string    `json:"currency" bson:"currency"`
	Status    string    `json:"status" bson:"status"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

var paymentsCollection *mongo.Collection

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Printf("write error: %v", err)
	}
}

func getPayments(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := paymentsCollection.Find(ctx, bson.M{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("db error")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	var payments []Payment
	if err := cursor.All(ctx, &payments); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("db error")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payments); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/payments/"):]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p Payment
	err := paymentsCollection.FindOne(ctx, bson.M{"id": id}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("not found")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("db error")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func createPayment(w http.ResponseWriter, r *http.Request) {
	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("invalid body")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	if p.Amount <= 0 || p.Currency == "" {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("amount and currency required")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	p.ID = shared.GenerateID()
	p.Status = "pending"
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := paymentsCollection.InsertOne(ctx, p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("db error")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	shared.Logger("Created payment: %+v", p)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func updatePayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/payments/"):]
	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("invalid body")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.UpdatedAt = time.Now()
	update := bson.M{"$set": bson.M{
		"amount":     p.Amount,
		"currency":   p.Currency,
		"status":     p.Status,
		"updated_at": p.UpdatedAt,
	}}
	res, err := paymentsCollection.UpdateOne(ctx, bson.M{"id": id}, update)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("db error")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	if res.MatchedCount == 0 {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("not found")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	shared.Logger("Updated payment: %s", id)
	w.WriteHeader(http.StatusNoContent)
}
func deletePayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/payments/"):]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := paymentsCollection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("db error")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	if res.DeletedCount == 0 {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("not found")); err != nil {
			log.Printf("write error: %v", err)
		}
		return
	}
	shared.Logger("Deleted payment: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	dbURL := shared.GetEnv("PAYMENTS_DB_URL", "mongodb://localhost:27017/payments")
	var err error
	paymentsCollection, err = shared.GetMongoCollection(dbURL, "payments", "payments")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

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
		switch r.Method {
		case http.MethodGet:
			getPayment(w, r)
		case http.MethodPut:
			updatePayment(w, r)
		case http.MethodDelete:
			deletePayment(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	shared.Logger("[payments] Using DB: %s", dbURL)
	fmt.Println("[payments] Service running on :8080")
	fmt.Println("Shared lib version:", shared.Version())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
