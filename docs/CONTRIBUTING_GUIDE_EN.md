> **Language:** [Bahasa Indonesia](CONTRIBUTING_GUIDE.md) | [English](CONTRIBUTING_GUIDE_EN.md)

# 🚀 NanoPony Contribution & Development Guide

> **IMPORTANT:** Before contributing, ensure you have read and agreed to **[CONTRIBUTING.md](./CONTRIBUTING.md)** and **[CLA.md](./CLA.md)**. All contributions will be jointly owned by Saut Manurung & JNE Indonesia.

Welcome to NanoPony! This framework is designed to be a "high-performance bridge" between Oracle Database and Kafka. This document will help you understand the code structure, design patterns, and technical standards we use.

---

## 📑 Table of Contents
1. [Design Philosophy](#-design-philosophy)
2. [Technical Architecture (Deep Dive)](#-technical-architecture-deep-dive)
3. [Coding Standards & Conventions](#-coding-standards--conventions)
4. [Development Steps](#-development-steps)
5. [Testing & Benchmarking](#-testing--benchmarking)
6. [Pull Request Workflow](#-pull-request-workflow)

---

## 🧠 Design Philosophy

NanoPony is built on three main pillars:
1. **Fluent Builder Pattern**: Minimizes boilerplate. Application setup should look like reading a sentence.
2. **Explicit Lifecycle**: Start and Shutdown must be coordinated. There should be no "orphaned goroutines" when the application exits.
3. **Backpressure Aware**: The system must know when to stop pulling data if downstream processing is overwhelmed.

---

## 🏗 Technical Architecture (Deep Dive)

### 1. Framework Builder (`framework.go`)
This is the main entry point. The framework uses *method chaining*.
* **Why Builder?** Because components like Poller need a `WorkerPool`, and Producer needs a `KafkaWriter`. The builder ensures all dependencies are correctly wired before the application starts.
* **Rule:** If you add a new component, add a `.WithComponent()` method that returns `*Framework`.

### 2. Worker Pool & Job System (`worker.go`, `job.go`)
NanoPony does not process data directly in a single loop. Data is wrapped into a `Job`.
* **Job Struct**: Holds an ID (traceability), Data (payload), and Meta (extra context).
* **Worker**: Goroutines that continuously pull `Job`s from a channel.
* **SubmitBlocking**: This is a crucial feature. If the queue (buffer channel) is full, this function will "block" the sender until a slot is available. This prevents RAM exhaustion due to data pile-up.

### 3. Poller & Semaphore (`poller.go`)
The Poller is responsible for fetching data periodically.
* **Slot/Semaphore**: We use a channel as a semaphore (`jobSlots`). If `JobSlotSize` is 1, the next poll will not run until the previous one completes. This prevents duplicate processing of the same data.
* **DataFetcher**: An interface. You can create a fetcher from a Database, API, or File, as long as it returns `[]any`.

### 4. Structured Logging (`logger.go`)
Our logger is designed for *production monitoring*.
* **Async Logging**: Logs are not written to files synchronously but via a channel to avoid slowing down the main process.
* **Hybrid Output**: Can log to Console (for dev) and Elasticsearch (for prod) simultaneously.

---

## 📏 Coding Standards & Conventions

### 1. Naming
* **Interfaces**: Use `-er` suffix if possible (e.g., `DataFetcher`, `MessageProducer`).
* **Private Variables**: Use `camelCase`.
* **Exported Functions**: Must have clear GoDoc comments.

### 2. Error Handling
* **Do not ignore**: Always check `if err != nil`.
* **Wrap Errors**: Use `fmt.Errorf("context: %w", err)` to provide context on where the error occurred.
* **Worker Errors**: Errors inside workers are sent to the `pool.Errors()` channel, not logged randomly inside the worker.

### 3. Concurrency
* **Mutex**: Always use `sync.RWMutex` for state that is frequently read but rarely written.
* **Context**: Use `ctx.Done()` in every long loop so goroutines can stop when the application is shut down.

---

## 🛠 Development Steps

1. **Prepare Environment:**
   * Copy `.env.example` to `.env`.
   * Ensure Go version 1.25+ is installed.
2. **Creating New Features:**
   * If adding features at the framework level, start in `framework.go`.
   * If adding a new database adaptor, create a new file (e.g., `postgres.go`).
3. **Surgical Updates:** Do not change code unrelated to your feature. Keep Pull Requests (PR) focused.

---

## 🧪 Testing & Benchmarking

### Running Unit Tests
```bash
# Run all tests with detail
go test -v ./...

# Check for race conditions (very important for worker pool)
go test -race ./...
```

### Running Benchmarks
NanoPony is performance-focused.
```bash
go test -bench=. -benchmem
```
* **Target**: Memory allocation should remain stable (no continuous growth/leak) during 40+ processing cycles.

---

## 🤝 Pull Request Workflow

1. **Fork & Branch**: Create a branch with a descriptive name (e.g., `feat/add-postgres-support`).
2. **Commit**: Use clear commit messages (e.g., `feat: implement postgres adapter for framework builder`).
3. **Documentation**: Update `README.md` or `DOKUMENTASI.md` if there are changes to framework usage.
4. **Review**: Tag maintainers for review. Ensure all checks (lint & test) are green ✅.

---

> **Remember:** At NanoPony, good code is code that reads like a story and runs like the wind. Happy coding! 🐎⚡
