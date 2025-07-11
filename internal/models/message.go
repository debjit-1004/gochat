package models

import "time"

// Message represents a chat message
type Message struct {
	Username string    `json:"username"`
	Content  string    `json:"content"`
	Time     time.Time `json:"time"`
	ID       string    `json:"id"`
}
