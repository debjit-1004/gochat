package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConnectionDetails holds connection information
type ConnectionDetails struct {
	Mode     string
	IP       string
	Port     string
	Username string
}

// GetConnectionDetails prompts user for connection details
func GetConnectionDetails() (ConnectionDetails, error) {
	reader := bufio.NewReader(os.Stdin)
	details := ConnectionDetails{}

	fmt.Print("Are you starting a server or client? (server/client): ")
	mode, err := reader.ReadString('\n')
	if err != nil {
		return details, err
	}
	details.Mode = strings.TrimSpace(mode)

	if details.Mode != "server" && details.Mode != "client" {
		return details, fmt.Errorf("invalid mode: %s", details.Mode)
	}

	fmt.Print("Enter your username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return details, err
	}
	details.Username = strings.TrimSpace(username)

	if details.Mode == "server" {
		fmt.Print("Enter the port to listen on (default 8080): ")
		port, err := reader.ReadString('\n')
		if err != nil {
			return details, err
		}
		details.Port = strings.TrimSpace(port)
		if details.Port == "" {
			details.Port = "8080"
		}
	} else {
		fmt.Print("Enter the server IP address (default localhost): ")
		ip, err := reader.ReadString('\n')
		if err != nil {
			return details, err
		}
		details.IP = strings.TrimSpace(ip)
		if details.IP == "" {
			details.IP = "localhost"
		}

		fmt.Print("Enter the server port (default 8080): ")
		port, err := reader.ReadString('\n')
		if err != nil {
			return details, err
		}
		details.Port = strings.TrimSpace(port)
		if details.Port == "" {
			details.Port = "8080"
		}
	}

	return details, nil
}
