# sync-docs

Synchronize all documentation sources when making changes to the nawala-checker codebase. This command ensures consistency across multilingual documentation, GoDoc, examples, and CLI usage text.

## When to Run

Run this command when your changes involve:
- New features or API changes
- Modified existing behavior
- Public interface updates
- CLI command additions/modifications

## Documentation Sources to Sync

### 1. Primary Documentation Files
- `README.md` - English documentation (primary)
- `README.id.md` - Indonesian documentation (localized)

### 2. Package-Level GoDoc
- `src/nawala/docs.go` - Core SDK package documentation
- `internal/cli/docs.go` - CLI package documentation

### 3. Code Examples
- `examples/` directory - Executable examples that demonstrate usage

### 4. CLI Usage Text
- `internal/cli/usage/` directory - Embedded CLI help text

### 5. Contributing Guides
- `CONTRIBUTING.md` - English contributing guidelines
- `CONTRIBUTING.id.md` - Indonesian contributing guidelines (localized)

## Sync Process

### Step 1: Identify Changes
Review your code changes to determine what documentation needs updating:
- New configuration options?
- Changed function signatures?
- New CLI commands or flags?
- Modified behavior or error conditions?

### Step 2: Update README Files
Update both language versions:
```bash
# Edit README.md (English)
# Edit README.id.md (Indonesian - maintain consistency)
```

### Step 3: Update GoDoc
Ensure package documentation reflects new/changed APIs:
```bash
# Edit src/nawala/docs.go for SDK changes
# Edit internal/cli/docs.go for CLI changes
```

### Step 4: Update Examples
Add or modify examples in `examples/` directory to demonstrate new features.

### Step 5: Update CLI Usage
For CLI changes, update embedded usage text in `internal/cli/usage/`.

### Step 6: Verify Consistency
Run tests to ensure all documentation sources are technically accurate:
```bash
make test-verbose
```

## Multilingual Requirements

Since this project maintains both English and Indonesian documentation:
- Keep technical accuracy consistent between languages
- Update both README files simultaneously
- Maintain the same structure and examples in both languages

## Commit Message

Use conventional commit format:
```
docs: sync documentation for [description]

- [+] Update README.md with changes
- [+] Update CONTRIBUTING.md and CONTRIBUTING.id.md
- [+] Update GoDoc in docs.go
- [+] Add/modify examples if needed
- [+] Update CLI usage text
```

## Verification Checklist

- [ ] README.md updated with new features/changes
- [ ] README.id.md updated (Indonesian)
- [ ] CONTRIBUTING.md updated
- [ ] CONTRIBUTING.id.md updated (Indonesian)
- [ ] GoDoc in docs.go files updated
- [ ] Examples directory updated if needed
- [ ] CLI usage text updated for CLI changes
