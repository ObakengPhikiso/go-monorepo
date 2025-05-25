package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/ObakengPhikiso/monorepo/libs/shared"
)

// Service discovery: static lists of backend addresses
var (
	userBackends    = []string{"http://users:8080"}
	orderBackends   = []string{"http://orders:8080"}
	paymentBackends = []string{"http://payments:8080"}
	authBackends    = []string{"http://auth:8084"}

	userIdx    uint32
	orderIdx   uint32
	paymentIdx uint32
	authIdx    uint32
)

func pickBackend(backends []string, idx *uint32) string {
	n := uint32(len(backends))
	if n == 0 {
		return ""
	}
	i := atomic.AddUint32(idx, 1)
	return backends[i%n]
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks, swagger docs, and auth endpoints
		if r.URL.Path == "/health" ||
			r.URL.Path == "/swagger" ||
			r.URL.Path == "/swagger.yaml" ||
			r.URL.Path == "/auth/login" ||
			r.URL.Path == "/auth/register" {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// Validate token
		claims, err := shared.ValidateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add user info to request headers for downstream services
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-Username", claims.Username)
		r.Header.Set("X-Authenticated", "true")

		// Preserve the original Authorization header for any services that might need it
		next.ServeHTTP(w, r)
	}
}

func proxy(backends []string, idx *uint32) http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		target := pickBackend(backends, idx)
		if target == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("no backend available"))
			return
		}
		url := target + r.URL.Path
		if r.URL.RawQuery != "" {
			url += "?" + r.URL.RawQuery
		}
		req, err := http.NewRequest(r.Method, url, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("gateway error"))
			return
		}
		req.Header = r.Header
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("service unavailable"))
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}

func checkServiceHealth(url string) bool {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	healthy := true
	status := map[string]bool{
		"users":    checkServiceHealth(userBackends[0]),
		"orders":   checkServiceHealth(orderBackends[0]),
		"payments": checkServiceHealth(paymentBackends[0]),
		"auth":     checkServiceHealth(authBackends[0]),
	}

	// If any service is unhealthy, set overall health to false
	for _, isHealthy := range status {
		if !isHealthy {
			healthy = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if !healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Convert status map to JSON
	json := "{"
	for service, isHealthy := range status {
		json += `"` + service + `":` + map[bool]string{true: "true", false: "false"}[isHealthy] + ","
	}
	json = json[:len(json)-1] + "}" // Remove trailing comma and close
	w.Write([]byte(json))
}

func main() {
	// Health endpoint
	http.HandleFunc("/health", healthHandler)

	// Auth endpoints
	http.HandleFunc("/auth/register", proxy(authBackends, &authIdx))
	http.HandleFunc("/auth/login", proxy(authBackends, &authIdx))
	http.HandleFunc("/auth/validate", proxy(authBackends, &authIdx))

	// Protected endpoints
	http.HandleFunc("/users", proxy(userBackends, &userIdx))
	http.HandleFunc("/users/", proxy(userBackends, &userIdx))
	http.HandleFunc("/orders", proxy(orderBackends, &orderIdx))
	http.HandleFunc("/orders/", proxy(orderBackends, &orderIdx))
	http.HandleFunc("/payments", proxy(paymentBackends, &paymentIdx))
	http.HandleFunc("/payments/", proxy(paymentBackends, &paymentIdx))

	http.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		f, err := os.Open("docs/swagger.yaml")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("swagger spec not found"))
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	http.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html><html><head><title>Swagger UI</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css" /></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script><script>window.onload = function() { window.ui = SwaggerUIBundle({ url: '/swagger.yaml', dom_id: '#swagger-ui' }); };</script></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	log.Println("[api-gateway] Running on :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}
