package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	defaultPath := os.Getenv("VAULT_DIR")

	vaultDir := flag.String(
		"vault",
		defaultPath,
		"Path to Obsidian LeetCode directory",
	)

	flag.Parse()

	if *vaultDir == "" {
		fmt.Println("VAULT_DIR is not set")
		os.Exit(1)
	}

	if err := ensureVaultDir(*vaultDir); err != nil {
		fmt.Printf("Failed to create/verify vault directory: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		initialModel(*vaultDir),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
