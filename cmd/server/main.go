package main

import (
    "github.com/joho/godotenv"
    "github.com/go-chi/chi/v5"
    "net/http"
    "os"
    "log"
    "github.com/MerchantGuildMember/uptimemonitor/internal/db"
    "github.com/MerchantGuildMember/uptimemonitor/internal/handlers"
	"github.com/MerchantGuildMember/uptimemonitor/internal/middleware"
)

func main() {
    // load env file
    if err := godotenv.Load(".env"); err != nil {
        // handle error and switch to system environment
        log.Printf("Unable to reach .env!: %v", err)
    }

    // call ConnectDB()
    pool, err := db.ConnectDB()
    if err != nil {
        log.Fatalf("Failed to connect to DB: %v", err)
    }
    defer pool.Close() // register pool clean up

    // create chi router
    r := chi.NewRouter()

	// global middleware, runs on every request before routing
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

    // health endpoint
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status": "ok"}`))
    })

	h := handlers.New(pool, []byte(secret))

	// public routes, no JWT required
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)

	// protected routes
	// authenticate validates the bearer token and injects the user ID
	// into the request context for all handlers in this group
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.Authenticate([]byte(secret)))
		r.Post("/monitors", h.CreateMonitor)
		r.Get("/monitors", h.GetMonitors)
		r.Get("/monitors/{id}/history", h.GetMonitorHistory)
	})

    port, exists := os.LookupEnv("PORT")
    if !exists || port == "" {
        port = "8080" // fallback
    }

    // print which port will be used
	log.Printf("Starting server on port %s...", port)

    // start http server
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}


