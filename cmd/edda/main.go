// Command edda translates an Edda specification into TLA+ and checks it with TLC.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexanderSlokov/naughtian-edda/internal/edda"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

const usage = `usage:
  edda check spec.hcl   do the static checks, run TLC and show the result
  edda emit spec.hcl    write spec.tla and spec.cfg next to spec.hcl`

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "check" && os.Args[1] != "emit") {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	path := os.Args[2]
	src, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	p := hclparse.NewParser()
	spec, diags := edda.Parse(p, src, path)
	if diags.HasErrors() {
		hcl.NewDiagnosticTextWriter(os.Stderr, p.Files(), 80, false).WriteDiagnostics(diags)
		os.Exit(1)
	}

	if os.Args[1] == "emit" {
		base := strings.TrimSuffix(path, filepath.Ext(path))
		for _, f := range [][2]string{{".tla", spec.TLA()}, {".cfg", spec.Cfg()}} {
			if err := os.WriteFile(base+f[0], []byte(f[1]), 0o644); err != nil {
				fail(err)
			}
			fmt.Println(base + f[0])
		}
		return
	}
	ok, err := edda.Check(spec, edda.JarPath(), os.Stdout)
	if err != nil {
		fail(err)
	}
	if !ok {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
