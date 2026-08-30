package goml

import (
	"fmt"
	"strings"
)

// gomlBuiltins maps goml's predeclared type spellings to their Go (or
// goplus) spellings.
var gomlBuiltins = map[string]string{
	"Int": "int", "Int8": "int8", "Int16": "int16", "Int32": "int32",
	"Int64": "int64", "UInt": "uint", "UInt8": "uint8", "UInt16": "uint16",
	"UInt32": "uint32", "UInt64": "uint64", "Float32": "float32",
	"Float64": "float64", "Bool": "bool", "String": "string",
	"Byte": "byte", "Rune": "rune", "Error": "error", "Any": "any",
	"Nat": "nat", "Mult": "mult",
}

// goPredeclared are lowercase names that are concrete Go types, never
// goml type variables.
var goPredeclared = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true,
	"uint64": true, "uintptr": true, "float32": true, "float64": true,
	"complex64": true, "complex128": true, "bool": true, "string": true,
	"byte": true, "rune": true, "error": true, "any": true,
	"nat": true, "mult": true,
}

type ctorInfo struct {
	arity  int
	parens bool   // print Name() for nullary uses (pinned-result ctors)
	fields []Type // field types, for expected-type threading at calls
}

type enumInfo struct {
	// binder names and sorts, in declaration order; sort is "any",
	// "nat", or a domain/constraint spelling.
	params []string
	sorts  []string
	ctors  map[string]*ctorInfo
}

// opSig is one class operation's parameter/result types (arrows split).
type opSig struct {
	params []Type
	result Type
}

// classInfo records a local class for instance-member type inference.
type classInfo struct {
	binder  string // the class type variable
	extends []string
	ops     map[string]opSig
}

type converter struct {
	path       string
	file       *File
	b          *strings.Builder
	enums      map[string]*enumInfo
	classes    map[string]*classInfo
	nullaryOps map[string]bool // class operations with no parameters
	fw         *fnWriter       // active function writer (for hoisting)
	lines      map[int]int     // emitted .gp line → source .goml line
	holes      map[string]Pos  // hole name → position of its `?`

	// Expected-type indexes: local let value-parameter types and record
	// field types, threaded to unannotated lambdas and list literals.
	letSigs map[string][]Type
	records map[string]map[string]Type
	retWant Type // result type of the let body being rendered

	whereOwner map[string]string // where-helper name → owning let
	opens      map[string]string // opened member name → package qualifier
	localNames map[string]bool   // file-local declaration names (win over opens)
}

// mark records that output emitted from here on originates at pos. The
// map is consulted with a floor lookup, so decl/statement granularity is
// enough to bring diagnostics back to a useful .goml line.
func (c *converter) mark(pos Pos) {
	if pos.Line > 0 {
		c.lines[strings.Count(c.b.String(), "\n")+1] = pos.Line
	}
}

// SourceLine maps a line of the transpiled text back to the .goml line
// it was emitted from (floor lookup; 0 when unknown).
func SourceLine(lines map[int]int, gpLine int) int {
	best := 0
	bestKey := -1
	for k, v := range lines {
		if k <= gpLine && k > bestKey {
			bestKey, best = k, v
		}
	}
	return best
}

func (c *converter) failf(pos Pos, format string, args ...any) {
	panic(bailout{&Error{Path: c.path, Pos: pos, Msg: fmt.Sprintf(format, args...)}})
}

// ConvertInfo carries the source-mapping byproducts of a conversion.
type ConvertInfo struct {
	// Lines maps an emitted .gp line to the .goml line it came from.
	Lines map[int]int
	// Holes maps each typed hole's name to the position of its `?` in the
	// .goml source. Holes are named, so their diagnostics can be placed
	// exactly rather than through the line map.
	Holes map[string]Pos
}

// Convert transpiles one .goml source file to .gp text.
func Convert(path string, src []byte) ([]byte, error) {
	out, _, err := ConvertWithInfo(path, src)
	return out, err
}

// ConvertWithMap also returns the emitted-line → source-line map used to
// bring pipeline diagnostics back to .goml positions.
func ConvertWithMap(path string, src []byte) ([]byte, map[int]int, error) {
	out, info, err := ConvertWithInfo(path, src)
	if err != nil {
		return nil, nil, err
	}
	return out, info.Lines, nil
}

// ConvertWithInfo transpiles one .goml source file, also reporting what is
// needed to map the pipeline's diagnostics back onto it.
func ConvertWithInfo(path string, src []byte) (out []byte, info *ConvertInfo, err error) {
	file, perr := Parse(path, src)
	if perr != nil {
		return nil, nil, perr
	}
	c := &converter{
		path: path, file: file, b: &strings.Builder{},
		enums: map[string]*enumInfo{}, classes: map[string]*classInfo{},
		nullaryOps: map[string]bool{}, lines: map[int]int{},
		holes:   map[string]Pos{},
		letSigs: map[string][]Type{}, records: map[string]map[string]Type{},
		whereOwner: map[string]string{},
		opens:      map[string]string{}, localNames: map[string]bool{},
	}
	defer func() {
		if r := recover(); r != nil {
			b, ok := r.(bailout)
			if !ok {
				panic(r)
			}
			out, info, err = nil, nil, b.err
		}
	}()
	c.indexEnums()
	c.printFile()
	return []byte(c.b.String()), &ConvertInfo{Lines: c.lines, Holes: c.holes}, nil
}

// indexEnums records local sum types: binder sorts (for nat/type
// classification) and constructor shapes (for pattern/expr printing).
func (c *converter) indexEnums() {
	for _, o := range c.file.Opens {
		for _, n := range o.Names {
			if pkg, dup := c.opens[n]; dup && pkg != o.Pkg {
				c.failf(o.Pos, "%s is exposed by both %s and %s; qualify its uses", n, pkg, o.Pkg)
			}
			c.opens[n] = o.Pkg
		}
	}
	for _, d := range c.file.Decls {
		switch d := d.(type) {
		case *LetDecl:
			c.localNames[d.Name] = true
		case *TypeDecl:
			c.localNames[d.Name] = true
			for _, ct := range d.Sum {
				c.localNames[ct.Name] = true
			}
		case *ClassDecl:
			c.localNames[d.Name] = true
			for _, op := range d.Ops {
				c.localNames[op.Name] = true
			}
		case *InstanceDecl:
			if d.Name != "" {
				c.localNames[d.Name] = true
			}
		}
	}
	for _, d := range c.file.Decls {
		if cd, ok := d.(*ClassDecl); ok {
			info := &classInfo{binder: cd.Binder.Names[0], ops: map[string]opSig{}}
			for _, e := range cd.Extends {
				if app, ok := e.(*TypeApp); ok && app.Head.Pkg == "" {
					info.extends = append(info.extends, app.Head.Name)
				}
			}
			for _, op := range cd.Ops {
				var sig opSig
				if op.Sig != nil {
					t := op.Sig
					for {
						arrow, ok := t.(*TypeArrow)
						if !ok {
							break
						}
						sig.params = append(sig.params, arrow.From)
						t = arrow.To
					}
					sig.result = t
				} else {
					for _, b := range op.Binders {
						for range b.Names {
							sig.params = append(sig.params, b.Type)
						}
					}
					sig.result = op.Result
				}
				info.ops[op.Name] = sig
				if len(sig.params) == 0 {
					c.nullaryOps[op.Name] = true
				}
			}
			c.classes[cd.Name] = info
			continue
		}
		if ld, ok := d.(*LetDecl); ok {
			c.indexLetSig(ld)
			for _, h := range ld.Where {
				if owner, dup := c.whereOwner[h.Name]; dup {
					c.failf(h.Pos, "where-helper %s already defined under %s; helper names are file-unique (they lower to package-private functions)", h.Name, owner)
				}
				c.whereOwner[h.Name] = ld.Name
				c.indexLetSig(h)
			}
			continue
		}
		td, ok := d.(*TypeDecl)
		if !ok {
			continue
		}
		if len(td.Record) > 0 {
			fields := map[string]Type{}
			for _, f := range td.Record {
				fields[f.Name] = f.Type
			}
			c.records[td.Name] = fields
			continue
		}
		if len(td.Sum) == 0 {
			continue
		}
		info := &enumInfo{ctors: map[string]*ctorInfo{}}
		for _, b := range td.Binders {
			sort := c.binderSort(b)
			for _, n := range b.Names {
				info.params = append(info.params, n)
				info.sorts = append(info.sorts, sort)
			}
		}
		for i, k := range td.Kind {
			sort := c.sortOf(k)
			name := c.kindSlotName(td, len(td.Binders), i, sort)
			info.params = append(info.params, name)
			info.sorts = append(info.sorts, sort)
		}
		for _, ct := range td.Sum {
			ci := &ctorInfo{arity: len(ct.Fields), parens: ct.Result != nil}
			for _, f := range ct.Fields {
				ci.fields = append(ci.fields, f.Type)
			}
			info.ctors[ct.Name] = ci
		}
		c.enums[td.Name] = info
	}
}

func (c *converter) binderSort(b *Binder) string {
	return c.sortOf(b.Type)
}

func (c *converter) sortOf(t Type) string {
	n, ok := t.(*TypeName)
	if !ok {
		c.failf(t.typePos(), "unsupported binder sort")
	}
	switch {
	case n.Pkg == "" && n.Name == "Type":
		return "any"
	case n.Pkg == "" && n.Name == "Nat":
		return "nat"
	case n.Pkg == "" && n.Name == "Mult":
		return "mult"
	default:
		return c.typeString(n) // index domain or constraint
	}
}

// kindSlotName finds the variable name constructors use for kind slot i
// (the i-th index parameter after the declared binders).
func (c *converter) kindSlotName(td *TypeDecl, declared, slot int, sort string) string {
	pos := declared + slot
	name := ""
	for _, ct := range td.Sum {
		app, ok := ct.Result.(*TypeApp)
		if !ok || len(app.Args) <= pos {
			continue
		}
		for _, v := range c.freeIndexVars(app.Args[pos], nil) {
			if name == "" {
				name = v
			} else if name != v {
				c.failf(ct.Pos, "constructors of %s disagree on the index variable for slot %d (%s vs %s); use one name", td.Name, slot+1, name, v)
			}
		}
	}
	// Constructor results may pin every position concretely (Expr Int,
	// Expr Bool), naming no variable; synthesize one in the slot's sort
	// so the generated parameter reads correctly.
	if name == "" {
		stem := "n"
		if sort == "any" {
			stem = "a"
		}
		name = stem
		if slot > 0 {
			name = fmt.Sprintf("%s%d", stem, slot)
		}
	}
	return name
}

// freeIndexVars lists lowercase variables inside an index term.
func (c *converter) freeIndexVars(t Type, acc []string) []string {
	switch t := t.(type) {
	case *TypeName:
		if t.Pkg == "" && !isUpperName(t.Name) && !goPredeclared[t.Name] {
			acc = append(acc, t.Name)
		}
	case *TypeIndexOp:
		acc = c.freeIndexVars(t.L, acc)
		acc = c.freeIndexVars(t.R, acc)
	}
	return acc
}

// ------------------------------------------------------------- file/top

func (c *converter) printFile() {
	f := c.file
	for _, a := range f.Attrs {
		if line := c.directiveFor(a); line != "" {
			c.b.WriteString(line + "\n")
		}
	}
	c.b.WriteString(fmt.Sprintf("package %s\n", f.Module))
	if len(f.Imports) > 0 {
		c.b.WriteString("\nimport (\n")
		for _, imp := range f.Imports {
			if imp.Alias != "" {
				fmt.Fprintf(c.b, "\t%s %s\n", imp.Alias, imp.Path)
			} else {
				fmt.Fprintf(c.b, "\t%s\n", imp.Path)
			}
		}
		c.b.WriteString(")\n")
	}
	for _, d := range f.Decls {
		c.b.WriteString("\n")
		c.printDecl(d, nil)
	}
}

func (c *converter) printDoc(doc []string, indent string) {
	for _, line := range doc {
		if line == "" {
			fmt.Fprintf(c.b, "%s//\n", indent)
		} else {
			fmt.Fprintf(c.b, "%s//%s\n", indent, line)
		}
	}
}

func (c *converter) printDecl(d Decl, ns *NamespaceDecl) {
	c.mark(d.declPos())
	switch d := d.(type) {
	case *LetDecl:
		c.printLet(d, ns)
	case *TypeDecl:
		c.printType(d)
	case *ClassDecl:
		c.printClass(d)
	case *InstanceDecl:
		c.printInstance(d)
	case *NamespaceDecl:
		for i, inner := range d.Decls {
			if i > 0 {
				c.b.WriteString("\n")
			}
			c.printDecl(inner, d)
		}
	}
}

// directiveFor maps a goml attribute to its //goplus: directive line
// ("" for attributes handled elsewhere). Quoted arguments are unquoted,
// so @[laws "out=lawtest"] carries the `=` spelling .gp uses.
func (c *converter) directiveFor(a Attr) string {
	args := make([]string, len(a.Args))
	for i, arg := range a.Args {
		if strings.HasPrefix(arg, `"`) && strings.HasSuffix(arg, `"`) && len(arg) >= 2 {
			args[i] = arg[1 : len(arg)-1]
		} else {
			args[i] = arg
		}
	}
	switch a.Name {
	case "box":
		return "//goplus:box " + strings.Join(args, " ")
	case "repr":
		return "//goplus:repr " + strings.Join(args, " ")
	case "derive":
		return "//goplus:derive " + strings.Join(args, " ")
	case "laws":
		return "//goplus:laws " + strings.Join(args, " ")
	case "tail":
		return ""
	}
	c.failf(a.Pos, "unknown attribute @[%s]", a.Name)
	return ""
}

func hasAttr(attrs []Attr, name string) bool {
	for _, a := range attrs {
		if a.Name == name {
			return true
		}
	}
	return false
}

// ----------------------------------------------------------------- types

// typeString renders a goml type in .gp/Go spelling.
func (c *converter) typeString(t Type) string {
	switch t := t.(type) {
	case *TypeName:
		if t.Pkg != "" {
			return t.Pkg + "." + t.Name
		}
		if g, ok := gomlBuiltins[t.Name]; ok {
			return g
		}
		if t.Name == "Unit" {
			c.failf(t.Pos, "Unit is only a function result in goml v0")
		}
		if pkg, ok := c.opens[t.Name]; ok && !c.localNames[t.Name] {
			return pkg + "." + t.Name
		}
		return t.Name
	case *TypeApp:
		head := t.Head
		if head.Pkg == "" {
			switch head.Name {
			case "Slice":
				c.wantArgs(t, 1)
				return "[]" + c.typeString(t.Args[0])
			case "Ptr":
				c.wantArgs(t, 1)
				return "*" + c.typeString(t.Args[0])
			case "Map":
				c.wantArgs(t, 2)
				return "map[" + c.typeString(t.Args[0]) + "]" + c.typeString(t.Args[1])
			case "Chan":
				c.wantArgs(t, 1)
				return "chan " + c.typeString(t.Args[0])
			case "Array":
				c.wantArgs(t, 2)
				return "[" + c.indexString(t.Args[0]) + "]" + c.typeString(t.Args[1])
			}
		}
		var sorts []string
		if head.Pkg == "" {
			if info, ok := c.enums[head.Name]; ok {
				sorts = info.sorts
			}
		}
		parts := make([]string, len(t.Args))
		for i, a := range t.Args {
			if sorts != nil && i < len(sorts) && sorts[i] != "any" {
				parts[i] = c.indexString(a)
			} else {
				parts[i] = c.typeArgString(a)
			}
		}
		return c.typeString(head) + "[" + strings.Join(parts, ", ") + "]"
	case *TypeArrow:
		return "func(" + c.typeString(t.From) + ") " + c.typeString(t.To)
	case *TypeNat:
		return t.Lit
	case *TypeIndexOp:
		return c.indexString(t)
	}
	c.failf(t.typePos(), "unsupported type form")
	return ""
}

func (c *converter) wantArgs(t *TypeApp, n int) {
	if len(t.Args) != n {
		c.failf(t.Pos, "%s takes %d argument(s)", t.Head.Name, n)
	}
}

// typeArgString renders one type-application argument (type or index).
func (c *converter) typeArgString(t Type) string {
	switch t.(type) {
	case *TypeNat, *TypeIndexOp:
		return c.indexString(t)
	}
	return c.typeString(t)
}

// indexString renders an index term without spaces: n+1, Circle(n),
// Plus(a, b).
func (c *converter) indexString(t Type) string {
	switch t := t.(type) {
	case *TypeNat:
		return t.Lit
	case *TypeName:
		if t.Pkg != "" {
			return t.Pkg + "." + t.Name
		}
		return t.Name
	case *TypeIndexOp:
		return c.indexString(t.L) + t.Op + c.indexString(t.R)
	case *TypeApp:
		// Domain-constructor or total-call index: Circle(n), Plus(a, b).
		parts := make([]string, len(t.Args))
		for i, a := range t.Args {
			parts[i] = c.indexString(a)
		}
		return c.indexString(t.Head) + "(" + strings.Join(parts, ", ") + ")"
	}
	c.failf(t.typePos(), "unsupported index term")
	return ""
}

// resultString renders a function result ("" for Unit / omitted).
func (c *converter) resultString(t Type) string {
	if t == nil {
		return ""
	}
	if n, ok := t.(*TypeName); ok && n.Pkg == "" && n.Name == "Unit" {
		return ""
	}
	return c.typeString(t)
}

// ------------------------------------------------------------ type decls

func (c *converter) printType(d *TypeDecl) {
	c.printDoc(d.Doc, "")
	for _, a := range d.Attrs {
		if line := c.directiveFor(a); line != "" {
			c.b.WriteString(line + "\n")
		}
	}
	for _, dv := range d.Deriving {
		switch dv {
		case "Gen":
			c.b.WriteString("//goplus:derive gen\n")
		case "Eq", "Le", "Lt", "Universe", "Transform", "Fold":
			// Derived by default; the clause documents intent.
		default:
			c.failf(d.Pos, "unknown deriving name %q", dv)
		}
	}
	switch {
	case len(d.Sum) > 0:
		c.printEnum(d)
	case len(d.Record) > 0:
		c.printRecord(d)
	case d.Refine != nil:
		fmt.Fprintf(c.b, "type %s refine(%s %s) { %s }\n",
			d.Name, d.Refine.Binder, c.typeString(d.Refine.Base), c.exprString(d.Refine.Pred, 0))
	case d.Prop != nil:
		fmt.Fprintf(c.b, "type %s%s prop { %s }\n", d.Name, c.binderTParams(d), c.fullTypeString(d.Prop))
	case d.Iface != nil:
		c.printIface(d)
	case d.Alias != nil:
		fmt.Fprintf(c.b, "type %s = %s\n", d.Name, c.typeString(d.Alias))
	default:
		c.failf(d.Pos, "empty type declaration")
	}
}

func (c *converter) enumTParams(d *TypeDecl) string {
	info := c.enums[d.Name]
	if len(info.params) == 0 {
		return ""
	}
	parts := make([]string, len(info.params))
	for i := range info.params {
		parts[i] = info.params[i] + " " + info.sorts[i]
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (c *converter) printEnum(d *TypeDecl) {
	fmt.Fprintf(c.b, "type %s%s enum {\n", d.Name, c.enumTParams(d))
	for _, ct := range d.Sum {
		c.printDoc(ct.Doc, "\t")
		var exist string
		if len(ct.Exist) > 0 {
			var parts []string
			for _, b := range ct.Exist {
				bound := c.typeString(b.Type)
				for _, n := range b.Names {
					parts = append(parts, n+" "+bound)
				}
			}
			exist = "[" + strings.Join(parts, ", ") + "]"
		}
		fields := make([]string, len(ct.Fields))
		for i, f := range ct.Fields {
			fields[i] = f.Name + " " + c.typeString(f.Type)
		}
		line := "\t" + ct.Name + exist
		if len(ct.Fields) > 0 || ct.Result != nil {
			line += "(" + strings.Join(fields, ", ") + ")"
		}
		if ct.Result != nil {
			line += " " + c.typeString(ct.Result)
		}
		c.b.WriteString(line + "\n")
	}
	c.b.WriteString("}\n")
}

// indexLetSig records a let's value-parameter types for expected-type
// threading at its call sites.
func (c *converter) indexLetSig(ld *LetDecl) {
	var sig []Type
	for _, b := range ld.Binders {
		if b.Implicit || b.Instance {
			continue // not passed at goml call sites
		}
		for range b.Names {
			sig = append(sig, b.Type)
		}
	}
	if len(sig) > 0 {
		c.letSigs[ld.Name] = sig
	}
}

// binderTParams renders a declaration's binders as a Go type-parameter
// list ("" when there are none).
func (c *converter) binderTParams(d *TypeDecl) string {
	if len(d.Binders) == 0 {
		return ""
	}
	var parts []string
	for _, b := range d.Binders {
		sort := c.binderSort(b)
		for _, n := range b.Names {
			parts = append(parts, n+" "+sort)
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (c *converter) printRecord(d *TypeDecl) {
	fmt.Fprintf(c.b, "type %s%s struct {\n", d.Name, c.binderTParams(d))
	for _, f := range d.Record {
		line := "\t" + f.Name + " " + c.typeString(f.Type)
		var tags []string
		for _, a := range f.Attrs {
			switch {
			case a.Name == "delegate" && len(a.Args) == 0:
				line += " delegate"
			case len(a.Args) == 1 && strings.HasPrefix(a.Args[0], `"`):
				tags = append(tags, a.Name+":"+a.Args[0])
			default:
				c.failf(a.Pos, "unsupported field attribute @[%s]", a.Name)
			}
		}
		if len(tags) > 0 {
			line += " `" + strings.Join(tags, " ") + "`"
		}
		c.b.WriteString(line + "\n")
	}
	c.b.WriteString("}\n")
}

func (c *converter) printIface(d *TypeDecl) {
	fmt.Fprintf(c.b, "type %s%s interface {\n", d.Name, c.binderTParams(d))
	for _, m := range d.Iface.Members {
		c.printDoc(m.Doc, "\t")
		if m.Name == "" {
			fmt.Fprintf(c.b, "\t%s\n", c.typeString(m.Sig))
			continue
		}
		fmt.Fprintf(c.b, "\t%s%s\n", m.Name, c.methodSigString(m))
	}
	c.b.WriteString("}\n")
}

// methodSigString flattens a curried arrow type into a Go method
// signature: `A -> B -> R` becomes `(A, B) R`, a lone `Unit` parameter
// becomes `()`, and a `Unit` result is dropped.
func (c *converter) methodSigString(m *IfaceMember) string {
	arrow, ok := m.Sig.(*TypeArrow)
	if !ok {
		c.failf(m.Pos, "interface member %s needs a function type (`A -> B`)", m.Name)
	}
	var params []string
	var t Type = arrow
	for {
		a, ok := t.(*TypeArrow)
		if !ok {
			break
		}
		if n, isName := a.From.(*TypeName); isName && n.Pkg == "" && n.Name == "Unit" {
			if _, more := a.To.(*TypeArrow); more || len(params) > 0 {
				c.failf(m.Pos, "a Unit parameter must stand alone in interface member %s", m.Name)
			}
		} else {
			params = append(params, c.typeString(a.From))
		}
		t = a.To
	}
	sig := "(" + strings.Join(params, ", ") + ")"
	if res := c.resultString(t); res != "" {
		sig += " " + res
	}
	return sig
}

// --------------------------------------------------------------- classes

func (c *converter) classHeadString(t Type) string {
	app, ok := t.(*TypeApp)
	if !ok {
		c.failf(t.typePos(), "expected an applied class head")
	}
	parts := make([]string, len(app.Args))
	for i, a := range app.Args {
		parts[i] = c.typeString(a)
	}
	return c.typeString(app.Head) + "[" + strings.Join(parts, ", ") + "]"
}

// opParams renders class-op/law/member parameters from binder groups.
func (c *converter) opParams(binders []*Binder) string {
	var parts []string
	for _, b := range binders {
		if b.Implicit || b.Instance || b.Quantity != "" {
			c.failf(b.Pos, "class members take plain typed binders")
		}
		parts = append(parts, strings.Join(b.Names, ", ")+" "+c.typeString(b.Type))
	}
	return strings.Join(parts, ", ")
}

// sigParams renders synthesized parameters for a signature-form op.
func (c *converter) sigParams(sig Type) (string, string) {
	var froms []Type
	t := sig
	for {
		arrow, ok := t.(*TypeArrow)
		if !ok {
			break
		}
		froms = append(froms, arrow.From)
		t = arrow.To
	}
	names := "abcdefgh"
	parts := make([]string, len(froms))
	for i, f := range froms {
		parts[i] = fmt.Sprintf("%c %s", names[i], c.typeString(f))
	}
	return strings.Join(parts, ", "), c.resultString(t)
}

func (c *converter) printClass(d *ClassDecl) {
	c.printDoc(d.Doc, "")
	for _, a := range d.Attrs {
		if line := c.directiveFor(a); line != "" {
			c.b.WriteString(line + "\n")
		}
	}
	binder := d.Binder.Names[0]
	fmt.Fprintf(c.b, "type %s[%s %s] class {\n", d.Name, binder, c.binderSort(d.Binder))
	for _, e := range d.Extends {
		fmt.Fprintf(c.b, "\t%s\n", c.classHeadString(e))
	}
	for _, op := range d.Ops {
		c.printDoc(op.Doc, "\t")
		if op.Sig != nil {
			params, res := c.sigParams(op.Sig)
			line := "\t" + op.Name + "(" + params + ")"
			if res != "" {
				line += " " + res
			}
			c.b.WriteString(line + "\n")
			continue
		}
		line := "\t" + op.Name + "(" + c.opParams(op.Binders) + ")"
		if res := c.resultString(op.Result); res != "" {
			line += " " + res
		}
		if op.Default != nil {
			line += " { return " + c.exprString(op.Default, 0) + " }"
		}
		c.b.WriteString(line + "\n")
	}
	for _, law := range d.Laws {
		c.printDoc(law.Doc, "\t")
		fmt.Fprintf(c.b, "\tlaw %s(%s) { return %s }\n",
			law.Name, c.opParams(law.Binders), c.exprString(law.Body, 0))
	}
	c.b.WriteString("}\n")
}

func (c *converter) printInstance(d *InstanceDecl) {
	c.printDoc(d.Doc, "")
	for _, a := range d.Attrs {
		if line := c.directiveFor(a); line != "" {
			c.b.WriteString(line + "\n")
		}
	}
	tparams := ""
	if len(d.Binders) > 0 {
		var parts []string
		for _, b := range d.Binders {
			sort := c.binderSort(b)
			for _, n := range b.Names {
				parts = append(parts, n+" "+sort)
			}
		}
		tparams = "[" + strings.Join(parts, ", ") + "]"
	}
	fmt.Fprintf(c.b, "instance %s%s %s {\n", d.Name, tparams, c.classHeadString(d.Head))
	for _, m := range d.Members {
		c.mark(m.Pos)
		params, result := c.memberSignature(d, m)
		line := "\t" + m.Name + "(" + params + ")"
		if result != "" {
			line += " " + result
		}
		c.b.WriteString(line + " {\n")
		fw := &fnWriter{c: c, returns: result != ""}
		c.fw = fw
		fw.writeTail(m.Body, "\t\t")
		c.fw = nil
		c.b.WriteString("\t}\n")
	}
	c.b.WriteString("}\n")
}

// memberSignature renders one instance member's parameters and result:
// typed binders as written, or bare names typed by the local class's
// operation signature with the head type substituted for the class var.
func (c *converter) memberSignature(d *InstanceDecl, m *InstMember) (string, string) {
	if len(m.Binders) > 0 || (len(m.Bare) == 0 && m.Result != nil) {
		return c.opParams(m.Binders), c.resultString(m.Result)
	}
	app, ok := d.Head.(*TypeApp)
	if !ok || app.Head.Pkg != "" || len(app.Args) != 1 {
		c.failf(m.Pos, "bare instance members need a locally declared class head")
	}
	op, binder, found := c.resolveOp(app.Head.Name, m.Name)
	if !found {
		c.failf(m.Pos, "cannot infer types for %s: class %s is not declared in this file; use typed binders (a b : T)", m.Name, app.Head.Name)
	}
	if len(op.params) != len(m.Bare) {
		c.failf(m.Pos, "%s takes %d parameter(s), %d given", m.Name, len(op.params), len(m.Bare))
	}
	// Group consecutive parameters sharing a rendered type: a, b int.
	var parts []string
	var group []string
	prev := ""
	flush := func() {
		if len(group) > 0 {
			parts = append(parts, strings.Join(group, ", ")+" "+prev)
			group = nil
		}
	}
	for i, name := range m.Bare {
		ts := c.typeString(substType(op.params[i], binder, app.Args[0]))
		if ts != prev {
			flush()
			prev = ts
		}
		group = append(group, name)
	}
	flush()
	result := m.Result
	if result == nil {
		result = substType(op.result, binder, app.Args[0])
	}
	return strings.Join(parts, ", "), c.resultString(result)
}

// resolveOp finds an operation through a local class's extends chain.
func (c *converter) resolveOp(class, op string) (opSig, string, bool) {
	seen := map[string]bool{}
	var walk func(name string) (opSig, string, bool)
	walk = func(name string) (opSig, string, bool) {
		if seen[name] {
			return opSig{}, "", false
		}
		seen[name] = true
		info, ok := c.classes[name]
		if !ok {
			return opSig{}, "", false
		}
		if sig, ok := info.ops[op]; ok {
			return sig, info.binder, true
		}
		for _, parent := range info.extends {
			if sig, binder, ok := walk(parent); ok {
				// The parent's binder name may differ; normalize to it.
				return sig, binder, ok
			}
		}
		return opSig{}, "", false
	}
	return walk(class)
}

// substType replaces the type variable named v with repl.
func substType(t Type, v string, repl Type) Type {
	switch t := t.(type) {
	case nil:
		return nil
	case *TypeName:
		if t.Pkg == "" && t.Name == v {
			return repl
		}
		return t
	case *TypeApp:
		args := make([]Type, len(t.Args))
		for i, a := range t.Args {
			args[i] = substType(a, v, repl)
		}
		return &TypeApp{Head: t.Head, Args: args, Pos: t.Pos}
	case *TypeArrow:
		return &TypeArrow{From: substType(t.From, v, repl), To: substType(t.To, v, repl), Pos: t.Pos}
	default:
		return t
	}
}

// captured renders f's output into a string instead of the main builder.
func (c *converter) captured(f func()) string {
	old := c.b
	c.b = &strings.Builder{}
	f()
	s := c.b.String()
	c.b = old
	return s
}
