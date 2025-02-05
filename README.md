# CShare - Command Line File Sharing Tool

A modern terminal-based file sharing application with an intuitive interface.

## Features

- 🖥️ Interactive Terminal UI
- 📁 Site Management
  - Create new sites
  - Access existing sites
- 📤 File Operations
  - Upload files with file picker
  - Download files to local system
- 🔐 Secure Authentication
- 🎨 Modern UI with colors and icons

## Installation

### Using Go Install
```bash
go install github.com/kunal697/cshare@latest
```

### Manual Installation
```bash
git clone https://github.com/kunal697/cshare.git
cd cshare
go build
```

## Usage

Simply run:
```bash
cshare
```

### Navigation

- **Arrow Keys** (↑/↓) - Navigate through menus
- **Enter** - Select/Confirm
- **Esc** - Go back/Cancel
- **U** - Upload file (when viewing a site)
- **F** - Open file picker (when uploading)

## Features Guide

1. **Access Existing Site**
   - Enter site name
   - Enter password
   - View and manage files

2. **Create New Site**
   - Choose a site name
   - Set a password
   - Start uploading files

3. **File Management**
   - Upload files using native file picker
   - Download selected files
   - Files are saved in `./downloads` directory

## Dependencies

- github.com/charmbracelet/bubbletea - Terminal UI framework
- github.com/charmbracelet/lipgloss - Styling
- github.com/sqweek/dialog - File picker
- github.com/joho/godotenv - Environment management

## Notes

- Make sure the backend server is running
- Files are downloaded to `./downloads` directory
- Authentication tokens are stored in `.env`
