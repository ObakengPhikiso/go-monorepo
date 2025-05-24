package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
)

// Service discovery: static lists of backend addresses
var (
	userBackends    = []string{"http://users:8080"}
	orderBackends   = []string{"http://orders:8080"}
	paymentBackends = []string{"http://payments:8080"}

	userIdx    uint32
	orderIdx   uint32
	paymentIdx uint32
)

func pickBackend(backends []string, idx *uint32) string {
	n := uint32(len(backends))
	if n == 0 {
		return ""
	}
	i := atomic.AddUint32(idx, 1)
	return backends[i%n]
}

func proxy(backends []string, idx *uint32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func main() {
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
