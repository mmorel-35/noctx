package main

import (
	"os"
	
	"github.com/sonatard/noctx"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	// Enable debugging
	os.Setenv("DEBUG", "1")
	singlechecker.Main(noctx.Analyzer)
}