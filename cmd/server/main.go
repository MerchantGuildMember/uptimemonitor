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

    // health endpoint
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status": "ok"}`))
    })

    // check port in env
    port, exists := os.LookupEnv("PORT")
    if !exists || port == "" {
        port = "8080" // fallback
    }

    // create handler instance
    h := handlers.New(pool)

    // register route
    r.Post("/monitors", h.CreateMonitor)

    // print which port will be used
    log.Printf("Starting server on port %s...", port)

    // start http server
    if err := http.ListenAndServe(":" + port, r); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}


