package handlers

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "net/http"
    "golang.org/x/net/html"
    "strings"
    "time"
    "context"
    "log"
    "encoding/json"
    "fmt"
)

// standard go pattern for handler struct
type Handler struct {
    pool *pgxpool.Pool  // shared instance
}


// constructor
func New(pool *pgxpool.Pool) *Handler {
    return &Handler{pool: pool}
}

// POST /monitors, first bit is a pointer receiver
func (h *Handler) CreateMonitor(w http.ResponseWriter, r *http.Request) {

    // generate UUID for the record
    genUUID, err := uuid.NewV7()
    if err != nil {
        http.Error(w, "Failed to generate UUID", http.StatusInternalServerError) // failed to generate uuid for some reason
        return
    }

    // create temp struct for incoming json data
    var input struct {
        URL string `json:"url"`
        USER string `json:"first_created_by"`
    }

    // read json data
    err = json.NewDecoder(r.Body).Decode(&input)
        if err != nil {
            http.Error(w, "Invalid JSON data", http.StatusBadRequest)
            return
        }

    // get title of the website, if err then return "Unknown"
    title, err := FetchTitle(context.Background(), input.URL)
    if err != nil {
        log.Printf("Failed to get Title!: %v", err)
        title = "Unknown"
    }

    // send query to handler and save it to database
    query := `INSERT INTO monitors (monitor_id, url, website_name) VALUES ($1, $2, $3);`
        _, err = h.pool.Exec(r.Context(), query, genUUID, input.URL, title)
        if err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }

    // send response back to user
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("Monitor created successfully!"))
}

func FetchTitle(ctx context.Context, targetURL string) (string, error) {
    client := &http.Client{Timeout: 5 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
    if err != nil {
        return "", err
    }

    // mimic browser
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close() // register cleanup

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
    }

    // html tokenizer
    tokenizer := html.NewTokenizer(resp.Body)
    for {
        tokenType := tokenizer.Next()
        switch tokenType {
            case html.ErrorToken:
                return "", fmt.Errorf("title tag not found in HTML")
            case html.StartTagToken:
                token := tokenizer.Token()
                if token.Data == "title" {
                    if tokenizer.Next() == html.TextToken {
                        titleText := strings.TrimSpace(tokenizer.Token().Data)
                        return titleText, nil
                    }
                }
        }
    }
}