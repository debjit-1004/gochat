package main

import (
	"chat/internal/client"
	"chat/internal/crypto"
	"chat/internal/server"
	"chat/internal/utils"
	"fmt"
	"os"
)

func main() {
	// Validate encryption key
	if err := crypto.ValidateKey(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Get connection details from user
	details, err := utils.GetConnectionDetails()
	if err != nil {
		fmt.Printf("Error getting connection details: %v\n", err)
		os.Exit(1)
	}

	// Start server or client based on user choice
	if details.Mode == "server" {
		fmt.Printf("Starting server on port %s...\n", details.Port)
		if err := server.Start(details.Port, details.Username); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Connecting to server at %s:%s...\n", details.IP, details.Port)
		if err := client.Connect(details.IP, details.Port, details.Username); err != nil {
			fmt.Printf("Client error: %v\n", err)
			os.Exit(1)
		}
	}
}
