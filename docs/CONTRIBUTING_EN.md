> **Language:** [Bahasa Indonesia](CONTRIBUTING.md) | [English](CONTRIBUTING_EN.md)

# Contributing to NanoPony

First of all, thank you for considering contributing to NanoPony! People like you make NanoPony an incredible tool for high-efficiency Kafka-Oracle integration.

## Legal Notice & Contributor License Agreement (CLA)

By contributing to this project, you agree to the terms in the [Contributor License Agreement (CLA)](CLA.md).

**Important:** Your contributions will be jointly owned by **Saut Manurung** and **JNE Indonesia**. By submitting a Pull Request (PR), you acknowledge and agree to this transfer of ownership as detailed in the CLA. Every Pull Request must include a checkmark in the PR template indicating your agreement.

## Development Standards

### Environment Setup
- **Go Version**: 1.21+ (1.22+ recommended)
- **Dependencies**: Managed via Go modules. Run `go mod download` to prepare.

### Coding Standards
- **Formatting**: Always run `go fmt ./...` before committing.
- **Documentation**: All public (exported) functions, types, and constants **MUST** have GoDoc comments explaining their purpose, parameters, and return values.
- **Naming**: Use `camelCase` for local variables and `PascalCase` for exported items.
- **Style**: Follow standard Go idioms. Prefer explicit composition over complex inheritance.

### Testing & Performance
We prioritize performance and stability.
- **Unit Test**: Ensure all logic is covered. Run `go test -v ./...`.
- **Benchmark**: If you change core logic (Worker Pool, Poller, Kafka/DB adaptors), you **MUST** run benchmarks to ensure no performance regression.
  - Run benchmarks: `go test -bench=. -benchmem -v ./...`

## How to Contribute

### Reporting Bugs
*   Check [Issues](https://github.com/sautmanurung/NanoPony/issues) to see if the bug has already been reported.
*   If not, open a new issue. Clearly describe the problem, including environment details and reproduction steps.

### Suggesting Improvements
*   Open a new issue and explain the feature you want to see, the problem it solves, and its potential impact on performance.

### Pull Request
1.  Fork this repository.
2.  Create a new branch (`git checkout -b feature/my-new-feature`).
3.  Apply your changes and add appropriate tests.
4.  Run all tests and benchmarks.
5.  Commit your changes with a clear and descriptive message (`git commit -m 'feat: add support for X'`).
6.  Push to your fork (`git push origin feature/my-new-feature`).
7.  Open a new Pull Request. Ensure you complete the PR template, including CLA agreement.

## Project Structure
- `framework.go`: Main entry point using Builder Pattern.
- `job.go`: Core Job definition and generic Handler.
- `worker.go`: Worker Pool implementation and concurrency management.
- `poller.go`: Periodic data fetching and rate limiting.
- `logger.go`: Structured logging system.
- `database.go` & `kafka.go`: Adaptors for Oracle and Kafka.

## Code of Conduct
Please be respectful, professional, and collaborative in all interactions.

---
Made with ❤️ by the NanoPony team.
