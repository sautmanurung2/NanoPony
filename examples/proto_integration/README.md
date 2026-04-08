# NanoPony Proto Integration Example

This example demonstrates how to use generated protobuf code with the NanoPony framework.

## Step 1: Create Proto File

Create `protos/user.proto`:

```protobuf
syntax = "proto3";

package user;

option go_package = "github.com/sautmanurung2/nanopony/examples/proto_integration/pb;userpb";

import "google/protobuf/timestamp.proto";

// User represents a system user
message User {
  string id = 1;
  string name = 2;
  string email = 3;
  int32 age = 4;
  repeated string roles = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}

// UserList is a collection of users
message UserList {
  repeated User users = 1;
  int32 total = 2;
  int32 page = 3;
  int32 page_size = 4;
}

// Event represents a user event (for Kafka)
message UserEvent {
  string id = 1;
  string user_id = 2;
  EventType type = 3;
  google.protobuf.Timestamp timestamp = 4;
  map<string, string> metadata = 5;
}

enum EventType {
  EVENT_TYPE_UNSPECIFIED = 0;
  EVENT_TYPE_CREATED = 1;
  EVENT_TYPE_UPDATED = 2;
  EVENT_TYPE_DELETED = 3;
}

// UserService provides user operations
service UserService {
  rpc GetUser (GetUserRequest) returns (UserResponse);
  rpc ListUsers (ListUsersRequest) returns (UserListResponse);
  rpc CreateUser (CreateUserRequest) returns (UserResponse);
  rpc UpdateUser (UpdateUserRequest) returns (UserResponse);
  rpc DeleteUser (DeleteUserRequest) returns (DeleteUserResponse);
  rpc StreamUsers (StreamUsersRequest) returns (stream User);
}

message GetUserRequest {
  string id = 1;
}

message ListUsersRequest {
  int32 page = 1;
  int32 page_size = 2;
  string search = 3;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  int32 age = 3;
  repeated string roles = 4;
}

message UpdateUserRequest {
  string id = 1;
  string name = 2;
  string email = 3;
  int32 age = 4;
  repeated string roles = 5;
}

message DeleteUserRequest {
  string id = 1;
}

message UserResponse {
  User user = 1;
  bool success = 2;
  string message = 3;
}

message UserListResponse {
  UserList users = 1;
  bool success = 2;
}

message DeleteUserResponse {
  bool success = 1;
  string message = 2;
}

message StreamUsersRequest {
  bool include_deleted = 1;
}
```

## Step 2: Generate Go Code

```bash
# Install dependencies (if not already installed)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
nanopony --proto protos/user.proto -I ./protos --output pb
```

This generates:
- `pb/user.pb.go` - Message types
- `pb/user_grpc.pb.go` - gRPC service stubs

## Step 3: Use with NanoPony Framework

Create `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sautmanurung2/nanopony"
	"github.com/sautmanurung2/nanopony/examples/proto_integration/pb"
)

func main() {
	// Initialize NanoPony framework
	config := nanopony.NewConfig()

	framework := nanopony.NewFramework().
		WithConfig(config).
		WithDatabase().
		WithKafkaWriter().
		WithProducer().
		WithWorkerPool(5, 100).
		WithPoller(nanopony.DefaultPollerConfig(), &UserFetcher{}).
		AddService(&UserService{})

	components := framework.Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start framework
	components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
		log.Printf("Processing job: %s", job.ID)
		return nil
	})

	// Example: Create user event and produce to Kafka
	userEvent := &pb.UserEvent{
		Id:        "evt-001",
		UserId:    "user-001",
		Type:      pb.EventType_EVENT_TYPE_CREATED,
		Timestamp: timestamppb.Now(),
		Metadata: map[string]string{
			"source": "api",
			"region": "us-east-1",
		},
	}

	// Serialize and produce to Kafka
	protoBytes, err := proto.Marshal(userEvent)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	producer := components.Producer()
	success, err := producer.Produce("user-events", protoBytes)
	if err != nil {
		log.Printf("Failed to produce event: %v", err)
		return
	}

	if success {
		log.Printf("Event produced successfully: %s", userEvent.Id)
	}

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	components.Shutdown(ctx)
}

// UserFetcher implements DataFetcher interface
type UserFetcher struct{}

func (f *UserFetcher) Fetch() ([]interface{}, error) {
	// Fetch users from database and convert to protobuf
	users := []*pb.User{
		{
			Id:    "user-001",
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   30,
			Roles: []string{"admin", "user"},
		},
		{
			Id:    "user-002",
			Name:  "Jane Smith",
			Email: "jane@example.com",
			Age:   25,
			Roles: []string{"user"},
		},
	}

	var result []interface{}
	for _, user := range users {
		result = append(result, user)
	}

	return result, nil
}

// UserService implements a gRPC service
type UserService struct {
	pb.UnimplementedUserServiceServer
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	// Implementation here
	user := &pb.User{
		Id:    req.Id,
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
		Roles: []string{"admin", "user"},
	}

	return &pb.UserResponse{
		User:    user,
		Success: true,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.UserListResponse, error) {
	// Implementation here
	users := []*pb.User{
		{
			Id:    "user-001",
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	return &pb.UserListResponse{
		Users: &pb.UserList{
			Users:    users,
			Total:    1,
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		Success: true,
	}, nil
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	// Implementation here
	user := &pb.User{
		Id:    fmt.Sprintf("user-%d", time.Now().Unix()),
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
		Roles: req.Roles,
	}

	return &pb.UserResponse{
		User:    user,
		Success: true,
		Message: "User created successfully",
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	// Implementation here
	return &pb.UserResponse{
		Success: true,
		Message: "User updated successfully",
	}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	// Implementation here
	return &pb.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

func (s *UserService) StreamUsers(req *pb.StreamUsersRequest, stream pb.UserService_StreamUsersServer) error {
	// Implementation here
	users := []*pb.User{
		{
			Id:    "user-001",
			Name:  "John Doe",
			Email: "john@example.com",
		},
		{
			Id:    "user-002",
			Name:  "Jane Smith",
			Email: "jane@example.com",
		},
	}

	for _, user := range users {
		if err := stream.Send(user); err != nil {
			return err
		}
	}

	return nil
}
```

## Step 4: Run the Example

```bash
# Build
go build -o nanopony-proto-example .

# Run (requires Kafka and Oracle to be configured)
./nanopony-proto-example
```

## Directory Structure

```
examples/proto_integration/
├── protos/
│   └── user.proto           # Protocol buffer definitions
├── pb/                       # Generated code (git ignored)
│   ├── user.pb.go
│   └── user_grpc.pb.go
├── main.go                   # Example application
├── go.mod                    # Go module
└── README.md                 # This file
```

## go.mod

```go
module github.com/sautmanurung2/nanopony/examples/proto_integration

go 1.25.1

require (
	github.com/sautmanurung2/nanopony v0.1.0
	google.golang.org/protobuf v1.33.0
	google.golang.org/grpc v1.62.0
)

replace github.com/sautmanurung2/nanopony => ../..
```

## Makefile

```makefile
.PHONY: proto build run clean

# Generate proto files
proto:
	@echo "Generating proto files..."
	@nanopony --proto protos/user.proto -I ./protos --output pb
	@echo "Done!"

# Build
build:
	@go build -o nanopony-proto-example .

# Run
run: build
	@./nanopony-proto-example

# Clean
clean:
	@rm -f nanopony-proto-example
	@rm -rf pb/*.go

# Help
help:
	@echo "Available targets:"
	@echo "  proto  - Generate proto files"
	@echo "  build  - Build the example"
	@echo "  run    - Build and run"
	@echo "  clean  - Clean build artifacts"
	@echo "  help   - Show this help"
```

## Key Points

1. **Use Generated Types**: Import and use generated protobuf types directly
2. **Serialization**: Use `proto.Marshal()` to serialize for Kafka
3. **Deserialization**: Use `proto.Unmarshal()` to deserialize from Kafka
4. **Type Safety**: Benefit from compile-time type checking
5. **gRPC Integration**: Generated service stubs work with standard gRPC

## Benefits

✅ **Type Safety** - Compile-time type checking
✅ **Schema Evolution** - Backward/forward compatible changes
✅ **Language Agnostic** - Same schema for different services
✅ **Performance** - Efficient binary serialization
✅ **Code Generation** - Less boilerplate code
✅ **Documentation** - Schema serves as documentation

## Additional Resources

- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)
- [gRPC Go Tutorial](https://grpc.io/docs/languages/go/)
- [NanoPony Documentation](../../DOKUMENTASI.md)
