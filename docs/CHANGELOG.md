# Changelog

All notable changes to the Infrastructure-Agent Telemetry Foundation will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Process Scanner implementation with the following features:
  - Efficient process collection on multiple platforms (Linux, Windows, macOS)
  - Event-based detection of process creation, updates, and termination
  - Adaptive scanning rate based on system load
  - Configurable inclusion/exclusion filters
  - Thread-safe consumer notification system
  - Self-monitoring for CPU and memory usage
- Core collector module with interfaces and configuration
- Consumer registry for process event distribution
- Platform abstraction for OS-specific implementations
- Top-N Process Sampler implementation with the following features:
  - Min-heap based process tracking with O(log N) operations
  - Adaptive scoring based on weighted CPU and RSS metrics
  - Circuit breaker pattern for self-protection under high load
  - Optimized handling of high process churn (up to 2000 PIDs/s)
  - Comprehensive metrics for monitoring capture ratio and performance
- Core sampler module with interfaces and configuration
- Metrics tracking system for sampler performance monitoring
- DDSketch implementation for statistical aggregation:
  - Guaranteed relative error bounds (≤1% for p95/p99)
  - Memory-efficient sparse and dense store implementations
  - Automatic store type switching based on data distribution
  - Thread-safe operations for concurrent access
  - Merging support for distributed aggregation
  - Full serialization support for transport
- Core sketch module with interfaces and configuration

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

## [0.1.0] - YYYY-MM-DD
### Added
- Initial project structure
- Bootstrap scripts and environment setup
- Task management system
- Blueprint alignment matrix
