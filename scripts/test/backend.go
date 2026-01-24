package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := "3000"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	mux := http.NewServeMux()

	// Root endpoint - serve HTML page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>webctl Test Backend</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #f093fb 0%%, #f5576c 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 3rem;
            max-width: 600px;
        }
        h1 {
            color: #f5576c;
            margin-bottom: 1rem;
            font-size: 2.5rem;
        }
        p {
            color: #555;
            margin-bottom: 1rem;
            line-height: 1.6;
        }
        .info {
            background: #f0f4f8;
            padding: 1rem;
            border-radius: 8px;
            margin: 1rem 0;
            border-left: 4px solid #f5576c;
        }
        .endpoint {
            font-family: 'Courier New', monospace;
            background: #1e1e1e;
            color: #4ec9b0;
            padding: 0.5rem;
            border-radius: 4px;
            margin: 0.5rem 0;
        }
        .badge {
            display: inline-block;
            background: #f5576c;
            color: white;
            padding: 0.25rem 0.75rem;
            border-radius: 12px;
            font-size: 0.85rem;
            margin: 0.25rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔧 Backend Server</h1>
        <p>This is the webctl test backend server. You're seeing this page because you're directly accessing the backend or through a proxy.</p>

        <div class="info">
            <strong>Server Info:</strong><br>
            Version: 1.0.0<br>
            Time: %s<br>
            Port: %s
        </div>

        <h2 style="color: #f5576c; margin-top: 2rem;">Available API Endpoints</h2>
        <div class="endpoint">GET /api/hello</div>
        <div class="endpoint">GET /api/users</div>
        <div class="endpoint">GET /api/echo</div>
        <div class="endpoint">GET /status/200|400|404|500</div>
        <div class="endpoint">GET /delay</div>

        <div style="margin-top: 2rem;">
            <span class="badge">Backend</span>
            <span class="badge">Proxy Testing</span>
            <span class="badge">API Server</span>
        </div>
    </div>
</body>
</html>`, time.Now().Format(time.RFC3339), port)
	})

	// API endpoints
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Hello from test backend!",
		})
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		users := []map[string]any{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		}
		_ = json.NewEncoder(w).Encode(users)
	})

	mux.HandleFunc("/api/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.Query(),
			"headers": r.Header,
		}
		_ = json.NewEncoder(w).Encode(data)
	})

	// Status code endpoints
	mux.HandleFunc("/status/200", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
	})

	mux.HandleFunc("/status/400", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Bad Request"})
	})

	mux.HandleFunc("/status/404", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
	})

	mux.HandleFunc("/status/500", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Internal Server Error"})
	})

	// Delay endpoint for testing
	mux.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Delayed response"})
	})

	// CORS headers for all endpoints
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		mux.ServeHTTP(w, r)
	})

	addr := ":" + port
	fmt.Printf("Test backend server starting on http://localhost%s\n", addr)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  GET  /                - Backend HTML page")
	fmt.Println("  GET  /api/hello       - Hello message (JSON)")
	fmt.Println("  GET  /api/users       - User list (JSON)")
	fmt.Println("  GET  /api/echo        - Echo request details (JSON)")
	fmt.Println("  GET  /status/200      - 200 OK response (JSON)")
	fmt.Println("  GET  /status/400      - 400 Bad Request (JSON)")
	fmt.Println("  GET  /status/404      - 404 Not Found (JSON)")
	fmt.Println("  GET  /status/500      - 500 Internal Server Error (JSON)")
	fmt.Println("  GET  /delay           - Delayed response (JSON, 2s)")
	fmt.Println("\nPress Ctrl+C to stop")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
