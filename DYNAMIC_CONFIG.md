# Dynamic Configuration Guide

This guide explains how to use the dynamic configuration feature in NanoPony.

## Overview

The dynamic configuration feature allows you to load environment variables without modifying the NanoPony framework code. This is useful when:

- Adding new environment variables without updating the framework
- Supporting custom configurations per application
- Loading feature flags or custom settings

## Usage

### 1. Basic Usage

```go
config := nanopony.NewConfig()

// Load all environment variables with a specific prefix
config.LoadDynamic("CUSTOM_")

// Access the loaded values
if apiKey, exists := config.Dynamic["CUSTOM_API_KEY"]; exists {
    fmt.Printf("API Key: %s\n", apiKey)
}
```

### 2. Loading Specific Prefix

When you have multiple related environment variables, use a prefix to load them all:

```go
// Environment variables:
// CUSTOM_API_URL=https://api.example.com
// CUSTOM_TIMEOUT=30s
// CUSTOM_RETRY_COUNT=3

config := nanopony.BuildConfig()
config.LoadDynamic("CUSTOM_")

// All three variables are now accessible:
fmt.Println(config.Dynamic["CUSTOM_API_URL"])    // https://api.example.com
fmt.Println(config.Dynamic["CUSTOM_TIMEOUT"])    // 30s
fmt.Println(config.Dynamic["CUSTOM_RETRY_COUNT"]) // 3
```

### 3. Loading All Environment Variables

You can load all environment variables (use with caution):

```go
config := nanopony.BuildConfig()
config.LoadDynamic("") // Empty prefix loads everything

// Access any environment variable
fmt.Println(config.Dynamic["HOME"])
fmt.Println(config.Dynamic["PATH"])
```

**Warning**: Loading all environment variables may include sensitive data. Use specific prefixes instead.

### 4. Adding New Environment Variables

When you need to add new environment variables, you don't need to modify NanoPony:

```go
// Just add them to your .env file or environment:
// MY_FEATURE_ENABLED=true
// MY_FEATURE_URL=https://my-feature.example.com
// MY_FEATURE_API_KEY=secret-key

// Then load them:
config := nanopony.NewConfig()
config.LoadDynamic("MY_FEATURE_")

// Use them:
if config.Dynamic["MY_FEATURE_ENABLED"] == "true" {
    // Enable feature
}
```

## Best Practices

### 1. Use Prefixes

Group related environment variables with a common prefix:

```go
// Good
config.LoadDynamic("AUTH_")      // Loads AUTH_API_URL, AUTH_SECRET, etc.
config.LoadDynamic("DATABASE_")  // Loads DATABASE_URL, DATABASE_POOL, etc.

// Avoid
config.LoadDynamic("")  // Loads everything (may include sensitive data)
```

### 2. Check for Existence

Always check if a key exists before using it:

```go
if value, exists := config.Dynamic["MY_KEY"]; exists {
    // Use value
} else {
    // Handle missing configuration
}
```

### 3. Use BuildConfig for Multiple Configurations

If you need multiple configurations with different dynamic loads:

```go
config1 := nanopony.BuildConfig()
config1.LoadDynamic("SERVICE_A_")

config2 := nanopony.BuildConfig()
config2.LoadDynamic("SERVICE_B_")
```

## Example: Complete Setup

```go
package main

import (
    "fmt"
    "github.com/sautmanurung2/nanopony"
)

func main() {
    // Initialize config
    config := nanopony.NewConfig()
    
    // Load custom environment variables
    config.LoadDynamic("MY_APP_")
    
    // Use standard config fields
    fmt.Printf("Environment: %s\n", config.App.Env)
    fmt.Printf("Kafka Model: %s\n", config.App.KafkaModels)
    
    // Use dynamic config fields
    if apiUrl, exists := config.Dynamic["MY_APP_API_URL"]; exists {
        fmt.Printf("API URL: %s\n", apiUrl)
    }
    
    if timeout, exists := config.Dynamic["MY_APP_TIMEOUT"]; exists {
        fmt.Printf("Timeout: %s\n", timeout)
    }
}
```

## Migration Guide

If you currently have hardcoded environment variables and want to make them dynamic:

### Before (Hardcoded)
```go
// You had to modify nanopony code to add new env vars
type MyConfig struct {
    ExistingField string
}
```

### After (Dynamic)
```go
// Just use the Dynamic map
config := nanopony.NewConfig()
config.LoadDynamic("MY_CUSTOM_")

// Access any variable without modifying framework code
config.Dynamic["MY_CUSTOM_NEW_FIELD"]
```

## API Reference

### `LoadDynamic(prefix string)`

Loads environment variables with the given prefix into the `Dynamic` map.

**Parameters:**
- `prefix` (string): The prefix to filter environment variables. If empty, loads all environment variables.

**Example:**
```go
config.LoadDynamic("APP_")  // Loads APP_*, e.g., APP_NAME, APP_VERSION
config.LoadDynamic("")      // Loads all environment variables
```

### `Dynamic map[string]string`

A map containing the dynamically loaded environment variables.

**Example:**
```go
for key, value := range config.Dynamic {
    fmt.Printf("%s = %s\n", key, value)
}
```
