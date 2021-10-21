package version

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"
)

//go:embed VERSION
var VERSION string

var ver = flag.Bool("version", false, "show version")

func Version() string {
	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	return strings.TrimSpace(VERSION)
}
