> **Language:** [Bahasa Indonesia](BENCHMARK_REPORT.md) | [English](BENCHMARK_REPORT_EN.md)

# NanoPony: Benchmark & Performance Report

*This document presents the performance analysis, memory efficiency, and comparison of the latest NanoPony framework (June 2026).*

---

## 🚀 Performance Summary (Update September 2026)

Following a series of optimizations on system *hot paths* and the implementation of a native pure Go standard library Kafka subsystem, NanoPony achieves extremely high efficiency for internal job processing and event streaming:

| Metric | Value | Note |
| :--- | :--- | :--- |
| **Throughput (Worker Pool)** | **~438.5 ns/op** | High performance thanks to sharded worker pool architecture |
| **Memory Allocation** | **2 allocs/op** | Highly efficient heap usage (16 B/op) |
| **Memory Leak Status** | **✅ PASSED** | Negative memory growth (-58 KB) after 50 cycles |
| **Kafka Wire Protocol** | **Zero-Third-Party** | 100% Go stdlib (`net`, `crypto/tls`, `encoding/binary`, `hash/crc32`) |
| **Test Coverage & Safety** | **85.7% (0 Race)** | Verified via `-race`, in-memory TCP mock server without external Kafka cluster |

---

## 📊 Multi-Framework Benchmark Results (Latest Update)

The benchmark was executed head-to-head on a Linux environment (13th Gen Intel Core i7-13700H) using `go test -tags benchmark -bench=. -benchmem` in the `benchmark/` directory.

### 1. Job Processing Throughput
NanoPony is purpose-built to process event and background workloads without HTTP stack overhead:

| Framework | Speed (Throughput) | Memory (B/op) | Allocation (Allocs/op) | Rank & Status |
| :--- | :--- | :--- | :--- | :--- |
| **NanoPony** | **~3.94 µs/op** (3,943 ns/op) | **98 B/op** | **2 allocs/op** | 🚀 **Rank 1 (Fastest & Most Efficient)** |
| **Echo** | ~6.00 µs/op (5,999 ns/op) | 5,312 B/op | 13 allocs/op | 🥈 Fast |
| **Iris** | ~6.86 µs/op (6,860 ns/op) | 5,312 B/op | 13 allocs/op | 🥉 Fast |
| **Fiber** | ~26.11 µs/op (26,112 ns/op) | 5,533 B/op | 21 allocs/op | 🐢 Slow (~6.6x slower) |

### 2. Setup Overhead & Idle Memory

| Framework | Setup Overhead | Setup Memory | Idle Memory Footprint |
| :--- | :--- | :--- | :--- |
| **NanoPony** | 689.4 µs/op | 4,037 B/op (17 allocs) | **14 KB** (Active Worker Pool) |
| **Fiber** | 4.7 µs/op | 2,912 B/op (12 allocs) | **2 KB** |
| **Echo** | 1,538.2 µs/op | 3,632 B/op (53 allocs) | **0 KB** |
| **Iris** | 216.3 µs/op | 21,560 B/op (221 allocs) | **14 KB** |

> **Analysis:**
> - **Superior Throughput**: NanoPony processes jobs **6.6x faster than Fiber**, **1.5x faster than Echo**, and **1.7x faster than Iris**.
> - **Extreme Memory Efficiency (54x Lower Allocation)**: Leveraging `sync.Pool` job recycling and zero third-party library overhead, NanoPony requires only **98 B/op** and **2 allocations** per job. By contrast, conventional web frameworks expend **5,300 - 5,500 B/op** and 13 - 21 allocations per cycle.
> - **Deliberate Setup**: NanoPony's initialization upfront boots the worker pool goroutines, bounded channels, and verifies native protocols so that execution hot paths experience zero lock contention and no buffer reallocations.

---

## ⚙️ Optimization Details (Updated June - September 2026)

To achieve current throughput and eliminate external library overhead, we implemented the following optimizations:

### 1. Sharded Worker Pool
We split the single `WorkerPool` into multiple independent *shards*.
- **Benefit**: Drastically reduces *lock contention* on the pool mutex, especially under high load with many goroutines.

### 2. Hot Path Efficiency (Poller ID)
Replaced `fmt.Sprintf` with `strings.Builder` and `strconv.AppendInt`.
- **Benefit**: Eliminates unnecessary *heap* allocations during job ID creation in the Poller.

### 3. Job Lifecycle Management (sync.Pool & Atomic CAS)
Utilized `sync.Pool` to recycle `Job` objects guarded by an `inUse atomic.Bool` fence via Compare-And-Swap (CAS).
- **Benefit**: Significantly reduces *Garbage Collector* load while preventing double-release / data race issues under intense concurrent execution.

### 4. Zero-Allocation Field Access
Field access within `FrameworkComponents` is optimized for zero allocations (0 allocs/op).

### 5. Native Pure Go Kafka Wire Protocol (September 2026)
Replaced third-party libraries (`github.com/segmentio/kafka-go`, `klauspost/compress`, `pierrec/lz4`) with a wire protocol built strictly on the Go standard library (`net`, `crypto/tls`, `encoding/binary`, `hash/crc32`).
- **Benefit**: Eliminates heavy external dependencies, minimizes binary footprint, avoids excessive reflection allocations, and supports distributed partitioning strategies (RoundRobin, LeastBytes, Hash/Murmur2) over raw TCP/TLS sockets.

---

## 🔍 Memory Leak Stability & Concurrency Safety Test Results

| Component | Explanation | Status |
| :--- | :--- | :--- |
| **Core Lifecycle** | 50 full setup-shutdown cycles | ✅ Stable (-58 KB growth) |
| **WorkerPool** | Stress test 1,000 jobs | ✅ Stable (+66 KB growth) |
| **Concurrent** | 20 simultaneous instances | ✅ Stable (+42 KB growth) |
| **Poller** | Long running (2 seconds) | ✅ Stable (100% processed) |
| **Native Kafka Protocol** | Mock in-memory TCP socket test (Produce/Fetch/ListOffsets/Metadata/SASL) | ✅ Stable (0 leak, 0 race, 85.7% coverage) |

---

## 🎯 Final Conclusion

The NanoPony framework is a *high-performance* solution for Kafka-Oracle integration. With its pure native Kafka implementation (zero third-party dependencies) and modular architecture featuring advanced job lifecycle optimizations, NanoPony delivers outstanding efficiency for background processing systems requiring high throughput with minimal memory footprint and clean dependencies.

*Report updated on September 10, 2026. Data valid for v0.0.60 (Pure Native Kafka).*
