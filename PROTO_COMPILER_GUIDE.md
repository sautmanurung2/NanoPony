# NanoPony Proto File Compiler Guide

This guide explains how to use NanoPony's built-in proto file compiler to generate Go code from `.proto` files.

## Overview

The `nanopony` CLI tool provides a convenient way to compile Protocol Buffer (`.proto`) files into Go code (`.pb.go` files) with proper configuration and options.

## Installation

### Option 1: Install Globally (Recommended)

```bash
go install github.com/sautmanurung2/nanopony/cmd/nanopony@latest
```

After installation, you can use `nanopony` command from anywhere.

### Option 2: Build Locally

```bash
cd /path/to/NanoPony/cmd/nanopony
go build -o nanopony .
```

Then use: `./nanopony --proto <file.proto>`

## Prerequisites

Before using the proto compiler, you need to install the following dependencies:

### 1. Protocol Buffers Compiler (protoc)

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install protobuf-compiler
protoc --version  # Verify installation
```

**macOS:**
```bash
brew install protobuf
protoc --version  # Verify installation
```

**Windows:**
1. Download from [Protocol Buffers Releases](https://github.com/protocolbuffers/protobuf/releases)
2. Extract and add to PATH

### 2. Go Protocol Buffers Plugin

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

### 3. Go gRPC Plugin (Optional, for gRPC services)

```bash
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

This is only needed if your `.proto` file defines gRPC services.

## Basic Usage

### Simple Proto File

Given a proto file `user.proto`:

```protobuf
syntax = "proto3";

package user;

option go_package = "./pb;userpb";

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}
```

Compile it:

```bash
nanopony --proto user.proto
```

This generates `user.pb.go` in the same directory.

### With Output Directory

```bash
nanopony --proto user.proto --output ./gen
```

This generates the file in `./gen/user.pb.go`.

### With Import Path

If your proto imports other proto files:

```bash
nanopony --proto service.proto -I ./protos --output ./gen
```

### With Custom Go Package

```bash
nanopony --proto user.proto --go_package github.com/example/api/user --output ./gen
```

## Command Reference

```bash
nanopony [flags]
```

### Flags

| Flag | Aliases | Required | Description | Default |
|------|---------|----------|-------------|---------|
| `--proto` | - | ✅ | Path to `.proto` file | - |
| `--proto_path` | `-I` | ❌ | Directory to search for imports | Proto file directory |
| `--output` | - | ❌ | Output directory for `.pb.go` | Proto file directory |
| `--go_package` | - | ❌ | Go package path | `.;<filename>` |
| `--help` | `-h` | ❌ | Show help message | - |

## Examples

### Example 1: Simple Message Types

**proto/messages.proto:**
```protobuf
syntax = "proto3";

package messages;

option go_package = "./pb;messages";

message CreateUserRequest {
  string name = 1;
  string email = 2;
  int32 age = 3;
}

message CreateUserResponse {
  string id = 1;
  bool success = 2;
}
```

**Command:**
```bash
nanopony --proto proto/messages.proto --output pb
```

**Output:**
- `pb/messages.pb.go`

### Example 2: With gRPC Services

**proto/user_service.proto:**
```protobuf
syntax = "proto3";

package userservice;

option go_package = "./pb;userservice";

import "google/protobuf/empty.proto";

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}

message UserId {
  string id = 1;
}

service UserService {
  rpc GetUser (UserId) returns (User);
  rpc ListUsers (google.protobuf.Empty) returns (stream User);
  rpc CreateUser (User) returns (User);
  rpc DeleteUser (UserId) returns (google.protobuf.Empty);
}
```

**Command:**
```bash
nanopony --proto proto/user_service.proto --output pb
```

**Output:**
- `pb/user_service.pb.go` - Message types
- `pb/user_service_grpc.pb.go` - gRPC service stubs

### Example 3: Multiple Proto Files with Imports

**Directory structure:**
```
protos/
├── common/
│   └── common.proto
├── user/
│   └── user.proto
└── order/
    └── order.proto
```

**protos/user/user.proto:**
```protobuf
syntax = "proto3";

package user;

option go_package = "./pb;userpb";

import "common/common.proto";

message User {
  string id = 1;
  common.Metadata metadata = 2;
  string name = 3;
}
```

**Command:**
```bash
nanopony --proto protos/user/user.proto -I ./protos --output gen/user
```

### Example 4: Batch Compilation (Shell Script)

Create a script `generate.sh`:

```bash
#!/bin/bash

# Clean old generated files
rm -rf ./gen/*

# Create output directory
mkdir -p ./gen

# Compile all proto files
find ./protos -name "*.proto" | while read -r proto_file; do
  echo "Compiling: $proto_file"
  nanopony --proto "$proto_file" -I ./protos --output ./gen
done

echo "All proto files compiled successfully!"
```

**Make executable and run:**
```bash
chmod +x generate.sh
./generate.sh
```

## Generated Files

### Without gRPC Services
- `*.pb.go` - Contains Go structs for protobuf messages

### With gRPC Services
- `*.pb.go` - Contains Go structs for protobuf messages
- `*_grpc.pb.go` - Contains gRPC client and server stubs

## Best Practices

### 1. Organize Proto Files

```
protos/
├── v1/
│   ├── common/
│   │   └── common.proto
│   ├── user/
│   │   └── user.proto
│   └── order/
│       └── order.proto
└── v2/
    └── ...
```

### 2. Use Consistent Go Package Names

```protobuf
option go_package = "github.com/yourorg/api/user/v1;userv1";
```

### 3. Version Your APIs

```protobuf
option go_package = "github.com/yourorg/api/user/v1;userv1";
```

### 4. Separate Generated Code

```bash
nanopony --proto proto/user.proto --output internal/gen/user
```

### 5. Use Makefile for Automation

```makefile
.PHONY: proto

PROTO_DIR = protos
GEN_DIR = internal/gen

proto:
	@echo "Generating proto files..."
	@find $(PROTO_DIR) -name "*.proto" | while read proto; do \
		nanopony --proto $$proto -I $(PROTO_DIR) --output $(GEN_DIR); \
	done
	@echo "Done!"
```

## Troubleshooting

### Error: "protoc is not installed"

**Solution:**
```bash
# Ubuntu/Debian
sudo apt install protobuf-compiler

# macOS
brew install protobuf
```

### Error: "protoc-gen-go is not installed"

**Solution:**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Make sure Go bin is in your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Error: "File not found" when importing

**Solution:**
Use the `-I` flag to specify import paths:
```bash
nanopony --proto service.proto -I ./protos -I ./protos/common
```

### Error: "go_package option is missing"

**Solution:**
Add `go_package` option to your `.proto` file:
```protobuf
option go_package = "./pb;packagename";
```

Or specify via command line:
```bash
nanopony --proto service.proto --go_package "./pb;packagename"
```

### Generated code has wrong package name

**Solution:**
Check the `go_package` option format:
```protobuf
// Format: "import_path;package_name"
option go_package = "github.com/example/api;apipb";
```

## Integration with NanoPony Framework

After generating your protobuf files, you can integrate them with the NanoPony framework:

```go
package main

import (
    "context"
    "log"
    
    "github.com/sautmanurung2/nanopony"
    "github.com/yourorg/api/pb"
)

func main() {
    // Initialize framework
    config := nanopony.NewConfig()
    
    framework := nanopony.NewFramework().
        WithConfig(config).
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(5, 100)
    
    components := framework.Build()
    
    // Use your generated protobuf messages
    user := &pb.User{
        Id:    "123",
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    // Produce to Kafka
    producer := components.Producer()
    producer.Produce("users", user)
}
```

## Additional Resources

- [Protocol Buffers Documentation](https://developers.google.com/protocol-buffers)
- [protoc-gen-go Documentation](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [NanoPony Framework Documentation](../DOKUMENTASI.md)
