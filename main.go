package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var encryptionKey = []byte("12345678901234567890123456789012")

type Message struct {
	Username string
	Content  string
	Time     time.Time
	ID       string
}

type ChatHistory struct {
	messages []Message
	mu       sync.Mutex
}

type Server struct {
	clients     sync.Map
	history     ChatHistory
	ln          net.Listener
	historyFile *os.File
}

var serverInstance *Server

var (
	appStyle    = lipgloss.NewStyle().Padding(1, 2)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).PaddingBottom(1)
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5A5A5A")).PaddingTop(1)
	inputStyle  = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3A3A3")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginRight(2)
	msgStyle    = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder(), false, false, false, true)
	userStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	timeStyle   = lipgloss.NewStyle().Faint(true)
	serverStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
)

func main() {
	if len(encryptionKey) != 32 {
		fmt.Printf("Invalid key size: %d bytes. Must be exactly 32 bytes.\n", len(encryptionKey))
		return
	}

	mode, ip, port, username, err := getConnectionDetails()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if mode == "server" {
		startServer(port, username)
	} else {
		startClient(ip, port, username)
	}
}

type model struct {
	conn     net.Conn
	username string
	viewport viewport.Model
	textarea textarea.Model
	messages []Message
	err      error
	ready    bool
	msgChan  chan string
	seenMsgs map[string]bool
	width    int
}

func initialModel(conn net.Conn, username string) model {
	ta := textarea.New()
	ta.Placeholder = "Type your message (Ctrl+S to send)..."
	ta.Prompt = "│ "
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("Ctrl+S", "send message"),
	)
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("enter"), // Add enter key for new lines
		key.WithHelp("enter", "new line"),
	)

	ta.SetWidth(50)
	ta.SetHeight(2)
	ta.CharLimit = 280

	return model{
		conn:     conn,
		username: username,
		textarea: ta,
		messages: []Message{},
		msgChan:  make(chan string),
		seenMsgs: make(map[string]bool),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		waitForMessages(m.conn, m.msgChan),
		readMessages(m.msgChan),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.textarea.KeyMap.InsertNewline):
			if m.textarea.Value() != "" {
				msgID := fmt.Sprintf("%s-%d", m.username, time.Now().UnixNano())
				newMsg := Message{
					Username: m.username,
					Content:  m.textarea.Value(),
					Time:     time.Now(),
					ID:       msgID,
				}

				if !m.seenMsgs[msgID] {
					m.messages = append(m.messages, newMsg)
					m.seenMsgs[msgID] = true
					m.viewport.SetContent(formatMessages(m.messages, m.width))
					m.viewport.GotoBottom()
				}

				encrypted, err := encryptMessage(fmt.Sprintf("%s|%s|%s|%s",
					newMsg.Username, newMsg.Content, newMsg.Time.Format(time.RFC3339), msgID))
				if err != nil {
					m.err = err
					return m, nil
				}
				m.conn.Write([]byte(encrypted + "\n"))
				m.textarea.Reset()
			}
		case msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc:
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-7)
			m.viewport.Width = msg.Width - 6
			m.textarea.SetWidth(msg.Width - 6)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 6
			m.viewport.Height = msg.Height - 7
			m.textarea.SetWidth(msg.Width - 6)
		}
		m.viewport.SetContent(formatMessages(m.messages, m.width))

	case string:
		parts := strings.SplitN(msg, "|", 4)
		if len(parts) == 4 {
			t, _ := time.Parse(time.RFC3339, parts[2])
			msgID := parts[3]

			if !m.seenMsgs[msgID] {
				newMsg := Message{
					Username: parts[0],
					Content:  parts[1],
					Time:     t,
					ID:       msgID,
				}
				m.messages = append(m.messages, newMsg)
				m.seenMsgs[msgID] = true
				m.viewport.SetContent(formatMessages(m.messages, m.width))
				m.viewport.GotoBottom()
			}
		}
	}

	return m, tea.Batch(tiCmd, vpCmd, readMessages(m.msgChan))
}

func waitForMessages(conn net.Conn, msgChan chan string) tea.Cmd {
	return func() tea.Msg {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			decrypted, err := decryptMessage(scanner.Text())
			if err != nil {
				continue
			}
			msgChan <- decrypted
		}
		return nil
	}
}

func readMessages(msgChan chan string) tea.Cmd {
	return func() tea.Msg {
		msg := <-msgChan
		return msg
	}
}

func formatMessages(messages []Message, width int) string {
	var formatted strings.Builder
	contentWidth := width - 30

	for _, msg := range messages {
		username := msg.Username
		if username == "server" {
			username = serverStyle.Render(username)
		}
		user := userStyle.Render(username + ":")
		time := timeStyle.Render(msg.Time.Format("15:04"))
		content := msgStyle.Render(lipgloss.NewStyle().Width(contentWidth).Render(msg.Content))

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(25).Render(lipgloss.JoinVertical(lipgloss.Left, user, time)),
			content,
		)
		formatted.WriteString(line + "\n")
	}
	return formatted.String()
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	title := titleStyle.Render("💬 GoChat - " + m.username)
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		m.viewport.View(),
		lipgloss.NewStyle().PaddingTop(1).Render(inputStyle.Render(m.textarea.View())),
		statusStyle.Render("Ctrl+S to send • Ctrl+C to quit • "+time.Now().Format("15:04")),
	)

	return appStyle.Render(view)
}

func encryptMessage(message string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(message))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(message))

	return hex.EncodeToString(ciphertext), nil
}

func decryptMessage(encrypted string) (string, error) {
	ciphertext, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

func getConnectionDetails() (mode, ip, port, username string, err error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Are you starting a server or client? (server/client): ")
	mode, err = reader.ReadString('\n')
	if err != nil {
		return "", "", "", "", err
	}
	mode = strings.TrimSpace(mode)

	if mode != "server" && mode != "client" {
		return "", "", "", "", fmt.Errorf("invalid mode: %s", mode)
	}

	fmt.Print("Enter your username: ")
	username, err = reader.ReadString('\n')
	if err != nil {
		return "", "", "", "", err
	}
	username = strings.TrimSpace(username)

	if mode == "server" {
		fmt.Print("Enter the port to listen on: ")
		port, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", "", err
		}
		port = strings.TrimSpace(port)
	} else {
		fmt.Print("Enter the server IP address: ")
		ip, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", "", err
		}
		ip = strings.TrimSpace(ip)

		fmt.Print("Enter the server port: ")
		port, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", "", err
		}
		port = strings.TrimSpace(port)
	}

	return mode, ip, port, username, nil
}

func startServer(port, username string) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		conn, err := net.Dial("tcp", ":"+port)
		if err != nil {
			fmt.Printf("Server UI connection error: %v\n", err)
			return
		}
		defer conn.Close()

		p := tea.NewProgram(initialModel(conn, username))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running server UI: %v\n", err)
		}
	}()

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
	defer ln.Close()

	historyFile, err := os.OpenFile("chat_history.json", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Error opening history file: %v\n", err)
		return
	}
	defer historyFile.Close()

	serverInstance = &Server{
		ln:          ln,
		historyFile: historyFile,
	}
	serverInstance.loadHistory()

	fmt.Printf("🚀 Server started by %s on port %s\n", username, port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go serverInstance.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	s.clients.Store(conn, true)
	defer s.clients.Delete(conn)

	s.history.mu.Lock()
	for _, msg := range s.history.messages {
		encrypted, _ := encryptMessage(fmt.Sprintf("%s|%s|%s|%s",
			msg.Username, msg.Content, msg.Time.Format(time.RFC3339), msg.ID))
		conn.Write([]byte(encrypted + "\n"))
	}
	s.history.mu.Unlock()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		decrypted, err := decryptMessage(scanner.Text())
		if err != nil {
			continue
		}

		parts := strings.SplitN(decrypted, "|", 4)
		if len(parts) != 4 {
			continue
		}

		t, _ := time.Parse(time.RFC3339, parts[2])
		msg := Message{
			Username: parts[0],
			Content:  parts[1],
			Time:     t,
			ID:       parts[3],
		}

		s.history.mu.Lock()
		s.history.messages = append(s.history.messages, msg)
		json.NewEncoder(s.historyFile).Encode(msg)
		s.history.mu.Unlock()

		encrypted, _ := encryptMessage(fmt.Sprintf("%s|%s|%s|%s",
			msg.Username, msg.Content, msg.Time.Format(time.RFC3339), msg.ID))
		s.clients.Range(func(key, value interface{}) bool {
			if client, ok := key.(net.Conn); ok {
				client.Write([]byte(encrypted + "\n"))
			}
			return true
		})
	}
}

func (s *Server) loadHistory() {
	decoder := json.NewDecoder(s.historyFile)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		s.history.messages = append(s.history.messages, msg)
	}
}

func startClient(ip, port, username string) {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Printf("Connection error: %v\n", err)
		return
	}
	defer conn.Close()

	p := tea.NewProgram(initialModel(conn, username))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
