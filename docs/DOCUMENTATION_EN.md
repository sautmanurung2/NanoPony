> **Language:** [Bahasa Indonesia](DOKUMENTASI.md) | [English](DOCUMENTATION_EN.md)

# NanoPony Framework Documentation (v0.0.59 - May 2026 Sync)

**NanoPony** is a Go framework for Kafka-Oracle integration featuring a clean, reusable, and production-ready architecture. This framework provides ready-to-use components to build scalable data pipelines with minimal boilerplate and high memory efficiency using `sync.Pool`.

---

## Table of Contents

1. [Overall Architecture](#overall-architecture)
2. [Configuration Layer](#1-configuration-layer)
3. [Database Layer (Oracle)](#2-database-layer-oracle)
4. [Kafka Layer](#3-kafka-layer)
5. [Worker Pool](#4-worker-pool)
6. [Poller](#5-poller)
7. [Logging System (Optimized)](#6-logging-system-optimized)
8. [Framework Builder & Lifecycle](#8-framework-builder--lifecycle)
9. [Best Practices](#best-practices)

---

## Overall Architecture

NanoPony uses the **Builder** pattern to assemble application components declaratively. The final result of the build process is a `FrameworkComponents` object that manages the lifecycle of the entire service.

```
┌───────────────────────────────────────────────┐
│           Framework Builder (Fluent API)      │
│  (WithConfig -> WithDatabase -> WithPoller)   │
└───────────────────────────────────────────────┘
                     ↓
┌───────────────────────────────────────────────┐
│            FrameworkComponents                │
│ (DB, Producer, WorkerPool, Poller, Cleanup)   │
└───────────────────────────────────────────────┘
          /          |          \
┌──────────┐   ┌────────────┐   ┌────────────┐
│  Poller  │──>│ Worker Pool│──>│ Job Handler│
└──────────┘   └────────────┘   └────────────┘
     ^               |                |
     |        (Submit Job)       (Business Logic)
  (Fetch)            |                |
     |               v                v
┌──────────┐   ┌────────────┐   ┌────────────┐
│  Oracle  │   │ Kafka/ES   │   │ Business   │
└──────────┘   └────────────┘   └────────────┘
```

---

## 1. Configuration Layer

**File**: `config.go`, `config_init.go`

### Description
Centralized configuration management based on environment variables with singleton synchronization support and automatic validation.

### Key Environment Variables

| Variable | Description | Allowed Values |
|----------|-----------|-------------------------|
| `GO_ENV` | Environment mode | `local`, `staging`, `production` |
| `KAFKA_MODELS` | Kafka configuration | `kafka-localhost`, `kafka-staging`, `kafka-production`, `kafka-confluent` |
| `OPERATION` | Operation mode | Custom per application |
| `LOG_FILE_PREFIX` | Log file name prefix | Any (Default: `orion-to-core`) |

### Advanced Features

#### A. Dynamic Config
Allows loading custom environment variables without modifying the NanoPony `Config` struct.
```go
config := nanopony.NewConfig()
config.LoadDynamic("CUSTOM_") // Loads all env vars prefixed with CUSTOM_
apiUrl := config.Dynamic["CUSTOM_API_URL"]
```

#### B. ElasticSearch Config
Out-of-the-box Elasticsearch support.
- `ELASTIC_HOST`, `ELASTIC_USERNAME`, `ELASTIC_PASSWORD`
- `ELASTIC_INDEX_DATA`, `ELASTIC_API_KEY`, `ELASTIC_PREFIX_INDEX`

#### C. Validation & Reset
- **`Validate()`**: Ensures mandatory variables (`GO_ENV`, `KAFKA_MODELS`) are set before running the framework.
- **`ResetConfig()`**: Used in unit testing to reset global configuration state.

---

## 2. Database Layer (Oracle)

**File**: `database.go`

### Connection Pooling
NanoPony sets optimal connection limits by default:
- `MaxIdleConns`: 2
- `MaxOpenConns`: 20
- `ConnIdleTime`: 5 Minutes
- `ConnMaxLifetime`: 60 Minutes

### Initialization
```go
// Safe way (Recommended)
fw, err := nanopony.NewFramework().
    WithConfig(config).
    WithDatabaseSafe() // Returns error if connection fails

// Using an existing instance
fw.WithDatabaseFromInstance(myExistingDB)
```

### Debugging Utilities (SQL Interpolation)
NanoPony provides tools to simplify query logging that contains parameters.
> [!WARNING]
> Use only for logging/debugging. Do not execute interpolated results directly to the DB to avoid SQL Injection.

```go
nanopony.LogInterpolatedQuery(
    "SELECT * FROM users WHERE id = :id",
    sql.Named("id", 123),
)
// Output: [SQL Query] SELECT * FROM users WHERE id = 123
```

---

## 3. Kafka Layer

**File**: `kafka.go`

### Automatic Optimization
When you call `.Build()` on the framework, NanoPony automatically adjusts the Kafka Writer's `BatchSize` to match the number of workers in the Worker Pool. This ensures a balanced throughput between data processing and message delivery.

### Confluent Cloud Mode
If `KAFKA_MODELS=kafka-confluent`, NanoPony automatically enables SASL/PLAIN authentication and TLS using:
- `API_KEY_KAFKA_CONFLUENT`
- `API_SECRET_KAFKA_CONFLUENT`
- `BOOTSTRAP_SERVER_KAFKA_CONFLUENT`

---

## 4. Worker Pool

**File**: `job.go`, `worker.go`

### Key Concept
The Worker Pool manages a collection of goroutines that process `Job`s concurrently. Since v0.0.59, NanoPony uses `sync.Pool` to recycle `Job` objects, drastically reducing memory allocation.

### Job Metadata & sync.Pool
You must use `nanopony.AcquireJob()` to create a new job. This object will be automatically returned to the pool by the worker after the handler finishes.
```go
job := nanopony.AcquireJob()
job.ID = "job-123"
job.Data = payload
job.Meta["source"] = "poller-01"
// No manual release needed if submitted to WorkerPool
pool.SubmitBlocking(ctx, job)
```

### Backpressure Mechanism (SubmitBlocking)
Highly recommended for production. If the queue is full, the caller will "wait" until space becomes available, preventing data loss.
```go
err := pool.SubmitBlocking(ctx, job)
```

---

## 5. Poller

**File**: `poller.go`

### Slot Mechanism (Semaphore)
The Poller uses `JobSlotSize` to limit how many polling operations run concurrently. If slots are full (e.g., because a previous poll is still in progress), the next poll is skipped.

### Job ID Uniqueness (May 2026)
Every job generated by the Poller has an ID format: `poll-[SessionID]-[Counter]`.
- **SessionID**: Based on the application startup timestamp.
- **Counter**: An atomic counter that increments.
To avoid excessive heap allocation, the ID is now constructed using `strings.Builder` and `strconv.FormatInt`, which are far more efficient than `fmt.Sprintf`.

---


## 7. HTTP Server (Fiber-Compatible)

**File**: 

### Description
NanoPony now includes an internal HTTP server module designed to provide a development experience identical to the Fiber framework. This module allows you to build control APIs, monitoring dashboards, or webhook integrations directly within the NanoPony framework lifecycle without needing external web framework dependencies.

### Key Features
- **Expressive Routing**: , , , , etc.
- **Path Parameters**: Supports , , etc.
- **Wildcard Support**: Supports  for catch-all routes or static files.
- **Middleware Chain**: Supports  for sequential and organized middleware workflows.
- **Group Routing**: Organize routes with shared prefixes for code modularity.
- **Context API**: Clean request/response abstraction (identical to Fiber).

### Complete Usage Example

json:"data"


### Built-in Middleware
NanoPony provides essential middleware ready to use (identical to Fiber):

- ****: Logs each request (Method, Path, Status, Latency).
- ****: Recovers the server from panics to prevent crashes.
- ****: Configures Cross-Origin Resource Sharing.
- ****: Adds a unique ID to each request header.
- ****: Compresses responses using Gzip.
- ****: Adds various security headers for protection.
- ****: Protects routes with HTTP Basic Authentication.
- ****: Limits request rate to prevent spam/abuse.
- ****: Provides a server metrics dashboard similar to Fiber's monitor.
- **`nanopony.BasicAuth()`**: Protects routes with HTTP Basic Authentication.
- **`nanopony.RateLimiter()`**: Limits request rate to prevent spam/abuse.
- **`nanopony.Monitor()`**: Provides a server metrics dashboard similar to Fiber.
- **`nanopony.Favicon()`**: Serves a favicon.ico icon.
- ****: Serves a favicon.ico icon.

#### Middleware Usage Example:

### Context (Ctx) Methods Explained
 is the heart of every HTTP request in NanoPony. Here are its main methods:

- ****: Sends a JSON response and automatically sets the  header.
- ****: Sends a plain text response.
- ****: Sets the HTTP status code (e.g., 200, 404, 500).
- ****: Parses the JSON request body directly into a struct or map.
- ****: Retrieves values from route parameters (e.g., ).
- ****: Retrieves values from the query string (e.g., ).
- ****: Crucial for middleware; calls the next handler in the chain.
- ****: Stores data that can be accessed by subsequent middleware or handlers within a single request cycle.

---

## 6. Logging System (Optimized)

**File**: `logger.go`, `logger_internal.go`

### Description
Structured logging with file rotation, console output, and Elasticsearch integration.

### Deep Copy Optimization (May 2026)
Previously, `processPayload` used JSON `Marshal`/`Unmarshal` cycles to perform a *deep copy* of the payload. This caused high CPU and memory load.
Now, NanoPony uses **Recursive Manual Copy** for `map[string]any` types, which is significantly faster and more allocation-efficient.

```go
// New deep copy illustration
func deepCopyMap(m map[string]any) map[string]any {
    // ... manual recursive copy is much more efficient ...
}
```

---

## 8. Framework Builder & Lifecycle

**File**: `framework.go`

### Build Stages (Fluent API)
```go
components := nanopony.NewFramework().
    WithConfig(config).
    WithDatabase().
    WithKafkaWriter().
    WithProducer().
    WithWorkerPool(5, 100, 3).
    WithPoller(pollerConfig, fetcher).
    AddCleanup(myCleanupFunc).
    Build()
```

### Execution Stages
1. **`Start(ctx, handler)`**:
   - Starts Worker Pool.
   - Starts Poller.
2. **`Shutdown(ctx)`**:
   - Stops Poller first.
   - Stops Worker Pool (waits for tasks to complete).
   - Closes DB and Kafka connections.
   - Runs custom cleanup functions concurrently.

---

## Best Practices

### 1. Use SubmitBlocking
Avoid regular `Submit` if you do not want to ignore data during high system load. `SubmitBlocking` provides a natural backpressure mechanism.

### 2. Implement Graceful Shutdown
Always ensure `components.Shutdown(ctx)` is called, ideally using OS signals (`SIGINT`, `SIGTERM`).
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
<-sigChan
components.Shutdown(ctx)
```

### 3. Monitor Error Channel
Do not let errors in the Worker Pool go unnoticed. Always listen to the `Errors()` channel.
```go
go func() {
    for err := range components.WorkerPool.Errors() {
        log.Printf("Worker Alert: %v", err)
    }
}()
```

### 4. Utilize Dynamic Config
Use `DynamicConfig` for optional settings or custom modules to keep the framework clean of dependencies not needed by all teams.

---

## Summary of Key Features

✅ **Auto-Scale Kafka Batching** - Based on worker pool capacity.  
✅ **Oracle Pool Optimization** - Safe default settings for production.  
✅ **Dynamic Configuration** - Flexible for custom environment variables.  
✅ **Safe-Fail Validation** - Validates configuration upon startup.  
✅ **Aggregated Shutdown Errors** - Reports all issues at the end of the shutdown process.  
✅ **SQL Debugger** - Interpolation utility for readable SQL logging.  
✅ **High Performance** - ~39x faster than Fiber for internal jobs, with optimized allocations on hot paths.
