# 🤖 Nawala Checker — AI Skills

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25.6-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Baca dalam Bahasa Indonesia](https://img.shields.io/badge/🇮🇩-Baca%20dalam%20Bahasa%20Indonesia-red)](README.id.md)

This directory contains **skill definitions** that allow AI bots and agents to integrate with nawala-checker. Each skill provides structured instructions for invoking nawala-checker via the **CLI** (any language) or the **Go SDK** (Go-based tools).

## What Are Skills?

Skills are standardized instruction files (`SKILL.md`) that tell AI agents how to use a tool. When an AI bot is pointed at this directory, it can discover and invoke nawala-checker's capabilities automatically — checking domains, monitoring DNS health, and managing configuration.

## Available Skills

| Skill | Description |
|---|---|
| [`check_domains`](check_domains/SKILL.md) | Check whether domains are blocked by Indonesian ISP DNS filters |
| [`dns_status`](dns_status/SKILL.md) | Check the health and latency of configured DNS servers |
| [`config_inspect`](config_inspect/SKILL.md) | Inspect effective configuration and generate config files |

## Prerequisites

### CLI (Required for CLI integration)

```bash
go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
```

### Go SDK (Required for SDK integration)

```bash
go get github.com/H0llyW00dzZ/nawala-checker
```

> [!IMPORTANT]
> This SDK requires an **Indonesian network** to function correctly. Nawala DNS servers only return blocking responses when queried from within Indonesia.

## Setup

### 1. Clone the Repository

Clone the nawala-checker repository to get the skills directory:

```bash
git clone https://github.com/H0llyW00dzZ/nawala-checker.git
```

### 2. Point Your AI Agent at the Skills Directory

Configure your AI framework to read from the cloned `skills/` directory:

- **[openclaw](https://openclaw.ai)** — Add the skills directory path to your agent config:
  ```
  nawala-checker/skills/
  ```
- **[opencode](https://opencode.ai)** — Copy or symlink the `skills/` directory into your project (auto-discovered)
- **[crush](https://github.com/charmbracelet/crush)** — Reference each `SKILL.md` as a tool definition
- **Generic agents** — Point the agent at the `skills/` directory or individual `SKILL.md` files

### 3. Verify the CLI is Available

Ensure the `nawala` binary is on your `PATH`:

```bash
nawala --version
```

If the command is not found, install it:

```bash
go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
```

### 4. Start Using Skills

Ask your AI agent to check domains:

```
> Check if reddit.com and google.com are blocked by Indonesian DNS
```

The agent will discover the `check_domains` skill and run:

```bash
nawala check --format json reddit.com google.com
```

## Integration Methods

Each skill supports two integration approaches:

| Method | Language | Best For |
|---|---|---|
| **CLI** | Any | Python, TypeScript, or any non-Go AI framework |
| **Go SDK** | Go | Go-based AI tools (opencode, openclaw, crush, etc.) |

### CLI Example

```bash
# AI agent runs this command and parses JSON output
nawala check --format json google.com reddit.com
```

```json
{
  "nawala": {
    "version": "0.7.1",
    "result": [
      {"domain": "google.com", "blocked": false, "server": "180.131.144.144"},
      {"domain": "reddit.com", "blocked": true, "server": "180.131.144.144"}
    ]
  }
}
```

### Go SDK Example

```go
checker := nawala.New()
defer checker.Close()

results, _ := checker.Check(ctx, "google.com", "reddit.com")
for _, r := range results {
    fmt.Printf("%s: blocked=%v\n", r.Domain, r.Blocked)
}
```

## Directory Structure

```
skills/
├── README.md                   # This file (English)
├── README.id.md                # Indonesian version
├── check_domains/
│   └── SKILL.md                # Domain blocking check skill
├── dns_status/
│   └── SKILL.md                # DNS server health skill
└── config_inspect/
    └── SKILL.md                # Configuration inspection skill
```
