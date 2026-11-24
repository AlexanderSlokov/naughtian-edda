package edda

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var update = flag.Bool("update", false, "write the golden files again")

func parseFile(t *testing.T, path string) *Spec {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s, diags := Parse(hclparse.NewParser(), src, path)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	return s
}

// TestGolden translates the PROPOSAL.md section 6 specification.
func TestGolden(t *testing.T) {
	s := parseFile(t, "testdata/log.hcl")
	for _, f := range [][2]string{{"testdata/log.tla", s.TLA()}, {"testdata/log.cfg", s.Cfg()}} {
		if *update {
			if err := os.WriteFile(f[0], []byte(f[1]), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(f[0])
		if err != nil {
			t.Fatal(err)
		}
		if f[1] != string(want) {
			t.Errorf("%s is different. Run go test -update and examine the diff.\ngot:\n%s", f[0], f[1])
		}
	}
}

// TestVocabulary has one approved example for each word in PROPOSAL.md section 6.2.
func TestVocabulary(t *testing.T) {
	for _, c := range []struct{ hcl, tla string }{
		{`length(log)`, `Len(log)`},
		{`append(buffer, "msg")`, `Append(buffer, "msg")`},
		{`concat(log, buffer)`, `log \o buffer`},
		{`acked + 1`, `acked + 1`},
		{`length(log) > acked`, `Len(log) > acked`},
		{`acked <= length(log)`, `acked <= Len(log)`},
		{`(acked + 1) <= 3`, `(acked + 1) <= 3`},
		{`[]`, `<<>>`},
		{`["a", "b"]`, `<<"a", "b">>`},
	} {
		e, diags := hclsyntax.ParseExpression([]byte(c.hcl), "test.hcl", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatal(diags)
		}
		tr := &translator{vars: []string{"log", "buffer", "acked"}}
		if got := tr.expr(e); got != c.tla || tr.diags.HasErrors() {
			t.Errorf("%s: got %q, want %q, %v", c.hcl, got, c.tla, tr.diags)
		}
	}
}

// TestStaticChecks has one specification for each error that Edda must find.
func TestStaticChecks(t *testing.T) {
	const ok = "state {\n  x = 0\n}\nstep \"Inc\" {\n  after = { x = x + 1 }\n}\n"
	for _, c := range []struct{ name, src, want string }{
		{"two state blocks", ok + "state {\n  y = 0\n}\n", "Two state blocks"},
		{"no state", "step \"A\" {\n}\n", "No state"},
		{"no steps", "state {\n  x = 0\n}\n", "No steps"},
		{"initial value refers to a variable", "state {\n  x = 0\n  y = x\n}\nstep \"A\" {\n}\n", "Initial value refers to a variable"},
		{"unknown variable in when", ok + "step \"B\" {\n  when = y > 0\n}\n", "Unknown variable"},
		{"after sets unknown variable", ok + "step \"B\" {\n  after = { y = 1 }\n}\n", "Unknown variable"},
		{"invariant uses unknown variable", ok + "invariant \"x is small\" {\n  check = y <= 3\n}\n", "Unknown variable"},
		{"after sets a variable two times", ok + "step \"B\" {\n  after = { x = 1, x = 2 }\n}\n", "Variable set two times"},
		{"after is not an object", ok + "step \"B\" {\n  after = x\n}\n", "Incorrect after"},
		{"unknown function", ok + "invariant \"x is small\" {\n  check = size(x) <= 3\n}\n", "Word not in the approved vocabulary"},
		{"unknown operator", ok + "invariant \"x is small\" {\n  check = x < 3\n}\n", "Word not in the approved vocabulary"},
		{"negative number", ok + "invariant \"x is small\" {\n  check = x <= -1\n}\n", "Word not in the approved vocabulary"},
		{"wrong number of arguments", ok + "invariant \"x is small\" {\n  check = length(x, x) <= 3\n}\n", "Incorrect number of arguments"},
		{"invariant name is not a sentence", ok + "invariant \"Inv1\" {\n  check = x <= 3\n}\n", "Invariant name is not a sentence"},
		{"step name is not a TLA+ name", ok + "step \"do it\" {\n}\n", "Incorrect name"},
		{"step name used two times", ok + "step \"Inc\" {\n}\n", "Name used two times"},
		{"step name is a generated name", ok + "step \"Next\" {\n}\n", "Name used two times"},
		{"two constraint blocks", ok + "constraint {\n  check = x <= 3\n}\nconstraint {\n  check = x <= 2\n}\n", "Name used two times"},
		{"constraint without check", ok + "constraint {\n}\n", "Missing required argument"},
		{"unknown block", ok + "procedure \"A\" {\n}\n", "Unsupported block type"},
		{"unknown attribute in a step", ok + "step \"B\" {\n  then = 1\n}\n", "Unsupported argument"},
		{"incorrect file name", ok, "Incorrect file name"},
	} {
		filename := "spec.hcl"
		if c.want == "Incorrect file name" {
			filename = "my-spec.hcl"
		}
		_, diags := Parse(hclparse.NewParser(), []byte(c.src), filename)
		if !strings.Contains(diags.Error(), c.want) {
			t.Errorf("%s: want %q, got %v", c.name, c.want, diags)
		}
	}
}

// TestCheck does the three steps of PROPOSAL.md section 9 with TLC.
func TestCheck(t *testing.T) {
	if _, err := os.Stat(JarPath()); err != nil {
		t.Skip("no tla2tools.jar. Run scripts/fetch-tla2tools.sh")
	}
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("no java")
	}
	var out strings.Builder
	if ok, err := Check(parseFile(t, "testdata/log.hcl"), JarPath(), &out); !ok || err != nil {
		t.Fatalf("correct spec: ok=%v err=%v\n%s", ok, err, out.String())
	}

	out.Reset()
	ok, err := Check(parseFile(t, "testdata/log_bug.hcl"), JarPath(), &out)
	if ok || err != nil {
		t.Fatalf("spec with the bug: ok=%v err=%v", ok, err)
	}
	want := `Step 1 — Write       (log_bug.hcl:7):  buffer [] → ["msg"]
Step 2 — Acknowledge (log_bug.hcl:18): acked 0 → 1
FAILURE: "an acknowledged record is durable" (log_bug.hcl:27): acked = 1, log = []
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}
