package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "new-project" {
		fmt.Println("Usage: nanopony new-project <project-name>")
		os.Exit(1)
	}

	projectName := os.Args[2]
	modName := fmt.Sprintf("github.com/user/%s", projectName)

	fmt.Printf("🐎 Generating NanoPony project: %s...\n", projectName)

	// Requested directory structure
	dirs := []string{
		"src/constant",
		"src/converter",
		"src/infrastructure",
		"src/interfaces",
		"src/models",
		"src/proto",
		"src/repository",
		"src/service",
		"src/utils",
	}

	for _, d := range dirs {
		path := filepath.Join(projectName, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	files := map[string]string{
		"src/constant/main.go": `// this is package for constants
package constant
`,
		"src/converter/main.go": `// this is package for data converters/mappers
package converter
`,
		"src/infrastructure/main.go": `// this is package for infrastructure setup (DB, Kafka, etc.)
package infrastructure
`,
		"src/interfaces/main.go": `// this is package for interfaces/contracts
package interfaces
`,
		"src/models/main.go": `// this is package for data models/entities
package models
`,
		"src/proto/main.go": `// this is package for protobuf definitions
package proto
`,
		"src/repository/main.go": `// this is package for data access implementation
package repository
`,
		"src/service/main.go": `// this is package for business logic services
package service
`,
		"src/utils/main.go": `// this is package for utility functions
package utils
`,
		"main.go": `package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/sautmanurung2/nanopony/pkg"
)

func main() {
    // 1. Setup NanoPony Config
    cfg := nanopony.NewConfig()
    
    // 2. Build Infrastructure via NanoPony
    framework := nanopony.NewFramework().
        WithConfig(cfg).
        WithDatabase().      // Setup Oracle
        WithKafkaWriter().   // Setup Kafka
        WithProducer().
        WithWorkerPool(5, 100)

    components := framework.Build()

    // 3. Start Framework with Job Handler
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
        log.Printf("Processing Job ID: %s", job.ID)
        return nil
    })

    log.Println("🐎 NanoPony App Started!")

    // Graceful Shutdown
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig

    log.Println("🛑 Shutting down...")
    components.Shutdown(ctx)
}
`,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectName, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			fmt.Printf("Error writing file %s: %v\n", fullPath, err)
			os.Exit(1)
		}
	}

	// Run go commands
	os.Chdir(projectName)
	exec.Command("go", "mod", "init", modName).Run()
	fmt.Println("📦 Installing NanoPony and dependencies...")
	// Note: User will need to point to /pkg for library imports
	exec.Command("go", "get", "github.com/sautmanurung2/nanopony/pkg@latest").Run()
	exec.Command("go", "mod", "tidy").Run()

	fmt.Printf("✨ Project %s is ready!\n", projectName)
}
