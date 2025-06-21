package main

import (
	"fmt"
	"os"

	"github.com/renancavalcantercb/healthcheck-cli/internal/app"
)

// version will be set during build
var version = "dev"

// main is the entry point of the application
func main() {
	app, err := app.NewApplicationWithDefaults()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	
	// Setup graceful shutdown
	defer func() {
		if err := app.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Shutdown error: %v\n", err)
		}
	}()

	rootCmd := setupCommands(app)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Command execution failed: %v\n", err)
		os.Exit(1)
	}
}

// showVersion displays version information
func showVersion() {
	fmt.Printf("healthcheck version %s\n", version)
}