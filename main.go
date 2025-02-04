package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"github.com/sqweek/dialog"
)

// Model represents the application's state.
type Model struct {
	cursor      int
	selectedIdx int
	siteName    string
	password    string
	files       []FileInfo
	state       string
	errorMsg    string
	authToken   string
	uploadPath  string
	fileToUpload string
}

type FileInfo struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
}

// Update the style definitions
var (
	appStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C")).
		Width(80)

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00")).
		Background(lipgloss.Color("#1A1A1A")).
		Width(76).
		Align(lipgloss.Center).
		Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
		Padding(1, 2)

	menuBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C")).
		Padding(1, 2).
		Width(70)

	inputBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C")).
		Padding(1, 2).
		Width(70)

	fileListStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C")).
		Padding(1, 2).
		Width(70)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Background(lipgloss.Color("#1A1A1A")).
		Width(76).
		Align(lipgloss.Left).
		Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Padding(0, 2)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Padding(0, 2)

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Bold(true)

	highlightStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")) // Gold
)

// Update the view states
const (
	stateMenu       = "menu"
	stateSiteName   = "siteName"
	statePassword   = "password"
	stateCreateSiteName = "createSiteName"    // New state for site creation name
	stateCreatePassword = "createPassword"    // New state for site creation password
	stateViewFiles  = "viewFiles"
	stateUploadFile = "uploadFile"
)

// Add file dialog support
type fileSelectMsg struct {
	path string
	err  error
}

// Init initializes the model (required by Bubble Tea).
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			return handleMenuInput(m, msg)
		case stateSiteName:
			return handleSiteNameInput(m, msg)
		case statePassword:
			return handlePasswordInput(m, msg)
		case stateCreateSiteName:
			return handleCreateSiteNameInput(m, msg)
		case stateCreatePassword:
			return handleCreatePasswordInput(m, msg)
		case stateViewFiles:
			return handleFileSelection(m, msg)
		case stateUploadFile:
			return handleUploadSelectInput(m, msg)
		}
	case []FileInfo:
		m.files = msg
		m.state = stateViewFiles
	case error:
		m.state = stateMenu
		m.errorMsg = msg.Error()
	case string:
		if strings.HasPrefix(msg, "Success") {
			m.errorMsg = ""
			m.state = stateMenu
		} else {
			m.errorMsg = msg
		}
	case fileSelectMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Error selecting file: %v", msg.err)
		} else {
			m.fileToUpload = msg.path
		}
	}
	return m, nil
}

// View renders the UI based on the current state.
func (m *Model) View() string {
	var content strings.Builder

	// Header
	header := headerStyle.Render("FileShare CLI")
	content.WriteString(header)
	content.WriteString("\n")

	// Error/Success message
	if m.errorMsg != "" {
		var msgBox string
		if strings.HasPrefix(m.errorMsg, "Success") {
			msgBox = successStyle.Render("✅ " + m.errorMsg)
		} else {
			msgBox = errorStyle.Render("❌ " + m.errorMsg)
		}
		content.WriteString(msgBox)
		content.WriteString("\n")
	}

	// Main content
	switch m.state {
	case stateMenu:
		menu := menuBoxStyle.Render(renderMenu(m.cursor))
		content.WriteString(menu)

	case stateSiteName:
		inputBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"Enter Site Name",
				m.siteName+"█",
				"",
				highlightStyle.Render("Enter - Continue • Esc - Back"),
			),
		)
		content.WriteString(inputBox)

	case statePassword:
		inputBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"Site: "+m.siteName,
				"Password: "+strings.Repeat("•", len(m.password))+"█",
				"",
				highlightStyle.Render("Enter - Continue • Esc - Back"),
			),
		)
		content.WriteString(inputBox)

	case stateCreateSiteName:
		inputBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"Create New Site",
				"Enter Site Name: " + m.siteName + "█",
				"",
				highlightStyle.Render("Enter - Continue • Esc - Back"),
			),
		)
		content.WriteString(inputBox)

	case stateCreatePassword:
		inputBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"Create Site: " + m.siteName,
				"Enter Password: " + strings.Repeat("•", len(m.password)) + "█",
				"",
				highlightStyle.Render("Enter - Create Site • Esc - Back"),
			),
		)
		content.WriteString(inputBox)

	case stateViewFiles:
		fileBox := fileListStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"�� "+m.siteName,
				strings.Repeat("─", 50),
				renderFileList(*m),
				"",
				highlightStyle.Render("U - Upload • Enter - Download • Esc - Back"),
			),
		)
		content.WriteString(fileBox)

	case stateUploadFile:
		uploadBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"📤 Upload to: "+m.siteName,
				"",
				"Press F to select file",
				m.fileToUpload,
				"",
				highlightStyle.Render("Enter - Upload • Esc - Cancel"),
			),
		)
		content.WriteString(uploadBox)
	}

	// Status bar
	statusBar := statusBarStyle.Render(getStatusText(*m))
	content.WriteString("\n" + statusBar)

	// Wrap everything in the app container
	return appStyle.Render(content.String())
}

// handleMenuInput handles input in the menu state.
func handleMenuInput(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down":
		if m.cursor < 2 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0:
			m.state = stateSiteName
			m.siteName = ""
			m.password = ""
		case 1:
			m.state = stateCreateSiteName
			m.siteName = ""
			m.password = ""
		case 2:
			return m, tea.Quit
		}
	}
	return m, nil
}

// handleSiteNameInput handles input in the siteName state.
func handleSiteNameInput(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.state = statePassword
	case "esc":
		m.state = stateMenu
		m.siteName = ""
	case "backspace":
		if len(m.siteName) > 0 {
			m.siteName = m.siteName[:len(m.siteName)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.siteName += msg.String()
		}
	}
	return m, nil
}

// handlePasswordInput handles input in the password state.
func handlePasswordInput(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m, fetchFiles(m.siteName, m.password)
	case "esc":
		m.state = stateMenu
		m.password = ""
	case "backspace":
		if len(m.password) > 0 {
			m.password = m.password[:len(m.password)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.password += msg.String()
		}
	}
	return m, nil
}

// handleCreateSiteNameInput handles input in the createSiteName state.
func handleCreateSiteNameInput(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.siteName != "" {
			m.state = stateCreatePassword
		}
	case "esc":
		m.state = stateMenu
		m.siteName = ""
	case "backspace":
		if len(m.siteName) > 0 {
			m.siteName = m.siteName[:len(m.siteName)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.siteName += msg.String()
		}
	}
	return m, nil
}

// handleCreatePasswordInput handles input in the createPassword state.
func handleCreatePasswordInput(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.siteName == "" || m.password == "" {
			return m, nil
		}
		return m, createSite(m.siteName, m.password)
	case "esc":
		m.state = stateCreateSiteName
		m.password = ""
	case "backspace":
		if len(m.password) > 0 {
			m.password = m.password[:len(m.password)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.password += msg.String()
		}
	}
	return m, nil
}

// handleUploadSelectInput handles input in the uploadSelect state.
func handleUploadSelectInput(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f", "F":
		return m, openFileDialog
	case "enter":
		if m.fileToUpload != "" {
			return m, uploadFile(m)
		}
	case "esc":
		m.state = stateViewFiles
		m.fileToUpload = ""
	}
	return m, nil
}

// handleFileSelection allows users to select a file using arrow keys.
func handleFileSelection(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "u", "U":
		m.state = stateUploadFile
		m.fileToUpload = ""
	case "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down":
		if m.selectedIdx < len(m.files)-1 {
			m.selectedIdx++
		}
	case "enter":
		if len(m.files) > 0 && m.selectedIdx >= 0 && m.selectedIdx < len(m.files) {
			selectedFile := m.files[m.selectedIdx]
			return m, downloadFile(selectedFile.ID, selectedFile.FileName)
		}
	case "esc":
		m.state = stateMenu
		m.selectedIdx = 0
	}
	return m, nil
}

// renderMenu renders the menu UI.
func renderMenu(cursor int) string {
	menuItems := []string{
		"📂  Access Existing Site",
		"✨  Create New Site",
		"🚪  Exit Application",
	}
	var menu strings.Builder

	menu.WriteString("Main Menu\n")
	menu.WriteString(strings.Repeat("─", 40))
	menu.WriteString("\n\n")

	for i, item := range menuItems {
		if i == cursor {
			menu.WriteString(selectedStyle.Render("➜  " + item))
		} else {
			menu.WriteString("   " + item)
		}
		menu.WriteString("\n")
	}

	return menu.String()
}

// fetchFiles fetches files from the server and stores the auth token.
func fetchFiles(siteName, password string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("http://localhost:8080/site/%s?password=%s", siteName, password)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("error connecting to server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to fetch site: %s (status code: %d)", string(body), resp.StatusCode)
		}

		var result struct {
			AuthToken string     `json:"auth_token"`
			Files     []FileInfo `json:"files"`
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading server response: %v", err)
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("error parsing server response: %v", err)
		}

		// Store auth token in .env file
		err = godotenv.Load()
		if err != nil {
			// If .env doesn't exist, create it
			f, err := os.Create(".env")
			if err != nil {
				return fmt.Errorf("error creating .env file: %v", err)
			}
			f.Close()
		}
		
		err = os.Setenv("auth_token", result.AuthToken)
		if err != nil {
			return fmt.Errorf("error saving auth token: %v", err)
		}

		// Return empty slice if no files, don't return error
		return result.Files
	}
}

// createSite creates a new site on the server.
func createSite(siteName, password string) tea.Cmd {
	return func() tea.Msg {
		// Prepare request data
		data := map[string]string{
			"site_name": siteName,
			"password": password,
		}
		
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("error preparing request: %v", err)
		}

		// Create request
		req, err := http.NewRequest("POST", "http://localhost:8080/createsite", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error connecting to server: %v", err)
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response: %v", err)
		}

		// Check response status
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create site: %s", string(body))
		}

		// Parse response
		var result struct {
			Message    string `json:"message"`
			AuthToken string `json:"auth_token"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("error parsing response: %v", err)
		}

		// Save auth token to .env file
		f, err := os.Create(".env")
		if err != nil {
			return fmt.Errorf("error creating .env file: %v", err)
		}
		defer f.Close()

		_, err = f.WriteString(fmt.Sprintf("auth_token=%s\n", result.AuthToken))
		if err != nil {
			return fmt.Errorf("error writing auth token: %v", err)
		}

		return "Success: Site created successfully!"
	}
}

// downloadFile fetches the selected file from the server.
func downloadFile(fileID int, fileName string) tea.Cmd {
	return func() tea.Msg {
		// Load auth token from .env file
		err := godotenv.Load()
		if err != nil {
			return fmt.Errorf("error loading .env file: %v", err)
		}

		authToken := os.Getenv("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is missing")
		}

		// Create the download request
		url := fmt.Sprintf("http://localhost:8080/getfile/%d", fileID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		// Add authorization token to the request header
		req.Header.Set("Authorization", authToken)

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error downloading file: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to download file: %s", string(body))
		}

		// Parse the response
		var result struct {
			Message string `json:"message"`
			File    string `json:"file"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("error parsing response: %v", err)
		}

		// Create downloads directory if it doesn't exist
		err = os.MkdirAll("downloads", 0755)
		if err != nil {
			return fmt.Errorf("error creating downloads directory: %v", err)
		}

		// Save the file
		downloadPath := filepath.Join("downloads", fileName)
		err = os.WriteFile(downloadPath, []byte(result.File), 0644)
		if err != nil {
			return fmt.Errorf("error saving file: %v", err)
		}

		return fmt.Sprintf("Success: File downloaded to %s", downloadPath)
	}
}

// uploadFile uploads a file to the server.
func uploadFile(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.fileToUpload == "" {
			return fmt.Errorf("no file selected")
		}

		file, err := os.Open(m.fileToUpload)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer file.Close()

		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file to form
		part, err := writer.CreateFormFile("file", filepath.Base(m.fileToUpload))
		if err != nil {
			return fmt.Errorf("error creating form file: %v", err)
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return fmt.Errorf("error copying file content: %v", err)
		}

		err = writer.Close()
		if err != nil {
			return fmt.Errorf("error closing writer: %v", err)
		}

		// Create request
		url := fmt.Sprintf("http://localhost:8080/upload/%s", m.siteName)
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		// Load auth token
		err = godotenv.Load()
		if err != nil {
			return fmt.Errorf("error loading .env file: %v", err)
		}

		authToken := os.Getenv("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is missing")
		}

		// Set headers
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", authToken)

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error uploading file: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to upload file: %s", string(bodyBytes))
		}

		// After successful upload, refresh the file list
		files, err := fetchFilesDirectly(m.siteName, m.password)
		if err != nil {
			return fmt.Errorf("file uploaded but error refreshing list: %v", err)
		}
		m.files = files
		return "Success: File uploaded successfully!"
	}
}

// Add helper function to fetch files directly
func fetchFilesDirectly(siteName, password string) ([]FileInfo, error) {
	url := fmt.Sprintf("http://localhost:8080/site/%s?password=%s", siteName, password)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch site: %s", string(body))
	}

	var result struct {
		AuthToken string     `json:"auth_token"`
		Files     []FileInfo `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result.Files, nil
}

// Update openFileDialog to use dialog package
func openFileDialog() tea.Msg {
	filename, err := dialog.File().Load()
	if err != nil {
		if err == dialog.Cancelled {
			return fileSelectMsg{path: "", err: nil}
		}
		return fileSelectMsg{path: "", err: err}
	}
	return fileSelectMsg{path: filename, err: nil}
}

// Update renderFileList function
func renderFileList(m Model) string {
	var files strings.Builder
	if len(m.files) == 0 {
		return "No files found. Press U to upload a file."
	}

	for i, file := range m.files {
		prefix := "   "
		if i == m.selectedIdx {
			prefix = "➜  "
			files.WriteString(selectedStyle.Render(prefix + file.FileName))
		} else {
			files.WriteString(prefix + file.FileName)
		}
		files.WriteString("\n")
	}
	return files.String()
}

// Add helper function for status text
func getStatusText(m Model) string {
	switch m.state {
	case stateMenu:
		return "Use ↑/↓ to navigate, Enter to select"
	case stateViewFiles:
		return fmt.Sprintf("Files: %d | Site: %s", len(m.files), m.siteName)
	default:
		return "FileShare CLI"
	}
}

// main is the entry point of the application.
func main() {
	p := tea.NewProgram(
		&Model{state: stateMenu},
		tea.WithAltScreen(),       // Use alternate screen
		tea.WithMouseCellMotion(), // Enables mouse support
	)
	
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
