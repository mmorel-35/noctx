# noctx Rules

The `noctx` analyzer detects functions that should use `context.Context` but don't, and provides automated fixes for them.

## Supported Functions

### Package-level Functions with Autofix

#### HTTP Package (5 functions)
- **`http.Get`** → Use `http.NewRequestWithContext` + `http.Client.Do`
- **`http.Head`** → Use `http.NewRequestWithContext` + `http.Client.Do`  
- **`http.Post`** → Use `http.NewRequestWithContext` + `http.Client.Do`
- **`http.PostForm`** → Use `http.NewRequestWithContext` + `http.Client.Do`
- **`http.NewRequest`** → Use `http.NewRequestWithContext`

#### Network Package (13 functions)
- **`net.Dial`** → Use `(*net.Dialer).DialContext`
- **`net.DialTimeout`** → Use `(*net.Dialer).DialContext` with timeout
- **`net.Listen`** → Use `(*net.ListenConfig).Listen`
- **`net.ListenPacket`** → Use `(*net.ListenConfig).ListenPacket`

##### DNS Lookup Functions (9 functions)
- **`net.LookupCNAME`** → Use `(*net.Resolver).LookupCNAME`
- **`net.LookupHost`** → Use `(*net.Resolver).LookupHost`
- **`net.LookupIP`** → Use `(*net.Resolver).LookupIPAddr`
- **`net.LookupPort`** → Use `(*net.Resolver).LookupPort`
- **`net.LookupSRV`** → Use `(*net.Resolver).LookupSRV`
- **`net.LookupMX`** → Use `(*net.Resolver).LookupMX`
- **`net.LookupNS`** → Use `(*net.Resolver).LookupNS`
- **`net.LookupTXT`** → Use `(*net.Resolver).LookupTXT`
- **`net.LookupAddr`** → Use `(*net.Resolver).LookupAddr`

#### Exec Package (1 function)
- **`os/exec.Command`** → Use `exec.CommandContext`

#### TLS Package (2 functions)
- **`crypto/tls.Dial`** → Use `(*tls.Dialer).DialContext`
- **`crypto/tls.DialWithDialer`** → Use `(*tls.Dialer).DialContext`

### Method Functions (Detection Only)

The following method functions are detected but don't have autofix support yet:

#### HTTP Client Methods (4 methods)
- `(*net/http.Client).Get`
- `(*net/http.Client).Head`
- `(*net/http.Client).Post`
- `(*net/http.Client).PostForm`

#### Database/SQL Methods (16 methods)
- `(*database/sql.DB).Begin`, `.Exec`, `.Ping`, `.Prepare`, `.Query`, `.QueryRow`
- `(*database/sql.Tx).Exec`, `.Prepare`, `.Query`, `.QueryRow`, `.Stmt`
- `(*database/sql.Stmt).Exec`, `.Query`, `.QueryRow`

#### TLS Methods (1 method)
- `(*crypto/tls.Conn).Handshake`

## Examples

### HTTP Function Transformations

```go
// Before: http.Get
resp, err := http.Get("https://example.com")

// After: Autofix applied
resp, err := func() (*http.Response, error) {
    req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", http.NoBody)
    if err != nil {
        return nil, err
    }
    return http.DefaultClient.Do(req)
}()
```

```go
// Before: http.NewRequest
req, err := http.NewRequest("GET", "https://example.com", nil)

// After: Autofix applied
req, err := http.NewRequestWithContext(context.Background(), "GET", "https://example.com", http.NoBody)
```

### Network Function Transformations

```go
// Before: net.Dial
conn, err := net.Dial("tcp", "localhost:8080")

// After: Autofix applied
conn, err := func() (net.Conn, error) {
    dialer := &net.Dialer{}
    return dialer.DialContext(context.Background(), "tcp", "localhost:8080")
}()
```

```go
// Before: net.LookupHost
hosts, err := net.LookupHost("example.com")

// After: Autofix applied
hosts, err := func() ([]string, error) {
    resolver := &net.Resolver{}
    return resolver.LookupHost(context.Background(), "example.com")
}()
```

## Context Detection

The analyzer intelligently detects the most appropriate context to use:

1. **Function Parameters**: Uses existing `context.Context` parameters in function signatures
2. **Test Functions**: Uses `t.Context()` when `testing` package is imported
3. **Available Variables**: Uses existing context variables (e.g., `ctx`) when available  
4. **Default Fallback**: Uses `context.Background()` when no context is available

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

Suggested fixes appear in LSP-compatible editors and can be applied automatically.