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

    // strt http server
    http.ListenAndServe(":" + os.Getenv("PORT"), r)
}


