package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ObakengPhikiso/monorepo/libs/shared"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var users = []User{
	{ID: shared.GenerateID(), Name: "Alice"},
	{ID: shared.GenerateID(), Name: "Bob"},
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/users/"):]
	for _, u := range users {
		if u.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(u)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid body"))
		return
	}
	u.ID = shared.GenerateID()
	users = append(users, u)
	shared.Logger("Created user: %+v", u)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUsers(w, r)
		case http.MethodPost:
			createUser(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getUser(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	dbURL := shared.GetEnv("USERS_DB_URL", "mongodb://localhost:27017/users")
	shared.Logger("[users] Using DB: %s", dbURL)
	fmt.Println("[users] Service running on :8080")
	fmt.Println("Shared lib version:", shared.Version())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
