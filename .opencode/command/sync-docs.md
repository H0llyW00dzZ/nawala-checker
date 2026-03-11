---
description: Synchronize all documentation sources
---

# sync-docs

## Context for AI
This command is used exclusively by opencode.ai to keep documentation consistent after changes to the nawala-checker codebase. When a human runs `/sync-docs`, you (the AI) must proactively analyze changes and update every documentation source while maintaining perfect bilingual consistency.

## When to Run

Invoke this process whenever changes involve:
- New features or API changes
- Modified behavior
- Public interface updates
- CLI command additions or modifications

## Documentation Sources to Sync

### 1. Primary Documentation Files
- `README.md` (English – source of truth)
- `README.id.md` (Indonesian – must stay semantically identical)

### 2. Package-Level GoDoc
- `src/nawala/docs.go` (SDK package)
- `internal/cli/docs.go` (CLI package)

### 3. Code Examples
- `examples/` directory (must remain executable)

### 4. CLI Usage Text
- `internal/cli/usage/` directory (embedded help text)

### 5. Contributing Guides
- `CONTRIBUTING.md` (English)
- `CONTRIBUTING.id.md` (Indonesian)

## Your Sync Workflow (follow exactly)

### Step 0: Analyze Changes

First, examine the git diff / recent code changes to identify every impacted area (new config options, function signatures, CLI flags, error messages, etc.).

### Step 1: Update README Files

Generate complete updated versions of **both** `README.md` and `README.id.md` simultaneously. Keep structure, examples, and technical details identical. Output the diff for human review.

### Step 2: Update GoDoc

Revise `src/nawala/docs.go` and `internal/cli/docs.go` following official Go documentation conventions. Make sure every public API change is documented.

### Step 3: Update Examples

Add or modify files in `examples/` so they demonstrate new behavior. Ensure every example remains fully executable.

### Step 4: Update CLI Usage Text

Update all files in `internal/cli/usage/` to reflect any CLI changes.

### Step 5: Verify Consistency & Correctness

- Run `make test-verbose` (or equivalent) to confirm examples and CLI help still work.
- Double-check that English and Indonesian versions are semantically identical.

## Multilingual Requirements

- Both language versions must remain 100% consistent in meaning, structure, and examples.
- Update both files together — never update only one.
- Preserve technical accuracy in Indonesian while keeping natural language flow.

## Commit Message (use exactly this format)

Use conventional commit format:
```
docs: sync documentation for [description]

- [+] Update README.md with changes
- [+] Update CONTRIBUTING.md and CONTRIBUTING.id.md
- [+] Update GoDoc in docs.go
- [+] Add/modify examples if needed
- [+] Update CLI usage text
```

> [!NOTE]
> When question tools are enabled, always ask the human whether the commit should be created by the AI or left for them.

## Verification Checklist (mark as you complete)
- [ ] README.md updated  
- [ ] README.id.md updated (identical content)  
- [ ] CONTRIBUTING.md updated  
- [ ] CONTRIBUTING.id.md updated (identical content)  
- [ ] GoDoc in both docs.go files updated  
- [ ] Examples directory updated and tested  
- [ ] CLI usage text updated  
- [ ] `make test-verbose` passes
