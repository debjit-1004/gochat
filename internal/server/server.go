package server

import (
	"bufio"
	"chat/internal/crypto"
	"chat/internal/models"
	"chat/internal/ui"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// ChatHistory manages chat message history
type ChatHistory struct {
	messages []models.Message
	mu       sync.Mutex
}

// Server represents the chat server
type Server struct {
	clients     sync.Map
	history     ChatHistory
	ln          net.Listener
	historyFile *os.File
}

// Start starts the chat server
func Start(port, username string) error {
	// Start server UI in a goroutine
	go startServerUI(port, username)

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("error starting server: %v", err)
	}
	defer ln.Close()

	historyFile, err := os.OpenFile("chat_history.json", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error opening history file: %v", err)
	}
	defer historyFile.Close()

	server := &Server{
		ln:          ln,
		historyFile: historyFile,
	}
	server.loadHistory()

	fmt.Printf("🚀 Server started by %s on port %s\n", username, port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go server.handleClient(conn)
	}
}

// handleClient handles individual client connections
func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	s.clients.Store(conn, true)
	defer s.clients.Delete(conn)

	// Send chat history to new client
	s.history.mu.Lock()
	for _, msg := range s.history.messages {
		encrypted, _ := crypto.EncryptMessage(fmt.Sprintf("%s|%s|%s|%s",
			msg.Username, msg.Content, msg.Time.Format(time.RFC3339), msg.ID))
		conn.Write([]byte(encrypted + "\n"))
	}
	s.history.mu.Unlock()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		decrypted, err := crypto.DecryptMessage(scanner.Text())
		if err != nil {
			continue
		}

		parts := strings.SplitN(decrypted, "|", 4)
		if len(parts) != 4 {
			continue
		}

		t, _ := time.Parse(time.RFC3339, parts[2])
		msg := models.Message{
			Username: parts[0],
			Content:  parts[1],
			Time:     t,
			ID:       parts[3],
		}

		s.history.mu.Lock()
		s.history.messages = append(s.history.messages, msg)
		json.NewEncoder(s.historyFile).Encode(msg)
		s.history.mu.Unlock()

		// Broadcast message to all clients
		encrypted, _ := crypto.EncryptMessage(fmt.Sprintf("%s|%s|%s|%s",
			msg.Username, msg.Content, msg.Time.Format(time.RFC3339), msg.ID))
		s.clients.Range(func(key, value interface{}) bool {
			if client, ok := key.(net.Conn); ok {
				client.Write([]byte(encrypted + "\n"))
			}
			return true
		})
	}
}

// loadHistory loads chat history from file
func (s *Server) loadHistory() {
	decoder := json.NewDecoder(s.historyFile)
	for {
		var msg models.Message
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		s.history.messages = append(s.history.messages, msg)
	}
}

// startServerUI starts the server UI
func startServerUI(port, username string) {
	// Wait a moment for server to start
	time.Sleep(500 * time.Millisecond)
	
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		fmt.Printf("Server UI connection error: %v\n", err)
		return
	}
	defer conn.Close()

	// Use the UI package to start the chat interface
	ui.StartChatUI(conn, username)
}
