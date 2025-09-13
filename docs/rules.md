# noctx Rules Documentation

This document describes all the rules implemented by the `noctx` analyzer. The analyzer detects functions that should use `context.Context` but don't, and provides automated fixes for them.

## Overview

The `noctx` analyzer identifies 21 functions across 4 packages that lack proper context support and provides suggested fixes to modernize the code.

## HTTP Package Rules

### http.Get
**Problem**: `http.Get` doesn't accept a context parameter.  
**Solution**: Replace with `http.NewRequestWithContext` + `http.Client.Do`  
**Method Preservation**: Uses `http.MethodGet` constant  

### http.Head  
**Problem**: `http.Head` doesn't accept a context parameter.  
**Solution**: Replace with `http.NewRequestWithContext` + `http.Client.Do`  
**Method Preservation**: Uses `http.MethodHead` constant  

### http.Post
**Problem**: `http.Post` doesn't accept a context parameter.  
**Solution**: Replace with `http.NewRequestWithContext` + `http.Client.Do`  
**Method Preservation**: Uses `http.MethodPost` constant  

### http.PostForm
**Problem**: `http.PostForm` doesn't accept a context parameter.  
**Solution**: Replace with `http.NewRequestWithContext` + `http.Client.Do` with form encoding  
**Method Preservation**: Uses `http.MethodPost` constant  

### http.NewRequest
**Problem**: `http.NewRequest` doesn't accept a context parameter.  
**Solution**: Replace with `http.NewRequestWithContext`  
**Body Optimization**: Replaces `nil` body with `http.NoBody`  

## Network Package Rules

### net.Dial
**Problem**: `net.Dial` doesn't accept a context parameter.  
**Solution**: Replace with `(*net.Dialer).DialContext`  

### net.DialTimeout  
**Problem**: `net.DialTimeout` doesn't accept a context parameter.  
**Solution**: Replace with `(*net.Dialer).DialContext` with timeout configuration  

### net.Listen
**Problem**: `net.Listen` doesn't accept a context parameter.  
**Solution**: Replace with `(*net.ListenConfig).Listen`  

### net.ListenPacket
**Problem**: `net.ListenPacket` doesn't accept a context parameter.  
**Solution**: Replace with `(*net.ListenConfig).ListenPacket`  

### Lookup Functions (9 functions)
All lookup functions lack context support:
- `net.LookupCNAME` → `(*net.Resolver).LookupCNAME`
- `net.LookupHost` → `(*net.Resolver).LookupHost`  
- `net.LookupIP` → `(*net.Resolver).LookupIPAddr`
- `net.LookupPort` → `(*net.Resolver).LookupPort`
- `net.LookupSRV` → `(*net.Resolver).LookupSRV`
- `net.LookupMX` → `(*net.Resolver).LookupMX`
- `net.LookupNS` → `(*net.Resolver).LookupNS`
- `net.LookupTXT` → `(*net.Resolver).LookupTXT`
- `net.LookupAddr` → `(*net.Resolver).LookupAddr`

## Exec Package Rules

### os/exec.Command
**Problem**: `os/exec.Command` doesn't accept a context parameter.  
**Solution**: Replace with `exec.CommandContext`  

## TLS Package Rules

### crypto/tls.Dial
**Problem**: `crypto/tls.Dial` doesn't accept a context parameter.  
**Solution**: Replace with `(*tls.Dialer).DialContext`  

### crypto/tls.DialWithDialer
**Problem**: `crypto/tls.DialWithDialer` doesn't accept a context parameter.  
**Solution**: Replace with `(*tls.Dialer).DialContext` with NetDialer configuration  

## Context Detection Strategy

The analyzer uses intelligent context detection:

1. **Function Parameters**: Analyzes function signature for existing `context.Context` parameters
2. **Test Functions**: Uses `t.Context()` when `testing` package is imported  
3. **Available Variables**: Uses existing context variables (e.g., `ctx`) when available
4. **Default Fallback**: Uses `context.Background()` when no context is available

## Variable Assignment Intelligence

The analyzer determines whether to use `:=` or `=` based on variable scope analysis:
- Uses `:=` when variables need to be declared
- Uses `=` when variables are already declared in the current scope

## Examples

### HTTP Get Transformation
```go
// Before
resp, err := http.Get("https://example.com")

// After (in test function)
resp, err := func() (*http.Response, error) {
    req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
    if err != nil {
        return nil, err
    }
    return http.DefaultClient.Do(req)
}()
```

### Network Dial Transformation
```go
// Before  
conn, err := net.Dial("tcp", "localhost:8080")

// After (with existing context parameter)
conn, err := func() (net.Conn, error) {
    dialer := &net.Dialer{}
    return dialer.DialContext(ctx, "tcp", "localhost:8080")
}()
```

### HTTP NewRequest Transformation
```go
// Before
req, err := http.NewRequest("GET", "https://example.com", nil)

// After 
req, err := http.NewRequestWithContext(context.Background(), "GET", "https://example.com", http.NoBody)
```

## Usage

The analyzer integrates with standard Go tooling:

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

Suggested fixes appear in LSP-compatible editors and can be applied automatically, making it easy to modernize code to use proper context handling.