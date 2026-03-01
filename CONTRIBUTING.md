# Contributing to Nawala Checker

[![Baca dalam Bahasa Indonesia](https://img.shields.io/badge/🇮🇩-Baca%20dalam%20Bahasa%20Indonesia-red)](CONTRIBUTING.id.md)

Thank you for your interest in contributing to the **Nawala Checker** SDK! This repository adheres to strict Go SDK development standards. We welcome contributions that improve reliability, performance, or documentation.

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before participating in our community.

## Project Structure

This project follows a canonical Go SDK layout to ensure idiomatic usage and minimal dependency overhead.

```text
nawala-checker/
├── src/
│   └── nawala/      # Core DNS checking logic, options, typed structs, and cache.
├── examples/        # Executable examples (basic, custom, status, hotreload).
├── .github/         # CI/CD workflows and GitHub templates.
├── Makefile         # Build, test, and coverage commands.
├── README.md        # Primary English documentation.
└── README.id.md     # Localized Indonesian documentation.
```

## Setup and Verification

To ensure a clean contribution process, please follow the Fork-First Workflow:

1. **Fork the repository** to your own GitHub account.
2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/nawala-checker.git
   cd nawala-checker
   ```
3. **Verify your setup** by running the test suite:
   ```bash
   make test-verbose
   ```
   *Note: If you are not on an Indonesian network, some live DNS tests might fail or be skipped. You can run unit tests only using `make test-short`.*

## Contribution Lifecycle

### 1. Branching
Create a dedicated branch for your work. Use descriptive prefixes:
*   `feature/` for new capabilities (e.g., `feature/redis-cache`)
*   `fix/` for bug fixes (e.g., `fix/edns0-parsing`)
*   `docs/` for documentation updates
*   `chore/` for maintenance (e.g., CI/CD updates)

```bash
git checkout -b feature/your-feature-name
```

### 2. Making Changes

**Code Standards**:
*   Ensure all new configuration options use the **Functional Options** pattern in `src/nawala/option.go`.
*   All methods performing I/O must accept `context.Context` as the first argument.
*   Avoid adding third-party dependencies unless absolutely necessary.

**Testing**:
*   We require tests for all new code paths.
*   Check your test coverage locally before submitting:
    ```bash
    make test-cover
    ```

**Documentation (Multilingual Sync)**:
*   The `nawala-checker` maintains both English (`README.md`) and Indonesian (`README.id.md`) documentation.
*   If your pull request adds a new feature, changes the public API, or modifies existing behavior, you **must update both `README.md` and `README.id.md`**.

### 3. Committing and Formatting
Before committing, ensure your code is properly formatted:
```bash
gofmt -s -w ./src/...
```

Write clear, descriptive commit messages. We encourage Conventional Commits:
```
feat: add custom EDNS0 size configuration
fix: resolve race condition in cache expiration
docs: update hot-reload example in README
```

### 4. Opening a Pull Request
1. Push your branch to your fork.
2. Open a Pull Request against the `master` branch of the upstream repository.
3. Fill out the PR template, describing what you changed and why.
4. The CI pipeline will automatically lint, format, and run the test suite across multiple Go versions with the race detector enabled.

## Code Review
Maintainers will review your PR. We may request changes to align with the core architectures described in our standards (Functional Options, Context-First, Typed Errors). Once approved and all CI checks pass, your PR will be merged!

---
*By contributing to this repository, you agree that your contributions will be licensed under the project's [BSD 3-Clause License](LICENSE).*
