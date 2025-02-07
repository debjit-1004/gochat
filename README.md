# 🚀 GoChat - Encrypted Terminal Chat Application

A secure, terminal-based chat application written in Go featuring real-time messaging, encryption, and persistent chat history.

## Features

- 🔒 End-to-end AES encryption
- 👥 Multi-user support
- 💾 Persistent chat history
- 🎨 Beautiful TUI using Bubble Tea
- 🌐 TCP-based networking
- ⌨️ Intuitive keyboard controls

## Prerequisites

```bash
go version >= 1.16
```

## Installation

```bash
# Clone the repository
git clone https://github.com/debjit-1004/gochat.git

# Navigate to project directory
cd gochat

# Install dependencies
go mod download

# Build the application
go build -o gochat
```

## Usage

### Starting the Server

```bash
./gochat
> Are you starting a server or client? server
> Enter your username: admin
> Enter the port to listen on: 8080
```

### Connecting as a Client

```bash
./gochat
> Are you starting a server or client? client
> Enter your username: user
> Enter the server public IP address: 
> Enter the server port: 8080
```

## Controls

- `Ctrl+S` - Send message
- `Enter` - New line
- `Ctrl+C` - Exit application
- `↑/↓` - Scroll through chat history

## Network Setup

### Server Requirements
1. Port forwarding (if behind router)
   - Forward external port to server's local IP
   - Protocol: TCP
   - Port: Your chosen port (e.g., 8080)

2. Firewall configuration
   ```powershell
   # Windows (Run as Administrator)
   New-NetFirewallRule -DisplayName "GoChat" -Direction Inbound -LocalPort 8080 -Protocol TCP -Action Allow
   ```

## Security

- Messages are encrypted using AES-256
- Each message has a unique identifier
- Chat history is stored encrypted

## Project Structure

```plaintext
gochat/
├── main.go         # Main application code
├── chat_history.json   # Persistent chat storage
└── README.md      # Documentation
```

## Technical Details

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for TUI
- Uses TCP sockets for networking
- AES encryption for message security
- JSON-based message persistence

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## License

MIT License - see LICENSE file for details

## Author

Your Name (@debjit-1004)

---
Made with ❤️ using Go