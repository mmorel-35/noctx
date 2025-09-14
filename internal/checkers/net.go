package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/sonatard/noctx/internal/fixes"
	"github.com/sonatard/noctx/internal/helpers"
	"github.com/sonatard/noctx/internal/registry"
)

// NetChecker detects network functions that should use context
type NetChecker struct {
	ctx *helpers.CheckerContext
}

// NewNetChecker creates a new network function checker
func NewNetChecker() *NetChecker {
	return &NetChecker{
		ctx: helpers.NewCheckerContext(),
	}
}

// Name returns the name of this checker
func (c *NetChecker) Name() string {
	return "net"
}

// Check performs the analysis for network functions
func (c *NetChecker) Check(pass *analysis.Pass) error {
	// Get network rules from registry
	rulesByChecker := registry.GetRulesByChecker()
	
	netRules, exists := rulesByChecker["net"]
	if !exists {
		return nil
	}

	// Convert rules to function configs
	netFunctions := make([]helpers.FunctionConfig, 0, len(netRules))
	for _, rule := range netRules {
		if rule.HasAutofix {
			netFunctions = append(netFunctions, helpers.FunctionConfig{
				PackagePath: rule.PackagePath,
				FuncName:    rule.FuncName,
				Handler:     c.getNetHandler(rule.FullName),
			})
		}
	}

	// Check functions with autofix support
	if len(netFunctions) > 0 {
		if err := helpers.CheckFunctionsWithAutofix(pass, netFunctions, c.ctx); err != nil {
			return err
		}
	}

	return nil
}

// getNetHandler returns the appropriate fix handler for a network function
func (c *NetChecker) getNetHandler(funcName string) func(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	switch funcName {
	case "net.Dial":
		return c.generateNetDialFix
	case "net.DialTimeout":
		return c.generateNetDialTimeoutFix
	case "net.Listen":
		return c.generateNetListenFix
	case "net.ListenPacket":
		return c.generateNetListenPacketFix
	case "net.LookupAddr":
		return c.generateNetLookupAddrFix
	case "net.LookupCNAME":
		return c.generateNetLookupCNAMEFix
	case "net.LookupHost":
		return c.generateNetLookupHostFix
	case "net.LookupIP":
		return c.generateNetLookupIPFix
	case "net.LookupMX":
		return c.generateNetLookupMXFix
	case "net.LookupNS":
		return c.generateNetLookupNSFix
	case "net.LookupPort":
		return c.generateNetLookupPortFix
	case "net.LookupSRV":
		return c.generateNetLookupSRVFix
	case "net.LookupTXT":
		return c.generateNetLookupTXTFix
	default:
		return nil
	}
}

func (c *NetChecker) generateNetDialFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	
	newCall := fmt.Sprintf("(&net.Dialer{}).DialContext(%s, %s, %s)", contextExpr, networkArg, addressArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext", callExpr, newCall)
}

func (c *NetChecker) generateNetDialTimeoutFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	timeoutArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	
	newCall := fmt.Sprintf("(&net.Dialer{Timeout: %s}).DialContext(%s, %s, %s)", timeoutArg, contextExpr, networkArg, addressArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Dialer).DialContext with timeout", callExpr, newCall)
}

func (c *NetChecker) generateNetListenFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	
	newCall := fmt.Sprintf("(&net.ListenConfig{}).Listen(%s, %s, %s)", contextExpr, networkArg, addressArg)
	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).Listen", callExpr, newCall)
}

func (c *NetChecker) generateNetListenPacketFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}

	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	addressArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	
	newCall := fmt.Sprintf("(&net.ListenConfig{}).ListenPacket(%s, %s, %s)", contextExpr, networkArg, addressArg)
	return fixes.CreateSuggestedFix("Replace with (*net.ListenConfig).ListenPacket", callExpr, newCall)
}

// Lookup function generators
func (c *NetChecker) generateNetLookupAddrFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	addrArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupAddr(%s, %s)", contextExpr, addrArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupAddr", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupCNAMEFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	hostArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupCNAME(%s, %s)", contextExpr, hostArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupCNAME", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupHostFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	hostArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupHost(%s, %s)", contextExpr, hostArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupHost", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupIPFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	hostArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupIPAddr(%s, %s)", contextExpr, hostArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupIPAddr", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupMXFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	nameArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupMX(%s, %s)", contextExpr, nameArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupMX", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupNSFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	nameArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupNS(%s, %s)", contextExpr, nameArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupNS", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupPortFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 2 {
		return nil
	}
	networkArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	serviceArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupPort(%s, %s, %s)", contextExpr, networkArg, serviceArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupPort", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupSRVFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 3 {
		return nil
	}
	serviceArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	protoArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[1])
	nameArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[2])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupSRV(%s, %s, %s, %s)", contextExpr, serviceArg, protoArg, nameArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupSRV", callExpr, newCall)
}

func (c *NetChecker) generateNetLookupTXTFix(pass *analysis.Pass, callExpr *ast.CallExpr, contextExpr string) *analysis.SuggestedFix {
	if len(callExpr.Args) != 1 {
		return nil
	}
	nameArg := c.ctx.ArgFormatter.FormatArgument(pass, callExpr.Args[0])
	newCall := fmt.Sprintf("(&net.Resolver{}).LookupTXT(%s, %s)", contextExpr, nameArg)
	return fixes.CreateSuggestedFix("Replace with (*net.Resolver).LookupTXT", callExpr, newCall)
}