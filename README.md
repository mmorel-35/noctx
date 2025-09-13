# noctx

![](https://github.com/sonatard/noctx/workflows/CI/badge.svg)

`noctx` finds function calls without context.Context and provides automated fixes.

Passing `context.Context` enables library users to cancel requests, get trace information and so on.

`noctx` helps you identify code that could be rewritten to use context.Context and provides automatic suggested fixes for 21 functions across HTTP, network, exec, and TLS packages.

## Features

- **Detection**: Identifies function calls that should use context.Context but don't
- **Autofix**: Provides automated suggested fixes that can be applied by LSP-compatible editors
- **Smart Context Detection**: Intelligently selects appropriate context based on code environment
- **Method Preservation**: Uses proper HTTP method constants (e.g., `http.MethodGet`) instead of hardcoded strings
- **Variable Assignment Intelligence**: Determines whether to use `:=` or `=` based on variable scope

## Supported Functions

`noctx` provides autofix support for 21 functions across 4 packages:
- **HTTP (5 functions)**: `Get`, `Head`, `Post`, `PostForm`, `NewRequest`
- **Network (13 functions)**: `Dial`, `DialTimeout`, `Listen`, `ListenPacket`, and all 9 `Lookup*` functions
- **Exec (1 function)**: `Command`
- **TLS (2 functions)**: `Dial`, `DialWithDialer`

For detailed information about all supported functions and their transformations, see [Rules Documentation](docs/rules.md).

## Usage

### noctx with go vet

go vet is a Go standard tool for analyzing source code.

1. Install noctx.
```sh
$ go install github.com/sonatard/noctx/cmd/noctx@latest
```

2. Execute noctx
```sh
$ go vet -vettool=`which noctx` main.go
./main.go:6:11: net/http.Get must not be called
```

### noctx with golangci-lint

golangci-lint is a fast Go linters runner.

1. Install golangci-lint.
[golangci-lint - Install](https://golangci-lint.run/usage/install/)

2. Setup .golangci.yml
```yaml:
# Add noctx to enable linters.
linters:
  enable:
    - noctx

# Or enable-all is true.
linters:
  default: all
  disable:
   - xxx # Add unused linter to disable linters.
```

3. Execute noctx
```sh
# Use .golangci.yml
$ golangci-lint run

# Only execute noctx
golangci-lint run --enable-only noctx
The suggested fixes can be applied automatically by editors that support the Language Server Protocol (LSP) or by tools that can process analysis diagnostics with suggested fixes.

## Rules and Examples

For comprehensive documentation of all supported functions, transformation examples, and usage patterns, see [Rules Documentation](docs/rules.md).

## Legacy References

The following sections contain legacy information for reference. Please see the [Rules Documentation](docs/rules.md) for the most up-to-date information about supported functions and their autofix transformations.

### net/http package (Legacy)
### Rules
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/noctx.go#L41-L50

### Sample
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/testdata/src/http_client/http_client.go#L11
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/testdata/src/http_request/http_request.go#L17

### Reference
- [net/http - NewRequest](https://pkg.go.dev/net/http#NewRequest)
- [net/http - NewRequestWithContext](https://pkg.go.dev/net/http#NewRequestWithContext)
- [net/http - Request.WithContext](https://pkg.go.dev/net/http#Request.WithContext)

## net package

### Rules
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/noctx.go#L26-L39

### Sample
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/testdata/src/network/net.go#L17

### References
- [net - ListenConfig](https://pkg.go.dev/net#ListenConfig)
- [net - Dialer.DialContext](https://pkg.go.dev/net#Dialer.DialContext)
- [net - Resolver](https://pkg.go.dev/net#Resolver)
- [net - DefaultResolver](https://pkg.go.dev/net#DefaultResolver)

## database/sql package
### Rules
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/noctx.go#L52-L66

### Sample
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/testdata/src/sql/sql.go#L18

### Reference
- [database/sql](https://pkg.go.dev/database/sql)

## crypt/tls package
### Rules
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/noctx.go#L71-L74

### Sample
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/testdata/src/crypto_tls/tls.go#L17

### Reference
- [crypto/tls - Dialer.DialContext](https://pkg.go.dev/crypto/tls#Dialer.DialContext)
- [crypto/tls - Conn.HandshakeContext](https://pkg.go.dev/crypto/tls#Conn.HandshakeContext)

## exec package
### Rules
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/noctx.go#L68-L69

### Sample
https://github.com/sonatard/noctx/blob/b768dab1764733f7f69c5075b7497eff4c58f260/testdata/src/exec_cmd/exec.go#L11

### Reference
- [exec - exec.CommandContext](https://pkg.go.dev/exec#CommandContext)

