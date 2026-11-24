package edda

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
)

var (
	distinctStates = regexp.MustCompile(`(\d+) distinct states found`)
	violated       = regexp.MustCompile(`Invariant (\w+) is violated`)
)

// JarPath gives the location of tla2tools.jar. scripts/fetch-tla2tools.sh uses the same default.
func JarPath() string {
	if p := os.Getenv("EDDA_TLA2TOOLS"); p != "" {
		return p
	}
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		home, _ := os.UserHomeDir()
		cache = filepath.Join(home, ".cache")
	}
	return filepath.Join(cache, "edda", "tla2tools.jar")
}

// Check runs TLC on the spec. It writes the result in the language of the spec.
// It returns false if TLC finds a failure.
func Check(s *Spec, jar string, out io.Writer) (bool, error) {
	if _, err := os.Stat(jar); err != nil {
		return false, fmt.Errorf("Edda cannot find tla2tools.jar at %s. Run scripts/fetch-tla2tools.sh, or set EDDA_TLA2TOOLS", jar)
	}
	// TLC writes files into its working directory, so give it a directory of its own.
	dir, err := os.MkdirTemp("", "edda-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, s.Module+".tla"), []byte(s.TLA()), 0o644); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(dir, s.Module+".cfg"), []byte(s.Cfg()), 0o644); err != nil {
		return false, err
	}

	cmd := exec.Command("java", "-XX:+UseParallelGC", "-cp", jar, "tlc2.TLC",
		"-dumpTrace", "json", "trace.json", "-workers", "auto", s.Module)
	cmd.Dir = dir
	output, runErr := cmd.CombinedOutput()
	if runErr == nil {
		n := "?"
		if m := distinctStates.FindSubmatch(output); m != nil {
			n = string(m[1])
		}
		fmt.Fprintf(out, "OK: TLC examined %s different states. TLC found no failure.\n", n)
		return true, nil
	}
	var exit *exec.ExitError
	if !errors.As(runErr, &exit) {
		return false, fmt.Errorf("Edda cannot start TLC: %w", runErr)
	}
	data, err := os.ReadFile(filepath.Join(dir, "trace.json"))
	if err != nil {
		return false, fmt.Errorf("TLC stopped without a trace. The cause can be an error in Edda. TLC output:\n%s", output)
	}
	return false, s.writeTrace(out, data, string(output))
}

type state map[string]json.RawMessage

// writeTrace writes a TLC trace as steps of the spec. See PROPOSAL.md section 8.
func (s *Spec) writeTrace(out io.Writer, data []byte, tlcOutput string) error {
	var tr struct {
		Counterexample struct {
			Action [][3]json.RawMessage `json:"action"`
			State  [][2]json.RawMessage `json:"state"`
		} `json:"counterexample"`
		Vars []string `json:"vars"`
	}
	if err := json.Unmarshal(data, &tr); err != nil {
		return fmt.Errorf("Edda cannot read the TLC trace: %w", err)
	}
	states := make([]state, len(tr.Counterexample.State))
	for i, st := range tr.Counterexample.State {
		if err := json.Unmarshal(st[1], &states[i]); err != nil {
			return fmt.Errorf("Edda cannot read the TLC trace: %w", err)
		}
	}
	if len(states) == 0 {
		return fmt.Errorf("the TLC trace has no states. TLC output:\n%s", tlcOutput)
	}

	tw := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	for i, a := range tr.Counterexample.Action {
		var action struct{ Name string }
		if err := json.Unmarshal(a[1], &action); err != nil {
			return fmt.Errorf("Edda cannot read the TLC trace: %w", err)
		}
		var changes []string
		for _, v := range tr.Vars {
			if before, after := states[i][v], states[i+1][v]; string(before) != string(after) {
				changes = append(changes, fmt.Sprintf("%s %s → %s", v, before, after))
			}
		}
		fmt.Fprintf(tw, "Step %d — %s\t(%s:%d):\t%s\n", i+1, action.Name, s.File, s.block(action.Name).Line, strings.Join(changes, ", "))
	}
	tw.Flush()

	last := states[len(states)-1]
	if m := violated.FindStringSubmatch(tlcOutput); m != nil {
		k := s.block(m[1])
		values := make([]string, len(k.Uses))
		for i, v := range k.Uses {
			values[i] = fmt.Sprintf("%s = %s", v, last[v])
		}
		fmt.Fprintf(out, "FAILURE: %q (%s:%d): %s\n", k.Label, s.File, k.Line, strings.Join(values, ", "))
		return nil
	}
	if strings.Contains(tlcOutput, "Deadlock reached") {
		fmt.Fprintln(out, "FAILURE: deadlock. No step can occur in the last state.")
		return nil
	}
	return fmt.Errorf("TLC found a failure that Edda cannot translate. TLC output:\n%s", tlcOutput)
}

func (s *Spec) block(name string) Block {
	for _, k := range s.Blocks {
		if k.Name == name {
			return k
		}
	}
	return Block{}
}
