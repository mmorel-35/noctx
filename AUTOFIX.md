# noctx Autofix Documentation

## Overview

The `noctx` analyzer now includes autofix functionality that provides suggested code fixes for functions that should use `context.Context` but don't.

## Supported Autofixes

### HTTP Functions

#### `http.NewRequest` → `http.NewRequestWithContext`

**Before:**
```go
req, err := http.NewRequest("GET", "https://example.com", nil)
```

**After:**
```go
req, err := http.NewRequestWithContext(context.Background(), "GET", "https://example.com", http.NoBody)
```

#### `http.Get` → `http.NewRequestWithContext` + `Client.Do`

**Before:**
```go
resp, err := http.Get("https://example.com")
```

**After:**
```go
resp, err := func() (*http.Response, error) {
    req, err := http.NewRequestWithContext(context.Background(), "GET", "https://example.com", http.NoBody)
    if err != nil {
        return nil, err
    }
    return http.DefaultClient.Do(req)
}()
```

#### `http.Head` → `http.NewRequestWithContext` + `Client.Do`

Similar to `http.Get` but uses "HEAD" method.

#### `http.Post` → `http.NewRequestWithContext` + `Client.Do`

**Before:**
```go
resp, err := http.Post("https://example.com", "application/json", body)
```

**After:**
```go
resp, err := func() (*http.Response, error) {
    req, err := http.NewRequestWithContext(context.Background(), "POST", "https://example.com", body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    return http.DefaultClient.Do(req)
}()
```

## Context Detection

The analyzer intelligently selects the appropriate context based on the code environment:

1. **Test functions**: Uses `t.Context()` when `testing` package is imported
2. **Functions with context parameters**: Uses existing context parameter (e.g., `ctx`)
3. **Default**: Uses `context.Background()`

## Body Parameter Handling

- `nil` body parameters are automatically replaced with `http.NoBody`
- Other body parameters are preserved as-is

## Usage

### With go vet
```bash
go vet -vettool=$(which noctx) ./...
```

### With golangci-lint
Add to `.golangci.yml`:
```yaml
linters:
  enable:
    - noctx
```

Then run:
```bash
golangci-lint run
```

The suggested fixes can be applied automatically by editors that support the Language Server Protocol (LSP) or by tools that can process analysis diagnostics with suggested fixes.

## Examples

### Simple case
```go
// Input
func download() {
    req, _ := http.NewRequest("GET", "https://api.example.com", nil)
    // ...
}

// Suggested fix
func download() {
    req, _ := http.NewRequestWithContext(context.Background(), "GET", "https://api.example.com", http.NoBody)
    // ...
}
```

### Test function
```go
// Input
func TestAPI(t *testing.T) {
    req, _ := http.NewRequest("GET", "https://api.example.com", nil)
    // ...
}

// Suggested fix
func TestAPI(t *testing.T) {
    req, _ := http.NewRequestWithContext(t.Context(), "GET", "https://api.example.com", http.NoBody)
    // ...
}
```

### Function with context parameter
```go
// Input
func fetchData(ctx context.Context) {
    req, _ := http.NewRequest("GET", "https://api.example.com", nil)
    // ...
}

// Suggested fix (when context detection is improved)
func fetchData(ctx context.Context) {
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com", http.NoBody)
    // ...
}
```