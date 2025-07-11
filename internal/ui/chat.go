package ui

import (
	"bufio"
	"chat/internal/crypto"
	"chat/internal/models"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
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

// chatModel represents the chat UI model
type chatModel struct {
	conn     net.Conn
	username string
	viewport viewport.Model
	textarea textarea.Model
	messages []models.Message
	err      error
	ready    bool
	msgChan  chan string
	seenMsgs map[string]bool
	width    int
}

// StartChatUI starts the chat user interface
func StartChatUI(conn net.Conn, username string) error {
	p := tea.NewProgram(initialModel(conn, username))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %v", err)
	}
	return nil
}

// initialModel creates the initial model for the chat UI
func initialModel(conn net.Conn, username string) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Prompt = "│ "
	ta.Focus()
	ta.ShowLineNumbers = false
	
	// Set up key bindings
	ta.KeyMap.InsertNewline.SetEnabled(false)
	
	// Custom key binding for sending messages
	sendKey := key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("Ctrl+S", "send message"),
	)
	ta.KeyMap.InsertNewline = sendKey

	ta.SetWidth(50)
	ta.SetHeight(2)
	ta.CharLimit = 280

	return chatModel{
		conn:     conn,
		username: username,
		textarea: ta,
		messages: []models.Message{},
		msgChan:  make(chan string),
		seenMsgs: make(map[string]bool),
	}
}

// Init initializes the chat model
func (m chatModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		waitForMessages(m.conn, m.msgChan),
		readMessages(m.msgChan),
	)
}

// Update handles model updates
func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m.sendMessage()
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
		m.processIncomingMessage(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd, readMessages(m.msgChan))
}

// sendMessage sends a message to the server
func (m chatModel) sendMessage() (tea.Model, tea.Cmd) {
	if m.textarea.Value() == "" {
		return m, nil
	}

	msgID := fmt.Sprintf("%s-%d", m.username, time.Now().UnixNano())
	newMsg := models.Message{
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

	encrypted, err := crypto.EncryptMessage(fmt.Sprintf("%s|%s|%s|%s",
		newMsg.Username, newMsg.Content, newMsg.Time.Format(time.RFC3339), msgID))
	if err != nil {
		m.err = err
		return m, nil
	}
	
	m.conn.Write([]byte(encrypted + "\n"))
	m.textarea.Reset()
	
	return m, nil
}

// processIncomingMessage processes incoming messages
func (m *chatModel) processIncomingMessage(msg string) {
	parts := strings.SplitN(msg, "|", 4)
	if len(parts) == 4 {
		t, _ := time.Parse(time.RFC3339, parts[2])
		msgID := parts[3]

		if !m.seenMsgs[msgID] {
			newMsg := models.Message{
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

// View renders the chat interface
func (m chatModel) View() string {
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

// waitForMessages waits for incoming messages from the server
func waitForMessages(conn net.Conn, msgChan chan string) tea.Cmd {
	return func() tea.Msg {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			decrypted, err := crypto.DecryptMessage(scanner.Text())
			if err != nil {
				continue
			}
			msgChan <- decrypted
		}
		return nil
	}
}

// readMessages reads messages from the message channel
func readMessages(msgChan chan string) tea.Cmd {
	return func() tea.Msg {
		msg := <-msgChan
		return msg
	}
}

// formatMessages formats messages for display
func formatMessages(messages []models.Message, width int) string {
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
