package handlers

import (
	"github.com/MerchantGuildMember/uptimemonitor/internal/middleware"
	"github.com/MerchantGuildMember/uptimemonitor/internal/models"
	"github.com/go-chi/chi/v5"
    "github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
    "golang.org/x/net/html"
    "strings"
    "time"
    "context"
    "log"
    "encoding/json"
    "fmt"
)

// Handler holds shared dependencies for all route handlers
type Handler struct {
	pool      *pgxpool.Pool
	jwtSecret []byte
}

// New creates a Handler with the given connection pool and JWT signing secret
func New(pool *pgxpool.Pool, jwtSecret []byte) *Handler {
	return &Handler{pool: pool, jwtSecret: jwtSecret}
}

// POST /monitors, register a URL to watch
func (h *Handler) CreateMonitor(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// require a valid http/https URL with a host
	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		http.Error(w, "invalid URL: must be an absolute http or https URL", http.StatusBadRequest)
		return
	}

	// user ID comes from the JWT — never from the request body (would be a privilege-escalation hole)
	userID := middleware.UserIDFromContext(r.Context())

	// get title of the website; fall back to "Unknown" on failure
	title, err := FetchTitle(r.Context(), input.URL)
	if err != nil {
		log.Printf("FetchTitle failed for %s: %v", input.URL, err)
		title = "Unknown"
	}

	// use a transaction so the monitor row and the user_monitors link are always created together
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context()) // no-op if Commit succeeds

	// if another user already registered this URL, reuse the existing monitor row
	var monitorID string
	err = tx.QueryRow(r.Context(), `SELECT monitor_id FROM monitors WHERE url = $1`, input.URL).Scan(&monitorID)
	if err != nil {
		// monitor doesn't exist yet — create it
		genUUID, err := uuid.NewV7()
		if err != nil {
			http.Error(w, "failed to generate UUID", http.StatusInternalServerError)
			return
		}
		monitorID = genUUID.String()
		_, err = tx.Exec(r.Context(),
			`INSERT INTO monitors (monitor_id, url, website_name, first_created_by) VALUES ($1, $2, $3, $4)`,
			monitorID, input.URL, title, userID,
		)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
	}

	// link this user to the monitor; ON CONFLICT DO NOTHING handles the case where
	// the user tries to add a URL they're already watching
	_, err = tx.Exec(r.Context(),
		`INSERT INTO user_monitors (user_id, monitor_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, monitorID,
	)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"monitor_id":   monitorID,
		"url":          input.URL,
		"website_name": title,
	})
}

// GET /monitors, return the authenticated user's monitors
func (h *Handler) GetMonitors(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	// join through user_monitors so each user only sees their own monitors
	query := `
		SELECT m.monitor_id, m.url, m.website_name, m.website_description, m.discord_webhook_url, m.created_at, m.first_created_by
		FROM monitors m
		JOIN user_monitors um ON um.monitor_id = m.monitor_id
		WHERE um.user_id = $1`

	rows, err := h.pool.Query(r.Context(), query, userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	monitors := make([]models.Monitor, 0)
	for rows.Next() {
		var m models.Monitor
		err := rows.Scan(
			&m.MonitorID,
			&m.URL,
			&m.WebsiteName,
			&m.WebsiteDescription,
			&m.DiscordWebhookURL,
			&m.CreatedAt,
			&m.FirstCreatedBy,
		)
		if err != nil {
			http.Error(w, "failed to read row", http.StatusInternalServerError)
			return
		}
		monitors = append(monitors, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitors)
}

// GET /monitors/{id}/history, return past checks for a monitor plus uptime percentage
func (h *Handler) GetMonitorHistory(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "id")
	userID := middleware.UserIDFromContext(r.Context())

	// verify the requesting user is linked to this monitor before returning its data
	var count int
	err := h.pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM user_monitors WHERE user_id = $1 AND monitor_id = $2`,
		userID, monitorID,
	).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "monitor not found", http.StatusNotFound)
		return
	}

	query := `
		SELECT check_id, monitor_id, recorded, summary_status, reports, response_time_ms
		FROM checks
		WHERE monitor_id = $1
		ORDER BY recorded DESC`

	rows, err := h.pool.Query(r.Context(), query, monitorID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	checks := make([]models.Check, 0)
	for rows.Next() {
		var c models.Check
		err := rows.Scan(
			&c.CheckID,
			&c.MonitorID,
			&c.Recorded,
			&c.SummaryStatus,
			&c.Reports,
			&c.ResponseTimeMs,
		)
		if err != nil {
			http.Error(w, "failed to read row", http.StatusInternalServerError)
			return
		}
		checks = append(checks, c)
	}

	// uptime = checks where status is not Down / total checks
	// Working and Struggling both count as "up" struggling means slow but reachable
	var upPercent float64
	if len(checks) > 0 {
		var upCount int
		for _, c := range checks {
			if c.SummaryStatus != "Down" {
				upCount++
			}
		}
		upPercent = float64(upCount) / float64(len(checks)) * 100
	}

	resp := struct {
		UptimePercent float64       `json:"uptime_percent"`
		Checks        []models.Check `json:"checks"`
	}{
		UptimePercent: upPercent,
		Checks:        checks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// FetchTitle fetches the <title> tag from a given URL.
// Mimics a browser User-Agent to avoid bot-detection blocks.
func FetchTitle(ctx context.Context, targetURL string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	// mimic a real browser so sites don't block the request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// walk HTML tokens until we find <title>
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return "", fmt.Errorf("title tag not found")
		case html.StartTagToken:
			if tokenizer.Token().Data == "title" {
				if tokenizer.Next() == html.TextToken {
					return strings.TrimSpace(tokenizer.Token().Data), nil
				}
			}
		}
	}
}
