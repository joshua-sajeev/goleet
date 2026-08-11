# GoLeet

A terminal UI for tracking LeetCode problems with spaced-repetition reviews in an Obsidian vault.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [huh](https://github.com/charmbracelet/huh), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [godotenv](https://github.com/joho/godotenv).

## Features

* Add LeetCode problems to your Obsidian vault
* Review problems that are due
* Review a specific problem
* Rate your confidence after each review
* Automatically schedule the next review
* Store problem metadata in Markdown frontmatter
* Terminal UI with keyboard navigation
* Use `.env` for the default vault path
* Override the vault path with `-vault`

## Setup

Clone the repository and install dependencies:

```bash
go mod tidy
```

Create a `.env` file in the project root:

```env
VAULT_DIR=/home/joshua/Second-Brain/02 - Fleeting/Leetcode
```

Build:

```bash
go build -o goleet .
```

Run:

```bash
./goleet
```

Or run directly with Go:

```bash
go run .
```

### Override vault path

The `-vault` flag overrides `VAULT_DIR`:

```bash
./goleet -vault "/path/to/another/vault"
```

Add `.env` to `.gitignore`:

```gitignore
.env
```

## Usage

```text
1  New Problem
2  Review Due Problems
3  Review Specific Problem
0  Quit
```

Inside forms:

* `↑` / `↓` or `j` / `k` — navigate
* `Enter` — select
* `Tab` / `Shift+Tab` — switch fields
* `Esc` — cancel
* `Ctrl+C` — quit

## Project Structure

```text
goleet/
├── main.go
├── model.go
├── vault.go
├── logic.go
├── forms.go
├── go.mod
├── go.sum
└── .env
```

| File       | Purpose                                       |
| ---------- | --------------------------------------------- |
| `main.go`  | CLI flags, configuration, application startup |
| `model.go` | Bubble Tea model and application state        |
| `vault.go` | Obsidian/Markdown file handling               |
| `logic.go` | Spaced-repetition calculations                |
| `forms.go` | huh forms                                     |

## Vault

Problems are stored as Markdown files in the configured Obsidian directory.

Example:

```markdown
---
leetcode: 217
title: Contains Duplicate
difficulty: Easy
next_review: 2026-08-15
confidence: 4
---

# Contains Duplicate

...
```

Since the files are standard Markdown, they can be edited normally in Obsidian.

## Development

Format:

```bash
go fmt ./...
```

Run tests:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

Build:

```bash
go build .
```
