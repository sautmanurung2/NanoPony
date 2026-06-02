> **Language:** [Bahasa Indonesia](BENCHMARK_REPORT.md) | [English](BENCHMARK_REPORT_EN.md)

# NanoPony: Benchmark & Performance Report

*This document presents the performance analysis, memory efficiency, and comparison of the latest NanoPony framework (June 2026).*

---

## 🚀 Performance Summary (Update June 2026)

Following a series of optimizations on system *hot paths*, NanoPony achieves extremely high efficiency for internal job processing:

| Metric | Value | Note |
| :--- | :--- | :--- |
| **Throughput (Worker Pool)** | **~438.5 ns/op** | High performance thanks to sharded worker pool architecture |
| **Memory Allocation** | **2 allocs/op** | Highly efficient heap usage (16 B/op) |
| **Memory Leak Status** | **✅ PASSED** | Negative memory growth (-58 KB) after 50 cycles |

---

## 📊 Benchmark Results (vs Other Frameworks)

NanoPony is designed to process *jobs* (background tasks) efficiently without the overhead of an HTTP stack. Here is a comparison with popular web frameworks:

| Framework | Throughput (Job Processing) | Status |
| :--- | :--- | :--- |
| **NanoPony** | **~1.48 µs/op** | 🚀 **Overall Champion** |
| **Iris** | ~3.10 µs/op | 🥈 Fast |
| **Echo** | ~3.17 µs/op | 🥉 Fast |
| **Fiber** | ~19.87 µs/op | 🐢 Slow (in this scenario) |

> **Analysis:** NanoPony excels by focusing on *internal job processing*. Our *sharded worker pool* architecture drastically reduces *lock contention* compared to frameworks designed for HTTP request-response cycles.

---

## ⚙️ Optimization Details (Updated June 2026)

To achieve current throughput, we implemented the following optimizations:

### 1. Sharded Worker Pool
We split the single `WorkerPool` into multiple independent *shards*.
- **Benefit**: Drastically reduces *lock contention* on the pool mutex, especially under high load with many goroutines.

### 2. Hot Path Efficiency (Poller ID)
Replaced `fmt.Sprintf` with `strings.Builder` and `strconv.AppendInt`.
- **Benefit**: Eliminates unnecessary *heap* allocations during job ID creation in the Poller.

### 3. Job Lifecycle Management (sync.Pool)
Utilized `sync.Pool` to recycle `Job` objects.
- **Benefit**: Significantly reduces *Garbage Collector* load by reusing existing memory allocations.

### 4. Zero-Allocation Field Access
Field access within `FrameworkComponents` is optimized for zero allocations (0 allocs/op).

---

## 🔍 Memory Leak Stability Test Results

| Component | Explanation | Status |
| :--- | :--- | :--- |
| **Core Lifecycle** | 50 full setup-shutdown cycles | ✅ Stable (-58 KB growth) |
| **WorkerPool** | Stress test 1,000 jobs | ✅ Stable (+66 KB growth) |
| **Concurrent** | 20 simultaneous instances | ✅ Stable (+42 KB growth) |
| **Poller** | Long running (2 seconds) | ✅ Stable (100% processed) |

---

## 🎯 Final Conclusion

The NanoPony framework is a *high-performance* solution for Kafka-Oracle integration. With its modular architecture and advanced job lifecycle management optimizations, NanoPony delivers excellent efficiency for background processing systems requiring high throughput with a minimal memory footprint.

*Report generated on June 3, 2026. Data valid for v0.0.59.*
