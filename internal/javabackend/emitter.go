package javabackend

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type emitter struct {
	pkg         *packages.Package
	mapper      packageMapper
	diags       []Diagnostic
	temp        int
	initNames   map[*ast.FuncDecl]string
	typeDecls   map[string]*ast.TypeSpec
	methodDecls map[string][]*ast.FuncDecl
	currentSig  *types.Signature
	lambdaVoid  bool
	renames     map[types.Object]string
	javaTypes   map[string]string
}

func newEmitter(pkg *packages.Package, mapper packageMapper) *emitter {
	return &emitter{
		pkg: pkg, mapper: mapper,
		initNames:   make(map[*ast.FuncDecl]string),
		typeDecls:   make(map[string]*ast.TypeSpec),
		methodDecls: make(map[string][]*ast.FuncDecl),
		renames:     make(map[types.Object]string),
		javaTypes:   javaTypeMarkers(pkg),
	}
}

func (e *emitter) emitPackage(app, includeTests bool) (map[string][]byte, []string, []Diagnostic) {
	var decls []ast.Decl
	var tests []string
	initIndex := 0
	for _, file := range e.pkg.Syntax {
		for _, decl := range file.Decls {
			decls = append(decls, decl)
			switch value := decl.(type) {
			case *ast.GenDecl:
				if value.Tok == token.TYPE {
					for _, spec := range value.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							e.typeDecls[ts.Name.Name] = ts
						}
					}
				}
			case *ast.FuncDecl:
				if includeTests && isTestFunction(e.pkg.TypesInfo.Defs[value.Name], value.Name.Name) {
					tests = append(tests, value.Name.Name)
				}
				if value.Name.Name == "init" && value.Recv == nil {
					initIndex++
					e.initNames[value] = fmt.Sprintf("__init%d", initIndex)
				}
				if value.Recv != nil {
					if recv := receiverName(value.Recv); recv != "" {
						e.methodDecls[recv] = append(e.methodDecls[recv], value)
					}
				}
			}
		}
	}
	sort.SliceStable(decls, func(i, j int) bool { return decls[i].Pos() < decls[j].Pos() })

	files := map[string][]byte{}
	var names []string
	for name := range e.typeDecls {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ts := e.typeDecls[name]
		obj, _ := e.pkg.TypesInfo.Defs[ts.Name].(*types.TypeName)
		if obj == nil || ts.Assign.IsValid() {
			continue
		}
		named, _ := types.Unalias(obj.Type()).(*types.Named)
		if named == nil {
			continue
		}
		if _, javaType := e.javaTypes[name]; javaType {
			continue
		}
		var data []byte
		switch named.Underlying().(type) {
		case *types.Struct:
			data = e.emitStruct(ts, named)
		case *types.Interface:
			data = e.emitInterface(ts, named)
		default:
			// Named primitives and collection handles use their underlying JVM
			// representation. Their Go distinction remains in compiler metadata.
			continue
		}
		if len(data) > 0 {
			files[javaIdent(name, ast.IsExported(name))+".java"] = data
		}
	}
	files["GpPackage.java"] = e.emitFacade(decls, app)
	if len(tests) > 0 {
		sort.Strings(tests)
		files["GpTests.java"] = e.emitTestLauncher(tests)
	}
	if len(e.diags) > 0 {
		return nil, nil, e.diags
	}
	return files, tests, nil
}

func javaTypeMarkers(pkg *packages.Package) map[string]string {
	out := map[string]string{}
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				fields := strings.Fields(strings.TrimSpace(comment.Text))
				if len(fields) == 3 && fields[0] == "//goplus:java-type" {
					out[fields[1]] = fields[2]
				}
			}
		}
	}
	return out
}

func (e *emitter) emitTestLauncher(tests []string) []byte {
	w := newJavaWriter()
	e.fileHeader(w)
	w.line("public final class GpTests {")
	w.indent++
	w.line("private GpTests() {}")
	w.line("public static void main(String[] args) {")
	w.indent++
	w.line("int failures = 0;")
	for _, test := range tests {
		w.line("failures += GpTest.run(%s, value -> GpPackage.%s(value));", javaString(test), javaIdent(test, true))
	}
	w.line("if (failures != 0) System.exit(1);")
	w.indent--
	w.line("}")
	w.indent--
	w.line("}")
	return w.bytes()
}

func isTestFunction(obj types.Object, name string) bool {
	if !strings.HasPrefix(name, "Test") || len(name) == len("Test") {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() != 0 || sig.Params().Len() != 1 {
		return false
	}
	t := types.Unalias(sig.Params().At(0).Type())
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "testing" && named.Obj().Name() == "T"
}

func (e *emitter) emitFacade(decls []ast.Decl, app bool) []byte {
	w := newJavaWriter()
	e.fileHeader(w)
	w.line("@SuppressWarnings({\"unchecked\", \"cast\"})")
	w.line("public final class GpPackage {")
	w.indent++
	w.line("private GpPackage() {}")

	var initCalls []string
	for _, decl := range decls {
		switch value := decl.(type) {
		case *ast.GenDecl:
			switch value.Tok {
			case token.CONST, token.VAR:
				e.emitGlobals(w, value)
			}
		case *ast.FuncDecl:
			if value.Recv != nil {
				continue
			}
			name := e.functionName(value)
			if value.Name.Name == "init" {
				initCalls = append(initCalls, name)
			}
			e.emitFunction(w, value, true, name)
		}
	}
	if len(initCalls) > 0 {
		w.line("static {")
		w.indent++
		for _, name := range initCalls {
			w.line("%s();", name)
		}
		w.indent--
		w.line("}")
	}
	if app {
		mainFound := false
		for _, decl := range decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "main" {
				mainFound = true
				break
			}
		}
		if !mainFound {
			e.unsupported(nil, "Java app target requires func main() in package %s", e.pkg.PkgPath)
		} else {
			w.line("public static void main(String[] args) {")
			w.indent++
			w.line("main();")
			w.indent--
			w.line("}")
		}
	}
	w.indent--
	w.line("}")
	return w.bytes()
}

func (e *emitter) emitGlobals(w *javaWriter, decl *ast.GenDecl) {
	for _, raw := range decl.Specs {
		spec, ok := raw.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, ident := range spec.Names {
			obj := e.pkg.TypesInfo.Defs[ident]
			if obj == nil || ident.Name == "_" {
				continue
			}
			visibility := ""
			if ast.IsExported(ident.Name) {
				visibility = "public "
			}
			modifier := "static "
			if decl.Tok == token.CONST {
				modifier += "final "
			}
			init := e.zeroValue(obj.Type())
			if i < len(spec.Values) {
				init = e.expr(spec.Values[i])
			} else if len(spec.Values) == 1 {
				init = e.expr(spec.Values[0])
			}
			w.line("%s%s%s %s = %s;", visibility, modifier, e.javaType(obj.Type(), false), e.objectName(obj), e.copyValue(init, obj.Type()))
		}
	}
}

func (e *emitter) emitStruct(ts *ast.TypeSpec, named *types.Named) []byte {
	w := newJavaWriter()
	e.fileHeader(w)
	name := javaIdent(ts.Name.Name, ast.IsExported(ts.Name.Name))
	visibility := ""
	if ast.IsExported(ts.Name.Name) {
		visibility = "public "
	}
	tparams := e.typeParams(named.TypeParams())
	implements := []string{"GpCopy<" + name + typeArgs(named.TypeParams()) + ">"}
	implements = append(implements, e.implementedInterfaces(named)...)
	w.line("%sfinal class %s%s implements %s {", visibility, name, tparams, strings.Join(implements, ", "))
	w.indent++
	strct := named.Underlying().(*types.Struct)
	for i := 0; i < strct.NumFields(); i++ {
		field := strct.Field(i)
		visibility := ""
		if field.Exported() {
			visibility = "public "
		}
		w.line("%s%s %s;", visibility, e.javaType(field.Type(), false), javaIdent(field.Name(), field.Exported()))
	}
	w.line("%s%s() {", visibility, name)
	w.indent++
	for i := 0; i < strct.NumFields(); i++ {
		field := strct.Field(i)
		w.line("this.%s = %s;", javaIdent(field.Name(), field.Exported()), e.zeroValue(field.Type()))
	}
	w.indent--
	w.line("}")
	if strct.NumFields() > 0 {
		var params []string
		for i := 0; i < strct.NumFields(); i++ {
			field := strct.Field(i)
			params = append(params, e.javaType(field.Type(), false)+" "+javaIdent(field.Name(), field.Exported()))
		}
		w.line("%s(%s) {", visibility+name, strings.Join(params, ", "))
		w.indent++
		for i := 0; i < strct.NumFields(); i++ {
			field := strct.Field(i)
			fieldName := javaIdent(field.Name(), field.Exported())
			w.line("this.%s = %s;", fieldName, e.copyValue(fieldName, field.Type()))
		}
		w.indent--
		w.line("}")
	}
	w.line("@Override public %s%s copy() {", name, typeArgs(named.TypeParams()))
	w.indent++
	if strct.NumFields() == 0 {
		w.line("return new %s%s();", name, diamond(named.TypeParams()))
	} else {
		var fields []string
		for i := 0; i < strct.NumFields(); i++ {
			field := strct.Field(i)
			fields = append(fields, e.copyValue("this."+javaIdent(field.Name(), field.Exported()), field.Type()))
		}
		w.line("return new %s%s(%s);", name, diamond(named.TypeParams()), strings.Join(fields, ", "))
	}
	w.indent--
	w.line("}")
	w.line("@Override public boolean equals(Object other) {")
	w.indent++
	instanceType := name
	if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
		wildcards := make([]string, named.TypeParams().Len())
		for i := range wildcards {
			wildcards[i] = "?"
		}
		instanceType += "<" + strings.Join(wildcards, ", ") + ">"
	}
	w.line("if (!(other instanceof %s value)) return false;", instanceType)
	if strct.NumFields() == 0 {
		w.line("return true;")
	} else {
		var comparisons []string
		for i := 0; i < strct.NumFields(); i++ {
			field := strct.Field(i)
			fieldName := javaIdent(field.Name(), field.Exported())
			comparisons = append(comparisons, e.fieldEquality("this."+fieldName, "value."+fieldName, field.Type()))
		}
		w.line("return %s;", strings.Join(comparisons, " && "))
	}
	w.indent--
	w.line("}")
	w.line("@Override public int hashCode() {")
	w.indent++
	w.line("int hash = 1;")
	for i := 0; i < strct.NumFields(); i++ {
		field := strct.Field(i)
		fieldName := "this." + javaIdent(field.Name(), field.Exported())
		w.line("hash = 31 * hash + %s;", e.hashExpression(fieldName, field.Type()))
	}
	w.line("return hash;")
	w.indent--
	w.line("}")
	for _, method := range e.methodDecls[ts.Name.Name] {
		e.emitFunction(w, method, false, javaIdent(method.Name.Name, ast.IsExported(method.Name.Name)))
	}
	w.indent--
	w.line("}")
	return w.bytes()
}

func (e *emitter) hashExpression(value string, t types.Type) string {
	if _, pointer := types.Unalias(t).(*types.Pointer); pointer {
		return "System.identityHashCode(" + value + ")"
	}
	return "GpRuntime.hash(" + value + ")"
}

func (e *emitter) fieldEquality(left, right string, t types.Type) string {
	base := types.Unalias(t)
	if _, pointer := base.(*types.Pointer); pointer {
		return left + " == " + right
	}
	if named, ok := base.(*types.Named); ok {
		base = named.Underlying()
	}
	if basic, ok := base.(*types.Basic); ok {
		if basic.Info()&(types.IsBoolean|types.IsNumeric) != 0 {
			return left + " == " + right
		}
		if basic.Info()&types.IsString != 0 {
			return left + ".equals(" + right + ")"
		}
	}
	return "GpRuntime.equal(" + left + ", " + right + ")"
}

func (e *emitter) emitInterface(ts *ast.TypeSpec, named *types.Named) []byte {
	w := newJavaWriter()
	e.fileHeader(w)
	name := javaIdent(ts.Name.Name, ast.IsExported(ts.Name.Name))
	visibility := ""
	if ast.IsExported(ts.Name.Name) {
		visibility = "public "
	}
	iface := named.Underlying().(*types.Interface).Complete()
	w.line("%sinterface %s%s {", visibility, name, e.typeParams(named.TypeParams()))
	w.indent++
	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		sig := method.Type().(*types.Signature)
		w.line("%s;", e.signature(method.Name(), sig, false, method.Exported()))
	}
	w.indent--
	w.line("}")
	return w.bytes()
}

func (e *emitter) implementedInterfaces(named *types.Named) []string {
	var out []string
	errorInterface, _ := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	if errorInterface != nil && (types.Implements(named, errorInterface) || types.Implements(types.NewPointer(named), errorInterface)) {
		out = append(out, "GpError")
	}
	for name, ts := range e.typeDecls {
		obj, _ := e.pkg.TypesInfo.Defs[ts.Name].(*types.TypeName)
		if obj == nil {
			continue
		}
		other, _ := types.Unalias(obj.Type()).(*types.Named)
		if other == nil {
			continue
		}
		iface, ok := other.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		iface.Complete()
		if types.Implements(named, iface) || types.Implements(types.NewPointer(named), iface) || methodSetCovers(named, iface) {
			out = append(out, javaIdent(name, ast.IsExported(name))+typeArgs(other.TypeParams()))
		}
	}
	sort.Strings(out)
	return compactStrings(out)
}

func methodSetCovers(named *types.Named, iface *types.Interface) bool {
	if iface.NumExplicitMethods() == 0 {
		return false
	}
	methods := types.NewMethodSet(named)
	for i := 0; i < iface.NumExplicitMethods(); i++ {
		want := iface.ExplicitMethod(i)
		// This fallback exists for Go+'s sealed, package-private marker methods
		// after dependent indexes are erased. Never use name-only matching to
		// claim an ordinary public behavioral interface.
		if want.Exported() || !strings.HasPrefix(want.Name(), "is") {
			return false
		}
		selection := methods.Lookup(want.Pkg(), want.Name())
		if selection == nil {
			return false
		}
		// Dependent indexes intentionally disappear before Java generation, so
		// the ordinary-Go marker signatures may differ only in erased phantom
		// parameters. Package identity plus the unexported is* name seals them.
	}
	return true
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func (e *emitter) emitFunction(w *javaWriter, decl *ast.FuncDecl, static bool, name string) {
	obj, _ := e.pkg.TypesInfo.Defs[decl.Name].(*types.Func)
	if obj == nil {
		// init declarations do not enter package scope, but go/types still
		// records their signature on the FuncDecl name in normal packages.
		e.unsupported(decl, "cannot recover type information for function %s", decl.Name.Name)
		return
	}
	sig, _ := obj.Type().(*types.Signature)
	if sig == nil {
		return
	}
	binding, bindingErr := parseJavaBinding(decl)
	if bindingErr != nil {
		e.unsupported(decl, "invalid //goplus:java directive: %v", bindingErr)
		return
	}
	if binding != nil {
		if !static {
			e.unsupported(decl, "//goplus:java bindings must be package functions")
			return
		}
		e.emitJavaBinding(w, decl, sig, name, binding)
		return
	}
	if decl.Body == nil {
		e.unsupported(decl, "bodyless function %s needs a //goplus:java binding", decl.Name.Name)
		return
	}
	previousSig := e.currentSig
	e.currentSig = sig
	defer func() { e.currentSig = previousSig }()
	visibility := ast.IsExported(decl.Name.Name) || !static
	header := e.signature(name, sig, static, visibility)
	w.line("%s {", header)
	w.indent++
	if !static && decl.Recv != nil && len(decl.Recv.List) > 0 && len(decl.Recv.List[0].Names) > 0 {
		receiverIdent := decl.Recv.List[0].Names[0]
		if receiverIdent.Name != "_" {
			if receiver := e.pkg.TypesInfo.Defs[receiverIdent]; receiver != nil {
				initial := "this"
				if _, pointer := types.Unalias(sig.Recv().Type()).(*types.Pointer); !pointer {
					initial = e.copyValue("this", receiver.Type())
				}
				w.line("%s %s = %s;", e.javaType(receiver.Type(), false), e.objectName(receiver), initial)
			}
		}
	}
	// Named results are ordinary mutable locals on the JVM; naked returns
	// below read them in declaration order.
	for i := 0; i < sig.Results().Len(); i++ {
		result := sig.Results().At(i)
		if result.Name() != "" {
			w.line("%s %s = %s;", e.javaType(result.Type(), false), e.objectName(result), e.zeroValue(result.Type()))
		}
	}
	if containsDefer(decl.Body) {
		w.line("try (var __defer = GpRuntime.deferScope()) {")
		w.indent++
		w.line("__defer.touch();")
		e.block(w, decl.Body)
		w.indent--
		w.line("}")
	} else {
		e.block(w, decl.Body)
	}
	w.indent--
	w.line("}")
}

func (e *emitter) signature(name string, sig *types.Signature, static, public bool) string {
	var b strings.Builder
	if public {
		b.WriteString("public ")
	}
	if static {
		b.WriteString("static ")
	}
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		b.WriteString(e.typeParams(sig.TypeParams()))
		b.WriteByte(' ')
	}
	result := "void"
	if sig.Results().Len() == 1 {
		result = e.javaType(sig.Results().At(0).Type(), false)
	} else if sig.Results().Len() > 1 {
		result = e.tupleType(sig.Results())
	}
	b.WriteString(result)
	b.WriteByte(' ')
	b.WriteString(javaIdent(name, ast.IsExported(name)))
	b.WriteByte('(')
	var params []string
	for i := 0; i < sig.Params().Len(); i++ {
		param := sig.Params().At(i)
		name := param.Name()
		if name == "" || name == "_" {
			name = fmt.Sprintf("arg%d", i)
		}
		typeText := e.javaType(param.Type(), false)
		params = append(params, typeText+" "+javaIdent(name, false))
	}
	b.WriteString(strings.Join(params, ", "))
	b.WriteByte(')')
	return b.String()
}

func (e *emitter) typeParams(params *types.TypeParamList) string {
	if params == nil || params.Len() == 0 {
		return ""
	}
	var names []string
	for i := 0; i < params.Len(); i++ {
		names = append(names, javaIdent(params.At(i).Obj().Name(), true))
	}
	return "<" + strings.Join(names, ", ") + ">"
}

func typeArgs(params *types.TypeParamList) string {
	if params == nil || params.Len() == 0 {
		return ""
	}
	var names []string
	for i := 0; i < params.Len(); i++ {
		names = append(names, javaIdent(params.At(i).Obj().Name(), true))
	}
	return "<" + strings.Join(names, ", ") + ">"
}

func diamond(params *types.TypeParamList) string {
	if params == nil || params.Len() == 0 {
		return ""
	}
	return "<>"
}

func (e *emitter) fileHeader(w *javaWriter) {
	w.line("// Code generated by goplus for Java 25+. DO NOT EDIT.")
	w.line("package %s;", e.mapper.javaPackage(e.pkg.PkgPath))
	w.line("")
	w.line("import dev.goforge.goplus.runtime.*;")
	w.line("")
}

func (e *emitter) functionName(decl *ast.FuncDecl) string {
	if name := e.initNames[decl]; name != "" {
		return name
	}
	return javaIdent(decl.Name.Name, ast.IsExported(decl.Name.Name))
}

func (e *emitter) objectName(obj types.Object) string {
	if renamed := e.renames[obj]; renamed != "" {
		return renamed
	}
	return javaIdent(obj.Name(), obj.Exported())
}

func receiverName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	expr := fields.List[0].Type
	for {
		switch value := expr.(type) {
		case *ast.StarExpr:
			expr = value.X
		case *ast.IndexExpr:
			expr = value.X
		case *ast.IndexListExpr:
			expr = value.X
		default:
			if ident, ok := expr.(*ast.Ident); ok {
				return ident.Name
			}
			return ""
		}
	}
}

func containsDefer(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(node ast.Node) bool {
		if found {
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		if _, ok := node.(*ast.DeferStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func (e *emitter) unsupported(node ast.Node, format string, args ...any) {
	pos := token.Position{}
	if node != nil {
		pos = e.pkg.Fset.Position(node.Pos())
	}
	e.diags = append(e.diags, Diagnostic{Pos: pos, Message: fmt.Sprintf(format, args...)})
}

type javaWriter struct {
	buf    bytes.Buffer
	indent int
}

func newJavaWriter() *javaWriter { return &javaWriter{} }

func (w *javaWriter) line(format string, args ...any) {
	if format != "" {
		w.buf.WriteString(strings.Repeat("    ", w.indent))
		fmt.Fprintf(&w.buf, format, args...)
	}
	w.buf.WriteByte('\n')
}

func (w *javaWriter) bytes() []byte {
	return append([]byte(nil), w.buf.Bytes()...)
}
