package screens

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LoginSubmitMsg is sent when the user submits the login form.
type LoginSubmitMsg struct {
	Username string
	Password string
}

const (
	loginFieldUsername = 0
	loginFieldPassword = 1
	loginFieldCount    = 2
)

var (
	loginTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	loginLabelStyle = lipgloss.NewStyle().
			Width(10).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("#D1D5DB"))

	loginHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	loginErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	loginInputFocused = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				PaddingLeft(1).PaddingRight(1).
				Width(42)

	loginInputBlurred = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#6B7280")).
				PaddingLeft(1).PaddingRight(1).
				Width(42)
)

// LoginScreen is the initial authentication form.
type LoginScreen struct {
	width    int
	height   int
	focusIdx int
	errMsg   string

	usernameInput textinput.Model
	passwordInput textinput.Model
}

// NewLoginScreen constructs a LoginScreen.
func NewLoginScreen() LoginScreen {
	userIn := textinput.New()
	userIn.Placeholder = "username"
	userIn.CharLimit = 128
	userIn.Width = 38
	userIn.Focus()

	passIn := textinput.New()
	passIn.Placeholder = "password"
	passIn.CharLimit = 128
	passIn.Width = 38
	passIn.EchoMode = textinput.EchoPassword
	passIn.EchoCharacter = '•'

	return LoginScreen{
		usernameInput: userIn,
		passwordInput: passIn,
	}
}

// SetError sets an error message to display below the form.
func (l LoginScreen) SetError(msg string) LoginScreen {
	l.errMsg = msg
	return l
}

// Init satisfies tea.Model.
func (l LoginScreen) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the login screen.
func (l LoginScreen) Update(msg tea.Msg) (LoginScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
		return l, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			l.focusIdx = (l.focusIdx + 1) % loginFieldCount
			l.syncFocus()
			return l, nil
		case "shift+tab", "up":
			l.focusIdx = (l.focusIdx - 1 + loginFieldCount) % loginFieldCount
			l.syncFocus()
			return l, nil
		case "enter", "ctrl+s":
			return l, l.submit()
		}
	}

	return l.updateActiveInput(msg)
}

// View renders the login form.
func (l LoginScreen) View() string {
	rows := []string{
		l.row("Username", l.usernameInput, l.focusIdx == loginFieldUsername),
		l.row("Password", l.passwordInput, l.focusIdx == loginFieldPassword),
	}

	help := loginHelpStyle.Render("enter  login  ·  tab  next field  ·  ctrl+c  quit")

	parts := []string{
		loginTitleStyle.Render("klt — Log Viewer"),
		"",
		strings.Join(rows, "\n"),
		"",
		help,
	}
	if l.errMsg != "" {
		parts = append(parts, "", loginErrStyle.Render("  "+l.errMsg))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(parts, "\n"))
}

func (l *LoginScreen) syncFocus() {
	if l.focusIdx == loginFieldUsername {
		l.usernameInput.Focus()
		l.passwordInput.Blur()
	} else {
		l.usernameInput.Blur()
		l.passwordInput.Focus()
	}
}

func (l LoginScreen) submit() tea.Cmd {
	return func() tea.Msg {
		return LoginSubmitMsg{
			Username: strings.TrimSpace(l.usernameInput.Value()),
			Password: l.passwordInput.Value(),
		}
	}
}

func (l LoginScreen) row(label string, ti textinput.Model, focused bool) string {
	var wrapped string
	if focused {
		wrapped = loginInputFocused.Render(ti.View())
	} else {
		wrapped = loginInputBlurred.Render(ti.View())
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		loginLabelStyle.Render(label)+"  ",
		wrapped,
	)
}

func (l LoginScreen) updateActiveInput(msg tea.Msg) (LoginScreen, tea.Cmd) {
	var cmd tea.Cmd
	if l.focusIdx == loginFieldUsername {
		l.usernameInput, cmd = l.usernameInput.Update(msg)
	} else {
		l.passwordInput, cmd = l.passwordInput.Update(msg)
	}
	return l, cmd
}
