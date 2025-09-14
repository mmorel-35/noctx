# Developer Documentation

This document provides technical details for developers working on the `noctx` analyzer.

## Architecture Overview

The analyzer follows a modular architecture inspired by the testifylint pattern with separate checkers per rule type:

### Public Components
- **`analyzer/analyzer.go`**: Public analyzer factory (moved from internal)
- **`noctx.go`**: Main package interface

### Internal Architecture  
- **`internal/checkers/`**: Individual checker implementations (http, net, exec, tls)
- **`internal/registry/`**: Unified function rule registry serving as single source of truth
- **`internal/fixes/`**: Reusable fix generation utilities and Go version detection
- **`internal/helpers/`**: Shared interface checking and analysis utilities

## Code Organization

### Checker Pattern

Each checker implements the `Checker` interface and handles a specific category of functions:

```go
type Checker interface {
    Check(pass *analysis.Pass) error
    Name() string
}
```

**Current Checkers:**
- `HTTPChecker` - HTTP package functions
- `NetChecker` - Network package functions  
- `ExecChecker` - Exec package functions
- `TLSChecker` - TLS package functions

#### Go Version Detection

Proper Go 1.24+ detection for `t.Context()` suggestions:

```go
type GoVersionDetector struct {
    skipGoVersionDetection bool
}

func (g *GoVersionDetector) IsGo124OrGreater(pass *analysis.Pass) bool
```

Uses pattern from [usetesting](https://github.com/ldez/usetesting) for reliable version detection.

#### Factory Pattern

Checkers are instantiated through the factory pattern:

```go
type CheckerFactory func() Checker

var Registry = map[CheckerName]CheckerFactory{
    HTTPCheckerName: func() Checker { return NewHTTPChecker() },
    NetCheckerName:  func() Checker { return NewNetChecker() },
    ExecCheckerName: func() Checker { return NewExecChecker() },
    TLSCheckerName:  func() Checker { return NewTLSChecker() },
}
```

### Registry System

The registry eliminates duplicate maps and serves as the single source of truth:

```go
type FunctionRule struct {
    PackagePath string
    FuncName    string
    FullName    string
    Message     string
    HasAutofix  bool
    CheckerType string
}
```

### Analysis Approach

The analyzer uses a hybrid approach:

1. **SSA Analysis**: For accurate function call detection
2. **AST Analysis**: For precise code generation and position correlation
3. **Position Correlation**: Fallback matching when SSA positions don't match AST nodes

## Implementing New Checkers

### Package-Level Functions

For package-level functions (like `http.Get`), use the `FunctionConfig` pattern:

```go
functions := []FunctionConfig{
    {"net/http", "Get", c.generateHTTPGetFix},
}
return c.checkFunctions(pass, functions)
```

### Method Functions

Method functions require special handling:

1. Use SSA to detect method invocations
2. Match receiver types against expected patterns
3. Correlate SSA instructions with AST nodes for autofix

Example pattern for method detection:

```go
for _, instr := range b.Instrs {
    if call, ok := instr.(*ssa.Call); ok {
        if call.Call.IsInvoke() {
            c.checkMethodCall(pass, call)
        }
    }
}
```

## Fix Generation

### Context Detection Logic

The `ContextDetector` follows this priority:

1. **Function parameter analysis**: Scan function signature for existing `context.Context`
2. **Import analysis**: Detect `testing` package imports for `t.Context()`
3. **Variable scope analysis**: Find existing context variables in scope
4. **Default fallback**: Use `context.Background()`

### Variable Assignment Detection

The `VariableAssignmentDetector` analyzes variable scope to determine `:=` vs `=`:

- Scans current block for existing variable declarations
- Uses `:=` for new variables
- Uses `=` for existing variables

### Code Transformation Patterns

#### Simple Replacement
For straightforward replacements:
```go
// Before: http.NewRequest(method, url, body)
// After:  http.NewRequestWithContext(ctx, method, url, body)
```

#### Function Wrapper Pattern
For complex transformations:
```go
// Before: http.Get(url)
// After:  func() (*http.Response, error) {
//             req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
//             if err != nil { return nil, err }
//             return http.DefaultClient.Do(req)
//         }()
```

## Testing Strategy

### Test Organization
- **Package tests**: Located in `/test` directory
- **Internal tests**: Cover checker factories, registry functionality, and fix generation
- **Integration tests**: Use existing testdata with SSA/AST analysis

### Test Patterns

#### Golden Files
For complex transformations, consider golden file testing:

```go
func TestHTTPGetFix(t *testing.T) {
    // Test autofix generation and application
}
```

#### Registry Testing
Ensure all functions are properly registered:

```go
func TestRegistryCompleteness(t *testing.T) {
    rules := registry.GetAllRules()
    // Verify expected functions are present
}
```

## Adding New Functions

### Step 1: Register Function
Add to `internal/registry/registry.go`:

```go
"package/path.FunctionName": {
    PackagePath: "package/path",
    FuncName:    "FunctionName", 
    FullName:    "package/path.FunctionName",
    Message:     "must not be called. use ContextAwareAlternative",
    HasAutofix:  true,
    CheckerType: "appropriate_checker",
},
```

### Step 2: Implement Fix Generator
Add to appropriate checker:

```go
func (c *MyChecker) generateMyFunctionFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
    // Implementation
}
```

### Step 3: Add Test Coverage
Create test cases in the appropriate test files.

## Performance Considerations

### SSA Analysis Overhead
- SSA analysis is expensive but necessary for accuracy
- Consider caching SSA results for repeated analysis
- Limit scope of analysis when possible

### AST Traversal Optimization
- Use `inspector.Preorder()` with specific node filters
- Collect nodes in single pass when possible
- Avoid repeated AST traversals

### Memory Management
- Avoid storing large AST/SSA structures beyond analysis scope
- Use position-based correlation instead of retaining full nodes

## Error Handling

### Graceful Degradation
- Always provide fallback to basic reporting when autofix fails
- Don't fail analysis due to autofix generation errors
- Log diagnostic information for debugging

### Position Correlation
- Handle cases where SSA and AST positions don't align perfectly
- Implement fuzzy matching for instruction correlation
- Validate position ranges before generating fixes

## Best Practices

### Code Generation
- Preserve semantic meaning (use `http.MethodGet` instead of `"GET"`)
- Optimize body parameters (`http.NoBody` instead of `nil`)
- Maintain consistent formatting and style

### Diagnostic Messages
- Use clear, actionable messages
- Include specific replacement suggestions
- Maintain consistency across similar functions

### Backward Compatibility
- Never break existing functionality when adding new features
- Maintain support for all previously supported functions
- Test against real-world codebases