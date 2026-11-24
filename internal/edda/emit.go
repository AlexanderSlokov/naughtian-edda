package edda

import (
	"fmt"
	"slices"
	"strings"
)

// TLA gives the TLA+ module. PROPOSAL.md section 7 gives the rules for this output.
func (s *Spec) TLA() string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	vars := make([]string, len(s.Vars))
	for i, v := range s.Vars {
		vars[i] = v.Name
	}

	w("---- MODULE %s ----\n", s.Module)
	w("\\* Edda translated this module from %s. Do not edit it.\n", s.File)
	w("EXTENDS Naturals, Sequences\n\n")
	w("VARIABLES %s\n", strings.Join(vars, ", "))
	w("vars == <<%s>>\n\n", strings.Join(vars, ", "))
	w("\\* %s:%d state\nInit ==\n", s.File, s.StateLine)
	for _, v := range s.Vars {
		w("    /\\ %s = %s\n", v.Name, v.TLA)
	}

	var steps []string
	for _, k := range s.Blocks {
		w("\n\\* %s:%d %s", s.File, k.Line, k.Kind)
		if k.Label != "" {
			w(" %q", k.Label)
		}
		w("\n")
		if k.Kind != "step" {
			w("%s == %s\n", k.Name, k.Check)
			continue
		}
		steps = append(steps, k.Name)
		w("%s ==\n", k.Name)
		if k.When != "" {
			w("    /\\ %s\n", k.When)
		}
		unchanged := slices.Clone(vars)
		for _, a := range k.After {
			w("    /\\ %s' = %s\n", a.Name, a.TLA)
			unchanged = slices.DeleteFunc(unchanged, func(v string) bool { return v == a.Name })
		}
		if len(unchanged) > 0 {
			w("    /\\ UNCHANGED <<%s>>\n", strings.Join(unchanged, ", "))
		}
	}

	w("\nNext == %s\n\n", strings.Join(steps, ` \/ `))
	w("Spec == Init /\\ [][Next]_vars\n====\n")
	return b.String()
}

// Cfg gives the TLC configuration.
func (s *Spec) Cfg() string {
	out := "SPECIFICATION Spec\n"
	for _, k := range s.Blocks {
		switch k.Kind {
		case "invariant":
			out += "INVARIANT " + k.Name + "\n"
		case "constraint":
			out += "CONSTRAINT " + k.Name + "\n"
		}
	}
	return out
}
