# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive Makefile with build automation, testing, and release targets
- GitHub Actions CI/CD pipeline with multi-platform builds
- Docker support with multi-stage builds and non-root user
- Professional README.md with badges and mermaid diagrams
- CONTRIBUTING.md with detailed development guidelines
- MIT LICENSE
- golangci-lint configuration
- CHANGELOG.md for tracking changes
- .editorconfig for consistent code formatting
- .dockerignore for optimized Docker builds

### Changed
- Updated Go toolchain from 1.23.0 to 1.24.0
- Updated all dependencies to latest versions:
  - goquery: v1.10.2 → v1.11.0
  - golang.org/x/time: v0.5.0 → v0.14.0
  - golang.org/x/net: v0.37.0 → v0.47.0
  - golang.org/x/text: v0.23.0 → v0.31.0
  - And other transitive dependencies

### Improved
- Project structure and organization
- Documentation with mermaid architecture diagrams
- Build process automation
- Code quality tooling

## [10.0.0] - 2025-11-02

### Added
- Two-tier HTML processing architecture
  - Fast path: Byte-level scanning for simple pages (<50μs)
  - Slow path: Full DOM parsing for complex pages (<500μs)
- Intelligent routing based on link density threshold (0.05)
- Dynamic path selection for 3-5x throughput improvement
- Tokenizer package with coordinator, fastpath, and slowpath modules

### Changed
- Restructured crawler to use two-tier processing
- Enhanced monitoring to track fast/slow path statistics
- Improved performance metrics and reporting

### Performance
- 3.6x faster throughput compared to single-tier (v9)
- 35% CPU usage reduction
- 10% memory usage improvement
- 90% of pages processed via fast path
- Average processing: 45μs (fast), 487μs (slow)

## [9.0.0] - 2025-11-02

### Added
- Modular architecture with 8 specialized packages
- Multi-NIC support for parallel network utilization
- Auto-scaling worker pool (100→800 workers)
- Real-time performance monitoring
- Priority queue system for failed downloads
- Comprehensive logging system

### Packages
- `config`: Centralized configuration
- `network`: Multi-NIC detection and management
- `downloader`: Worker pool and task queuing
- `crawler`: Web crawling with Colly
- `monitor`: Performance monitoring and auto-scaling
- `system`: System optimizations
- `utils`: Utility functions

### Features
- Network interface auto-detection
- Intelligent load balancing across NICs
- Dynamic worker scaling based on queue utilization
- Memory-aware GC management
- Document type detection (PDF, DOC, ZIP, etc.)

### Performance
- Tested on AMD Ryzen 9 5950X with 2x 10GbE
- Successfully crawled 78,109 URLs in 15 seconds
- ~5,000 URLs/sec throughput
- Proper queue management and error handling

## [8.0.0] and earlier

Previous versions were experimental and not formally released.

### Development History
- Versions 1-8: Iterative development and testing
- Focus on basic crawling functionality
- Single network interface support
- Monolithic architecture

---

## Version Naming

- **v10**: Two-Tier Edition - Intelligent dual-path processing
- **v9**: Modular Edition - Clean package separation
- **v8 and earlier**: Experimental versions

## Migration Guides

- [v9 to v10](docs/MIGRATION_GUIDE.md): Upgrading to two-tier architecture

## Links

- [Repository](https://github.com/yourusername/url_crawler_twotier)
- [Issue Tracker](https://github.com/yourusername/url_crawler_twotier/issues)
- [Documentation](docs/)

---

**Legend:**
- `Added` for new features
- `Changed` for changes in existing functionality
- `Deprecated` for soon-to-be removed features
- `Removed` for now removed features
- `Fixed` for any bug fixes
- `Security` for vulnerability fixes
- `Performance` for performance improvements
