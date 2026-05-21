package db

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "os"
    "context"
    "log"
    "time"
    )

func ConnectDB() (*pgxpool.Pool, error) {
    dbURL, exists := os.LookupEnv("DATABASE_URL")
    if !exists {
        log.Fatal("DATABASE_URL is missing from the environment!")
    }

    // create a 5 second timeout context
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel() // tells Go to clean up the context's resources when func finishes

    pool, err := pgxpool.New(ctx, dbURL)
    if err != nil {
        return nil, err
    }

    // ping DB to verify conn
    if err := pool.Ping(ctx); err != nil {
        pool.Close() // clean pool if ping fails
        return nil, err
    }

    return pool, nil
}