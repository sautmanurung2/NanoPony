> **Language:** [Bahasa Indonesia](ARCHITECTURE.md) | [English](ARCHITECTURE_EN.md)

# NanoPony Framework - Architecture Documentation

## 📋 Table of Contents

1. [Summary](#summary)
2. [Architecture Diagram](#architecture-diagram)
3. [Core Components](#core-components)
   - [Configuration Layer](#1-configuration-layer)
   - [Database Layer](#2-database-layer)
   - [Kafka Layer](#3-kafka-layer)
   - [Producer & Consumer](#4-producer--consumer)
   - [Worker Pool](#5-worker-pool)
   - [Poller](#6-poller)
   - [Framework Builder](#9-framework-builder)
   - [Logger](#10-logger)
4. [Data Flow Diagram](#data-flow-diagram)
5. [Design Patterns](#design-patterns)
6. [Configuration Reference](#configuration-reference)
7. [Interface Reference](#interface-reference)
8. [Quick Start Guide](#quick-start-guide)

---

## Summary

**NanoPony** is a Go framework (`github.com/sautmanurung2/nanopony`) providing a comprehensive Kafka-Oracle integration platform with worker pool and polling capabilities. It is designed using a fluent builder pattern to wire database connections, Kafka producers/consumers, concurrent worker pools, and data pollers.

**Key Features:**
- ✅ **Ultra-Efficient sync.Pool**: Reuses Job objects for high performance and low GC pressure.
- ✅ **Kafka Producer & Consumer**: Integration with Kafka using `kafka-go`.
- ✅ **Oracle Database**: Connection management using `go-ora` with connection pooling.
- ✅ **Worker Pool**: Concurrent job processing with bounded queues.
- ✅ **Poller**: Periodic data fetching with configurable intervals.
- ✅ **HTTP Server (Fiber-Compatible)** - Internal HTTP Server with routing, middleware, groups, and context API identical to the Fiber framework.
- ✅ **Builder Pattern**: Clean and fluent API.
- ✅ **Graceful Shutdown**: Safe teardown for all components.
- ✅ **Environment-based Configuration**: Settings via environment variables.
- ✅ **Structured Logging**: File rotation and Elasticsearch integration.

**Core Dependencies:**
- `segmentio/kafka-go` - Kafka client
- `sijms/go-ora/v2` - Oracle driver
- `elastic/go-elasticsearch/v8` - Elasticsearch client
- `joho/godotenv` - Environment variables loader
- `natefinch/lumberjack` - Log rotation

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Framework Builder                         │
│  (framework.go - Fluent API, orchestrates all components)    │
└───────────────┬───────────────────────────────┬─────────────┘
                │                               │
                ▼                               ▼
┌───────────────────────┐         ┌─────────────────────────────┐
│   Config Layer        │         │   FrameworkComponents       │
│  config.go            │         │   (holds all artifacts)     │
│  config_init.go       │         │   Start() / Shutdown()      │
│  (env-based, singleton)│        │   Getters for DB, Producer  │
└───────────┬───────────┘         └─────────────────────────────┘
            │
    ┌───────┼───────────┬──────────────────────┐
    ▼       ▼           ▼                      ▼
┌────────┐ ┌──────────┐ ┌──────────────┐ ┌──────────────┐
│ Oracle │ │ Standard │ │ Confluent    │ │ Elastic      │
│  DB    │ │ Kafka    │ │ Cloud Kafka  │ │ Search       │
│database│ │ kafka.go │ │ (SASL/TLS)   │ │ logger.go    │
│  .go   │ │          │ │              │ │ (Public)     │
└───┬────┘ └────┬─────┘ └──────┬───────┘ └──────┬───────┘
    │           │              │                │
    ▼           ▼              ▼                ▼
┌──────────────────────────────────────┐  ┌──────────────┐
│           Data Access Layer          │  │ logger_inter │
│  producer.go  - KafkaProducer        │  │ nal.go       │
│               - KafkaConsumer        │  │ (Machinery)  │
└───────────────────┬──────────────────┘  └──────────────┘
                    │
                    ▼
┌──────────────────────────────────────────────────────────┐
│                 Processing Layer                         │
│  job.go    - Unit of Work (Job Struct)                   │
│  poller.go - Data Fetching & Rate Limiting               │
│  worker.go - Worker Pool Execution Logic                 │
└──────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Configuration Layer

**File:** `config.go`, `config_init.go`

**Purpose:** Centralized, environment-driven singleton configuration.

**Main Structs:**
- `Config` - Top-level container for `AppConfig`, `OracleConfig`, `KafkaConfig`, `KafkaConfluentConfig`, `ElasticSearchConfig`.
- `envConfig` - Validation helper with `validValues` and `defaultVal`.
- `oracleEnv` - Maps environment variable names (staging vs production).

**Key Functions:**
- `NewConfig()` - Initializes singleton; loads `.env`, runs all `init*` functions.
- `BuildConfig(initFuncs ...func(*Config))` - Allows custom initialization callbacks.
- `ResetConfig()` - Resets singleton (for testing).

**Design Pattern:** Singleton + Strategy (env-based routing). Configuration is validated upon loading.

---

### 2. Database Layer

**File:** `database.go`

**Purpose:** Oracle database connection management with connection pooling.

**Key Features:**
- `MaxIdleConns`, `MaxOpenConns`, `ConnIdleTime`, `ConnMaxLifetime` configurations.
- `NewOracleConnection(config)` - Builds Oracle URL, opens `*sql.DB`, configures pool.
- `InterpolateQuery(query, args...)` - Formats SQL queries for debugging.

---

### 3. Kafka Layer

**File:** `kafka.go`

**Purpose:** Kafka writer creation supporting standard Kafka and Confluent Cloud (SASL/TLS).

**Key Features:**
- `NewKafkaWriter(config)` - Creates `*kafka.Writer` with load balancing and batch timeouts.
- SASL/TLS support via `createSASLTransport`.

---

### 4. Producer & Consumer

**File:** `producer.go`

**Purpose:** Message production and consumption interface with JSON serialization.

**Interfaces/Structs:**
- `MessageProducer` interface.
- `KafkaProducer` - Wraps `*kafka.Writer`.
- `KafkaConsumer` - Wraps `*kafka.Reader`.

---

### 5. Worker Pool

**File:** `job.go`, `worker.go`

**Purpose:** Concurrent job processing with fixed goroutine pool and bounded queues.

**Design Pattern:** Worker Pool (bounded goroutine pool with channel communication). Thread-safe via `sync.RWMutex`.

---

### 6. Poller

**File:** `poller.go`

**Purpose:** Periodic data fetching with rate-limited job submission.

**Mechanism:** Timer-based polling + Semaphore (channel `jobSlots`) to limit concurrent polls.

---

### 7. Framework Builder

**File:** `framework.go`

**Purpose:** Fluent builder managing component dependencies and lifecycles.

**Method Builder:** Chainable `With*()` methods. `Build()` returns `FrameworkComponents`.

**FrameworkComponents.Start/Shutdown:** Coordinated lifecycle management.

---

### 8. Logger

**File:** `logger.go`, `logger_internal.go`

**Purpose:** Structured logging with file rotation, console output, and Elasticsearch integration.

**Design Pattern:** Lazy init singleton, Strategy.

---

## Data Flow Diagram

### Poll-to-Process Flow

```
   Ticker (every Interval)
        │
        ▼
   pollOnce()
        │
        ├── Acquire job slot (semaphore)
        │
        ▼
   DataFetcher.Fetch()  ──►  []any  (raw items)
        │
        ▼
   For each item:
        │
        ├── Create Job{Data: item}
        │
        ▼
   WorkerPool.Submit(job)  ──►  jobChan (buffered)
                                      │
                                      ▼
                              Worker Goroutine
                                      │
                                      ▼
                              JobHandler(ctx, job)
                                      │
                                      │
                                      ▼
                          Kafka Produce / Database Query
```

---

## Best Practices

1. **Always use Graceful Shutdown** - Ensures all resources are properly released.
2. **Use Builder Pattern** - Improves setup readability.
3. **Use Layered Architecture** - Increases separation of concerns.
4. **Use Context** - Enables correct lifecycle management.
5. **Proper Error Handling** - Monitor the `WorkerPool` error channel.
6. **Connection Pool Tuning** - Tune based on workload.
7. **Dependency Injection** - Use `With*FromInstance()` for testing.

---

## Quick Start Guide

### Build Framework
```go
framework := nanopony.NewFramework().
    WithConfig(config).
    WithDatabase().
    WithKafkaWriter().
    WithProducer().
    WithWorkerPool(5, 100, 3).
    WithPoller(nanopony.DefaultPollerConfig(), dataFetcher)

components := framework.Build()
```

### Start Framework
```go
components.Start(ctx, jobHandler)
```

### Graceful Shutdown
```go
components.Shutdown(ctx)
```
