// Package edda translates an Edda specification (HCL) into TLA+.
// PROPOSAL.md sections 5 and 6 give the design.
package edda

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Spec is the Spec AST. The expressions in it are already in TLA+.
type Spec struct {
	Module    string
	File      string // base name of the source, for reference comments
	StateLine int
	Vars      []Assign // initial values, in source order
	Blocks    []Block  // steps, invariants and constraints, in source order
}

type Assign struct {
	Name string
	TLA  string
}

type Block struct {
	Kind  string // "step", "invariant" or "constraint"
	Label string // step name or invariant sentence
	Name  string // TLA+ definition name
	Line  int
	When  string   // step only
	After []Assign // step only
	Check string   // invariant and constraint only
	Uses  []string // variables that Check reads, for the failure line
}

var ident = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var schema = &hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{
	{Type: "state"},
	{Type: "step", LabelNames: []string{"name"}},
	{Type: "invariant", LabelNames: []string{"sentence"}},
	{Type: "constraint"},
}}

// Parse reads the source and does the static checks of PROPOSAL.md section 5.3.
func Parse(p *hclparse.Parser, src []byte, filename string) (*Spec, hcl.Diagnostics) {
	f, diags := p.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}
	content, diags := f.Body.Content(schema)
	base := filepath.Base(filename)
	s := &Spec{File: base, Module: strings.TrimSuffix(base, filepath.Ext(base))}
	if !ident.MatchString(s.Module) {
		diags = append(diags, &hcl.Diagnostic{Severity: hcl.DiagError, Summary: "Incorrect file name",
			Detail: fmt.Sprintf("The file name %q must be a TLA+ module name. Use letters, digits and _ only.", base)})
	}
	t := &translator{src: src, names: map[string]bool{"Init": true, "Next": true, "Spec": true, "vars": true}}

	// Read the state block first, because all other blocks refer to its variables.
	for _, b := range content.Blocks.OfType("state") {
		if s.StateLine != 0 {
			t.errorf(b.DefRange, "Two state blocks", "There is only one initial state. Put all variables in one state block.")
			continue
		}
		s.StateLine = b.DefRange.Start.Line
		attrs, d := b.Body.JustAttributes()
		t.diags = append(t.diags, d...)
		for _, a := range sorted(attrs) {
			// An initial value is a constant, so the translator knows no variables yet.
			s.Vars = append(s.Vars, Assign{a.Name, t.expr(a.Expr.(hclsyntax.Expression))})
		}
		for _, v := range s.Vars {
			t.vars = append(t.vars, v.Name)
			t.define(v.Name, b.DefRange)
		}
	}
	if len(s.Vars) == 0 {
		t.errorf(f.Body.MissingItemRange(), "No state", "Add a state block with one or more variables.")
		return nil, append(diags, t.diags...)
	}

	steps := 0
	for _, b := range content.Blocks {
		k := Block{Kind: b.Type, Line: b.DefRange.Start.Line}
		switch b.Type {
		case "step":
			steps++
			k.Label, k.Name = b.Labels[0], b.Labels[0]
			t.define(k.Name, b.LabelRanges[0])
			c, d := b.Body.Content(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "when"}, {Name: "after"}}})
			t.diags = append(t.diags, d...)
			if a := c.Attributes["when"]; a != nil {
				k.When = t.expr(a.Expr.(hclsyntax.Expression))
			}
			if a := c.Attributes["after"]; a != nil {
				k.After = t.after(a.Expr.(hclsyntax.Expression))
			}
		case "invariant":
			k.Label = b.Labels[0]
			if !strings.Contains(strings.TrimSpace(k.Label), " ") {
				t.errorf(b.LabelRanges[0], "Invariant name is not a sentence",
					"Write the rule as a sentence in human language. Example: \"an acknowledged record is durable\".")
			}
			k.Name = camel(k.Label)
			t.define(k.Name, b.LabelRanges[0])
			k.Check, k.Uses = t.check(b)
		case "constraint":
			// shortcut: one constraint block only.
			// TLC accepts more, but v0 has no use for them and they need names.
			// Upgrade when a real specification needs two constraints.
			k.Name = "Constraint"
			if !t.define(k.Name, b.DefRange) {
				continue
			}
			k.Check, _ = t.check(b)
		default:
			continue
		}
		s.Blocks = append(s.Blocks, k)
	}
	if steps == 0 {
		t.errorf(f.Body.MissingItemRange(), "No steps", "Add one or more step blocks.")
	}
	return s, append(diags, t.diags...)
}

type translator struct {
	src   []byte          // the source, to quote an operator in an error
	vars  []string        // the state variables
	names map[string]bool // TLA+ names that are in use
	used  []string        // the variables that the current expression reads
	diags hcl.Diagnostics
}

func (t *translator) errorf(r hcl.Range, summary, detail string, args ...any) {
	t.diags = append(t.diags, &hcl.Diagnostic{Severity: hcl.DiagError, Summary: summary,
		Detail: fmt.Sprintf(detail, args...), Subject: r.Ptr()})
}

// define reserves a TLA+ name. It returns false if the name is not available.
func (t *translator) define(name string, r hcl.Range) bool {
	switch {
	case !ident.MatchString(name):
		t.errorf(r, "Incorrect name", "%q is not a TLA+ name. Start with a letter, then use letters, digits or _.", name)
	case t.names[name]:
		t.errorf(r, "Name used two times", "The TLA+ output already uses the name %q. Select a different name.", name)
	default:
		t.names[name] = true
		return true
	}
	return false
}

func (t *translator) check(b *hcl.Block) (string, []string) {
	c, d := b.Body.Content(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "check", Required: true}}})
	t.diags = append(t.diags, d...)
	if c.Attributes["check"] == nil {
		return "", nil
	}
	t.used = nil
	return t.expr(c.Attributes["check"].Expr.(hclsyntax.Expression)), t.used
}

func (t *translator) after(e hclsyntax.Expression) []Assign {
	obj, ok := e.(*hclsyntax.ObjectConsExpr)
	if !ok {
		t.errorf(e.Range(), "Incorrect after", "Write after as { variable = value }.")
		return nil
	}
	var out []Assign
	for _, item := range obj.Items {
		name := hcl.ExprAsKeyword(item.KeyExpr)
		switch {
		case !slices.Contains(t.vars, name):
			t.errorf(item.KeyExpr.Range(), "Unknown variable", "after sets %q, but the state block has no variable with this name.", name)
		case slices.ContainsFunc(out, func(a Assign) bool { return a.Name == name }):
			t.errorf(item.KeyExpr.Range(), "Variable set two times", "after sets %q two times. A step gives one value to each variable.", name)
		}
		out = append(out, Assign{name, t.expr(item.ValueExpr)})
	}
	return out
}

var binaryOps = map[*hclsyntax.Operation]string{
	hclsyntax.OpAdd:                "+",
	hclsyntax.OpLessThan:           "<",
	hclsyntax.OpGreaterThan:        ">",
	hclsyntax.OpLessThanOrEqual:    "<=",
	hclsyntax.OpGreaterThanOrEqual: ">=",
}

var functions = map[string]struct {
	args int
	tla  string
}{
	"length": {1, "Len(%s)"},
	"append": {2, "Append(%s, %s)"},
	"concat": {2, `%s \o %s`},
}

// expr translates one expression word for word. README.md lists the vocabulary.
func (t *translator) expr(e hclsyntax.Expression) string {
	switch e := e.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.Number {
			if f := e.Val.AsBigFloat(); f.IsInt() && f.Sign() >= 0 {
				return f.Text('f', 0)
			}
		}
	case *hclsyntax.TemplateExpr:
		if s, ok := literalString(e); ok {
			return `"` + s + `"`
		}
	case *hclsyntax.ScopeTraversalExpr:
		name := e.Traversal.RootName()
		if len(e.Traversal) == 1 {
			if t.vars == nil {
				t.errorf(e.SrcRange, "Initial value refers to a variable", "An initial value in the state block must be a constant.")
				return name
			}
			if !slices.Contains(t.vars, name) {
				t.errorf(e.SrcRange, "Unknown variable", "%q is not a variable in the state block.", name)
			} else if !slices.Contains(t.used, name) {
				t.used = append(t.used, name)
			}
			return name
		}
	case *hclsyntax.TupleConsExpr:
		parts := make([]string, len(e.Exprs))
		for i, x := range e.Exprs {
			parts[i] = t.expr(x)
		}
		return "<<" + strings.Join(parts, ", ") + ">>"
	case *hclsyntax.ParenthesesExpr:
		return "(" + t.expr(e.Expression) + ")"
	case *hclsyntax.BinaryOpExpr:
		if op, ok := binaryOps[e.Op]; ok {
			return t.expr(e.LHS) + " " + op + " " + t.expr(e.RHS)
		}
		// HCL does not keep the operator symbol, but the source between the operands is the symbol.
		opRange := hcl.Range{Filename: e.SrcRange.Filename, Start: e.LHS.Range().End, End: e.RHS.Range().Start}
		op := strings.TrimSpace(string(opRange.SliceBytes(t.src)))
		t.errorf(opRange, "Word not in the approved vocabulary", "%q is not in the approved vocabulary. Refer to the vocabulary in README.md.", op)
		return ""
	case *hclsyntax.FunctionCallExpr:
		f, ok := functions[e.Name]
		if !ok {
			t.errorf(e.NameRange, "Word not in the approved vocabulary", "%q is not in the approved vocabulary. Refer to the vocabulary in README.md.", e.Name)
			return ""
		}
		if len(e.Args) != f.args {
			t.errorf(e.Range(), "Incorrect number of arguments", "%s must have %d arguments.", e.Name, f.args)
			return ""
		}
		args := make([]any, len(e.Args))
		for i, x := range e.Args {
			args[i] = t.expr(x)
		}
		return fmt.Sprintf(f.tla, args...)
	}
	t.errorf(e.Range(), "Word not in the approved vocabulary", "Edda version 0 cannot translate this expression. Refer to the vocabulary in README.md.")
	return ""
}

// literalString gives the text of a string without interpolation.
func literalString(e *hclsyntax.TemplateExpr) (string, bool) {
	var b strings.Builder
	for _, p := range e.Parts {
		lit, ok := p.(*hclsyntax.LiteralValueExpr)
		if !ok || lit.Val.Type() != cty.String {
			return "", false
		}
		b.WriteString(lit.Val.AsString())
	}
	// shortcut: refuse " and \ instead of escaping them.
	// The TLA+ output then needs no escape rules, and the MVP strings are plain words.
	// Upgrade when a real specification needs these characters.
	return b.String(), !strings.ContainsAny(b.String(), `"\`)
}

// camel makes a TLA+ name from a sentence: "an acknowledged record" gives AnAcknowledgedRecord.
func camel(sentence string) string {
	words := strings.FieldsFunc(sentence, func(r rune) bool {
		return !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9')
	})
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, "")
}

func sorted(attrs hcl.Attributes) []*hcl.Attribute {
	out := make([]*hcl.Attribute, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, a)
	}
	slices.SortFunc(out, func(a, b *hcl.Attribute) int { return a.Range.Start.Byte - b.Range.Start.Byte })
	return out
}
