package models

import (
    "time"
)

type Monitor struct {
	MonitorID          string    `json:"monitor_id"`
	URL                string    `json:"url"`
	WebsiteName        string    `json:"website_name"`
	WebsiteDescription *string   `json:"website_description,omitempty"`
	DiscordWebhookURL  *string   `json:"discord_webhook_url,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	FirstCreatedBy     *string   `json:"first_created_by,omitempty"`
}