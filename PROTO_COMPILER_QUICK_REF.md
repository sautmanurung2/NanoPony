# NanoPony Proto Compiler - Quick Reference

## One-Line Commands

```bash
# Install
go install github.com/sautmanurung2/nanopony/cmd/nanopony@latest

# Basic usage
nanopony --proto <file.proto>

# With output dir
nanopony --proto <file.proto> --output ./gen

# With import path
nanopony --proto <file.proto> -I ./protos --output ./gen
```

## Proto File Template

```protobuf
syntax = "proto3";

package mypackage;

option go_package = "./pb;myPackage";

message MyMessage {
  string id = 1;
  string name = 2;
}
```

## Install Dependencies

```bash
# Ubuntu/Debian
sudo apt install protobuf-compiler

# macOS
brew install protobuf

# Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Flags Reference

| Flag | Description | Example |
|------|-------------|---------|
| `--proto` | Proto file path | `--proto service.proto` |
| `--proto_path`, `-I` | Import search path | `-I ./protos` |
| `--output` | Output directory | `--output ./gen` |
| `--go_package` | Go package path | `--go_package github.com/api` |
| `--help`, `-h` | Show help | `--help` |

## Common Patterns

### Simple Messages
```bash
nanopony --proto messages.proto --output pb
```

### With gRPC Services
```bash
nanopony --proto service.proto --output pb
# Generates: service.pb.go + service_grpc.pb.go
```

### Multiple Files
```bash
find ./protos -name "*.proto" -exec nanopony --proto {} -I ./protos --output ./gen \;
```

### Batch Script
```bash
#!/bin/bash
rm -rf ./gen/*
mkdir -p ./gen
find ./protos -name "*.proto" | while read proto; do
  nanopony --proto "$proto" -I ./protos --output ./gen
done
```

## Generated Files

- `*.pb.go` - Message types
- `*_grpc.pb.go` - gRPC services (if protoc-gen-go-grpc installed)

## Troubleshooting

| Error | Solution |
|-------|----------|
| "protoc is not installed" | Install protobuf-compiler |
| "protoc-gen-go is not installed" | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` |
| "File not found" | Use `-I` flag for import paths |
| "go_package option is missing" | Add `option go_package` to .proto or use `--go_package` flag |

## Full Documentation

See [PROTO_COMPILER_GUIDE.md](PROTO_COMPILER_GUIDE.md) for complete guide.
