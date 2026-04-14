package main

import (
	"fmt"

	"github.com/sautmanurung2/nanopony"
)

func main() {
	// Example 1: Using the standard config
	fmt.Println("=== Example 1: Standard Configuration ===")
	nanopony.ResetConfig()
	config := nanopony.NewConfig()

	fmt.Printf("App Env: %s\n", config.App.Env)
	fmt.Printf("Kafka Model: %s\n", config.App.KafkaModels)
	fmt.Printf("Operation: %s\n", config.App.Operation)

	// Example 2: Loading dynamic environment variables
	fmt.Println("\n=== Example 2: Dynamic Configuration ===")
	
	// This will load all environment variables with CUSTOM_ prefix
	// For demonstration, we'll show the concept
	config.LoadDynamic("CUSTOM_")

	// Access dynamic values
	if apiKey, exists := config.Dynamic["CUSTOM_API_KEY"]; exists {
		fmt.Printf("Custom API Key: %s\n", apiKey)
	} else {
		fmt.Println("CUSTOM_API_KEY not set (this is expected in demo)")
	}

	if timeout, exists := config.Dynamic["CUSTOM_TIMEOUT"]; exists {
		fmt.Printf("Custom Timeout: %s\n", timeout)
	} else {
		fmt.Println("CUSTOM_TIMEOUT not set (this is expected in demo)")
	}

	// Example 3: Loading ALL environment variables (use with caution)
	fmt.Println("\n=== Example 3: Load All Environment Variables ===")
	config2 := nanopony.BuildConfig()
	config2.LoadDynamic("") // Empty prefix loads everything

	fmt.Printf("Loaded %d environment variables\n", len(config2.Dynamic))

	// Example 4: Adding new env vars without modifying nanopony code
	fmt.Println("\n=== Example 4: Future-Proof Configuration ===")
	fmt.Println("When you add new environment variables like:")
	fmt.Println("  - MY_NEW_FEATURE_URL=https://api.example.com")
	fmt.Println("  - MY_NEW_FEATURE_TIMEOUT=30s")
	fmt.Println("")
	fmt.Println("Just call: config.LoadDynamic(\"MY_NEW_FEATURE_\")")
	fmt.Println("No need to modify nanopony code!")

	// Demonstrate with actual values if they exist
	config.LoadDynamic("MY_NEW_FEATURE_")
	if len(config.Dynamic) > 0 {
		fmt.Printf("\nLoaded %d custom feature configurations\n", len(config.Dynamic))
		for key, value := range config.Dynamic {
			fmt.Printf("  %s = %s\n", key, value)
		}
	}
}
