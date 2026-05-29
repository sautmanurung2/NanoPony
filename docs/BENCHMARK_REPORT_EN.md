> **Language:** [Bahasa Indonesia](BENCHMARK_REPORT.md) | [English](BENCHMARK_REPORT_EN.md)

# NanoPony: Benchmark & Performance Report

*This document presents the performance analysis, memory efficiency, and comparison of the latest NanoPony framework (May 2026).*

---

## 🚀 Performance Summary (Update May 2026)

Following a series of optimizations on system *hot paths*, NanoPony now achieves even higher efficiency:

| Metric | Value | Note |
| :--- | :--- | :--- |
| **Throughput** | **~390.2 ns/op** | Improved efficiency via sharding optimization |
| **Memory Allocation** | **3 allocs/op** | Highly stable for high-volume processing |
| **Memory Leak Status** | **✅ PASSED** | No leaks detected after 40+ cycles |

---

## 📊 Benchmark Results (vs Other Frameworks)

NanoPony is designed to process *jobs* (background tasks) efficiently. Here is a comparison with other popular Go frameworks:

| Framework | Throughput (Job Processing) | Status |
| :--- | :--- | :--- |
| **NanoPony** | **~390 ns/op** | 🚀 **Overall Champion** |
| **Echo** | ~2.502 ns/op | 🥈 Fast |
| **Iris** | ~2.827 ns/op | 🥉 Fast |
| **Fiber** | ~10.769 ns/op | 🐢 Slow (in this scenario) |

> **Analysis:** NanoPony excels by avoiding heavy HTTP stack overhead. Our *sharded worker pool* architecture drastically reduces *lock contention*, which typically hampers scalability.

---

## ⚙️ Optimization Details (Updated May 2026)

To achieve current throughput, we implemented the following optimizations:

### 1. Sharded Worker Pool
We split the single `WorkerPool` into multiple independent *shards*.
- **Benefit**: Drastically reduces *lock contention* on the pool mutex, especially under high load with many goroutines.

### 2. Hot Path Efficiency (Poller ID)
Replaced `fmt.Sprintf` with `strings.Builder` and `strconv.FormatInt`.
- **Benefit**: Eliminates unnecessary *heap* allocations during job ID creation.

### 3. Logging System (Manual Deep-Copy)
Replaced JSON `Marshal`/`Unmarshal` cycles with manual recursion for deep-copying `map[string]any` payloads.
- **Benefit**: Drastically reduces CPU usage and memory allocation in the structured logging system.

### 4. Job Lifecycle Management
Changed the strategy for clearing map metadata in `Job.Release` by recreating the map (`make`).
- **Benefit**: More efficient for the *Garbage Collector* compared to removing keys one-by-one (*key-by-key deletion*).

---

## 🔍 Memory Leak Stability Test Results

| Component | Explanation | Status |
| :--- | :--- | :--- |
| **Core Lifecycle** | 40 full setup-shutdown cycles | ✅ Stable (+19 KB) |
| **WorkerPool** | Stress test 1,000+ jobs | ✅ Stable (+6 KB) |
| **Poller** | 10 polling cycles | ✅ Stable (+2 KB) |
| **Concurrent** | 20 simultaneous instances | ✅ Stable (+50 KB) |

---

## 🎯 Final Conclusion

The NanoPony framework is a *high-performance* solution for Kafka-Oracle integration. With its modular architecture and advanced optimizations, NanoPony delivers excellent efficiency for background processing systems requiring high throughput with a minimal memory footprint.

*Report generated on May 29, 2026. Data valid for v0.0.59.*
