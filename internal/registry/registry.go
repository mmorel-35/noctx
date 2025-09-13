package registry

// FunctionRule defines a rule for a specific function
type FunctionRule struct {
	// Function identification
	PackagePath string
	FuncName    string
	FullName    string // Computed from PackagePath and FuncName
	
	// Rule properties
	Message     string
	HasAutofix  bool
	CheckerType string
}

// Rules contains all function rules with autofix support
var Rules = map[string]*FunctionRule{
	// HTTP Package Functions
	"net/http.Get": {
		PackagePath: "net/http",
		FuncName:    "Get",
		FullName:    "net/http.Get",
		Message:     "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		HasAutofix:  true,
		CheckerType: "http",
	},
	"net/http.Head": {
		PackagePath: "net/http",
		FuncName:    "Head",
		FullName:    "net/http.Head",
		Message:     "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		HasAutofix:  true,
		CheckerType: "http",
	},
	"net/http.Post": {
		PackagePath: "net/http",
		FuncName:    "Post",
		FullName:    "net/http.Post",
		Message:     "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		HasAutofix:  true,
		CheckerType: "http",
	},
	"net/http.PostForm": {
		PackagePath: "net/http",
		FuncName:    "PostForm",
		FullName:    "net/http.PostForm",
		Message:     "must not be called. use net/http.NewRequestWithContext and (*net/http.Client).Do(*http.Request)",
		HasAutofix:  true,
		CheckerType: "http",
	},
	"net/http.NewRequest": {
		PackagePath: "net/http",
		FuncName:    "NewRequest",
		FullName:    "net/http.NewRequest",
		Message:     "must not be called. use net/http.NewRequestWithContext",
		HasAutofix:  true,
		CheckerType: "http",
	},
	
	// Network Package Functions
	"net.Dial": {
		PackagePath: "net",
		FuncName:    "Dial",
		FullName:    "net.Dial",
		Message:     "must not be called. use (*net.Dialer).DialContext",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.DialTimeout": {
		PackagePath: "net",
		FuncName:    "DialTimeout",
		FullName:    "net.DialTimeout",
		Message:     "must not be called. use (*net.Dialer).DialContext with (*net.Dialer).Timeout",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.Listen": {
		PackagePath: "net",
		FuncName:    "Listen",
		FullName:    "net.Listen",
		Message:     "must not be called. use (*net.ListenConfig).Listen",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.ListenPacket": {
		PackagePath: "net",
		FuncName:    "ListenPacket",
		FullName:    "net.ListenPacket",
		Message:     "must not be called. use (*net.ListenConfig).ListenPacket",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupCNAME": {
		PackagePath: "net",
		FuncName:    "LookupCNAME",
		FullName:    "net.LookupCNAME",
		Message:     "must not be called. use (*net.Resolver).LookupCNAME with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupHost": {
		PackagePath: "net",
		FuncName:    "LookupHost",
		FullName:    "net.LookupHost",
		Message:     "must not be called. use (*net.Resolver).LookupHost with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupIP": {
		PackagePath: "net",
		FuncName:    "LookupIP",
		FullName:    "net.LookupIP",
		Message:     "must not be called. use (*net.Resolver).LookupIPAddr with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupPort": {
		PackagePath: "net",
		FuncName:    "LookupPort",
		FullName:    "net.LookupPort",
		Message:     "must not be called. use (*net.Resolver).LookupPort with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupSRV": {
		PackagePath: "net",
		FuncName:    "LookupSRV",
		FullName:    "net.LookupSRV",
		Message:     "must not be called. use (*net.Resolver).LookupSRV with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupMX": {
		PackagePath: "net",
		FuncName:    "LookupMX",
		FullName:    "net.LookupMX",
		Message:     "must not be called. use (*net.Resolver).LookupMX with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupNS": {
		PackagePath: "net",
		FuncName:    "LookupNS",
		FullName:    "net.LookupNS",
		Message:     "must not be called. use (*net.Resolver).LookupNS with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupTXT": {
		PackagePath: "net",
		FuncName:    "LookupTXT",
		FullName:    "net.LookupTXT",
		Message:     "must not be called. use (*net.Resolver).LookupTXT with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	"net.LookupAddr": {
		PackagePath: "net",
		FuncName:    "LookupAddr",
		FullName:    "net.LookupAddr",
		Message:     "must not be called. use (*net.Resolver).LookupAddr with a context",
		HasAutofix:  true,
		CheckerType: "net",
	},
	
	// Exec Package Functions
	"os/exec.Command": {
		PackagePath: "os/exec",
		FuncName:    "Command",
		FullName:    "os/exec.Command",
		Message:     "must not be called. use os/exec.CommandContext",
		HasAutofix:  true,
		CheckerType: "exec",
	},
	
	// TLS Package Functions
	"crypto/tls.Dial": {
		PackagePath: "crypto/tls",
		FuncName:    "Dial",
		FullName:    "crypto/tls.Dial",
		Message:     "must not be called. use (*crypto/tls.Dialer).DialContext",
		HasAutofix:  true,
		CheckerType: "tls",
	},
	"crypto/tls.DialWithDialer": {
		PackagePath: "crypto/tls",
		FuncName:    "DialWithDialer",
		FullName:    "crypto/tls.DialWithDialer",
		Message:     "must not be called. use (*crypto/tls.Dialer).DialContext with NetDialer",
		HasAutofix:  true,
		CheckerType: "tls",
	},
	"(*crypto/tls.Conn).Handshake": {
		PackagePath: "crypto/tls",
		FuncName:    "Handshake",
		FullName:    "(*crypto/tls.Conn).Handshake",
		Message:     "must not be called. use (*crypto/tls.Conn).HandshakeContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	
	// HTTP Client methods (will be handled by fallback for now)
	"(*net/http.Client).Get": {
		PackagePath: "net/http",
		FuncName:    "Get",
		FullName:    "(*net/http.Client).Get",
		Message:     "must not be called. use (*net/http.Client).Do(*http.Request)",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*net/http.Client).Head": {
		PackagePath: "net/http",
		FuncName:    "Head",
		FullName:    "(*net/http.Client).Head",
		Message:     "must not be called. use (*net/http.Client).Do(*http.Request)",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*net/http.Client).Post": {
		PackagePath: "net/http",
		FuncName:    "Post",
		FullName:    "(*net/http.Client).Post",
		Message:     "must not be called. use (*net/http.Client).Do(*http.Request)",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*net/http.Client).PostForm": {
		PackagePath: "net/http",
		FuncName:    "PostForm",
		FullName:    "(*net/http.Client).PostForm",
		Message:     "must not be called. use (*net/http.Client).Do(*http.Request)",
		HasAutofix:  false,
		CheckerType: "",
	},
	
	// Database/SQL functions (will be handled by fallback for now)
	"(*database/sql.DB).Begin": {
		PackagePath: "database/sql",
		FuncName:    "Begin",
		FullName:    "(*database/sql.DB).Begin",
		Message:     "must not be called. use (*database/sql.DB).BeginTx",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.DB).Exec": {
		PackagePath: "database/sql",
		FuncName:    "Exec",
		FullName:    "(*database/sql.DB).Exec",
		Message:     "must not be called. use (*database/sql.DB).ExecContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.DB).Ping": {
		PackagePath: "database/sql",
		FuncName:    "Ping",
		FullName:    "(*database/sql.DB).Ping",
		Message:     "must not be called. use (*database/sql.DB).PingContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.DB).Prepare": {
		PackagePath: "database/sql",
		FuncName:    "Prepare",
		FullName:    "(*database/sql.DB).Prepare",
		Message:     "must not be called. use (*database/sql.DB).PrepareContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.DB).Query": {
		PackagePath: "database/sql",
		FuncName:    "Query",
		FullName:    "(*database/sql.DB).Query",
		Message:     "must not be called. use (*database/sql.DB).QueryContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.DB).QueryRow": {
		PackagePath: "database/sql",
		FuncName:    "QueryRow",
		FullName:    "(*database/sql.DB).QueryRow",
		Message:     "must not be called. use (*database/sql.DB).QueryRowContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Tx).Exec": {
		PackagePath: "database/sql",
		FuncName:    "Exec",
		FullName:    "(*database/sql.Tx).Exec",
		Message:     "must not be called. use (*database/sql.Tx).ExecContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Tx).Prepare": {
		PackagePath: "database/sql",
		FuncName:    "Prepare",
		FullName:    "(*database/sql.Tx).Prepare",
		Message:     "must not be called. use (*database/sql.Tx).PrepareContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Tx).Query": {
		PackagePath: "database/sql",
		FuncName:    "Query",
		FullName:    "(*database/sql.Tx).Query",
		Message:     "must not be called. use (*database/sql.Tx).QueryContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Tx).QueryRow": {
		PackagePath: "database/sql",
		FuncName:    "QueryRow",
		FullName:    "(*database/sql.Tx).QueryRow",
		Message:     "must not be called. use (*database/sql.Tx).QueryRowContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Tx).Stmt": {
		PackagePath: "database/sql",
		FuncName:    "Stmt",
		FullName:    "(*database/sql.Tx).Stmt",
		Message:     "must not be called. use (*database/sql.Tx).StmtContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Stmt).Exec": {
		PackagePath: "database/sql",
		FuncName:    "Exec",
		FullName:    "(*database/sql.Stmt).Exec",
		Message:     "must not be called. use (*database/sql.Stmt).ExecContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Stmt).Query": {
		PackagePath: "database/sql",
		FuncName:    "Query",
		FullName:    "(*database/sql.Stmt).Query",
		Message:     "must not be called. use (*database/sql.Stmt).QueryContext",
		HasAutofix:  false,
		CheckerType: "",
	},
	"(*database/sql.Stmt).QueryRow": {
		PackagePath: "database/sql",
		FuncName:    "QueryRow",
		FullName:    "(*database/sql.Stmt).QueryRow",
		Message:     "must not be called. use (*database/sql.Stmt).QueryRowContext",
		HasAutofix:  false,
		CheckerType: "",
	},
}

// GetAllRules returns all rules (all now have autofix support)
func GetAllRules() map[string]*FunctionRule {
	allRules := make(map[string]*FunctionRule)
	
	// Add all rules (all have autofix support now)
	for name, rule := range Rules {
		allRules[name] = rule
	}
	
	return allRules
}

// GetRulesByChecker returns rules grouped by checker type
func GetRulesByChecker() map[string][]*FunctionRule {
	grouped := make(map[string][]*FunctionRule)
	
	for _, rule := range Rules {
		if rule.HasAutofix {
			grouped[rule.CheckerType] = append(grouped[rule.CheckerType], rule)
		}
	}
	
	return grouped
}

// GetAutofixFunctions returns a map of function names that have autofix support
func GetAutofixFunctions() map[string]bool {
	functions := make(map[string]bool)
	for name, rule := range Rules {
		if rule.HasAutofix {
			functions[name] = true
		}
	}
	return functions
}

// GetMessage returns the diagnostic message for a function
func GetMessage(funcName string) string {
	if rule, exists := Rules[funcName]; exists {
		return rule.Message
	}
	return "must not be called without context"
}

// FormatDiagnostic formats a diagnostic message with the function name
func FormatDiagnostic(funcName string) string {
	return funcName + " " + GetMessage(funcName)
}