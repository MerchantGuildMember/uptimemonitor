package main

import (
    "github.com/joho/godotenv"
    "github.com/go-chi/chi/v5"
    "net/http"
    "os"
    "log"
)

func main() {
    // load env file
    err := godotenv.Load(".env")
    if err != nil {
        // handle error
        log.Fatal("Unable to reach .env!")
    }

    // create chi router
    r := chi.NewRouter()

    // health endpoint
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status": "ok"}`))
    })

    // strt http server
    err = http.ListenAndServe(":" + os.Getenv("PORT"), r)
    if err != nil {
            // handle error
            log.Fatal("Unable to serve!")
        }
}


