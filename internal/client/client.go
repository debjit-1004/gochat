package client

import (
	"chat/internal/ui"
	"fmt"
	"net"
)

// Connect connects to a chat server
func Connect(ip, port, username string) error {
	address := ip + ":" + port
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("connection error: %v", err)
	}
	defer conn.Close()

	// Start the chat UI
	return ui.StartChatUI(conn, username)
}
