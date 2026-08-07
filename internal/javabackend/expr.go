package javabackend

import (
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

func (e *emitter) expr(node ast.Expr) string {
	if node == nil {
		return "null"
	}
	switch value := node.(type) {
	case *ast.BasicLit:
		return e.basicLiteral(value)
	case *ast.Ident:
		return e.ident(value)
	case *ast.ParenExpr:
		return "(" + e.expr(value.X) + ")"
	case *ast.BinaryExpr:
		return e.binary(value)
	case *ast.UnaryExpr:
		return e.unary(value)
	case *ast.CallExpr:
		return e.call(value)
	case *ast.SelectorExpr:
		return e.selector(value)
	case *ast.IndexExpr:
		if e.isInstantiation(value) {
			return e.expr(value.X)
		}
		return e.index(value.X, value.Index)
	case *ast.IndexListExpr:
		return e.expr(value.X)
	case *ast.SliceExpr:
		low := "0"
		if value.Low != nil {
			low = e.expr(value.Low)
		}
		high := "GpRuntime.length(" + e.expr(value.X) + ")"
		if value.High != nil {
			high = e.expr(value.High)
		}
		if value.Max != nil {
			e.unsupported(value, "three-index slices are not yet in the portable Java subset")
		}
		return e.expr(value.X) + ".slice(GpRuntime.index(" + low + "), GpRuntime.index(" + high + "))"
	case *ast.CompositeLit:
		return e.composite(value)
	case *ast.FuncLit:
		return e.functionLiteral(value)
	case *ast.StarExpr:
		if pointer, ok := types.Unalias(e.pkg.TypesInfo.TypeOf(value.X)).(*types.Pointer); ok {
			if named, ok := types.Unalias(pointer.Elem()).(*types.Named); ok {
				if _, ok := named.Underlying().(*types.Struct); ok {
					return e.expr(value.X)
				}
			}
		}
		return e.expr(value.X) + ".get()"
	case *ast.TypeAssertExpr:
		if value.Type == nil {
			e.unsupported(value, "type-switch assertions are not yet in the portable Java subset")
			return e.expr(value.X)
		}
		return "((" + e.javaType(e.pkg.TypesInfo.TypeOf(value), false) + ") " + e.expr(value.X) + ")"
	case *ast.KeyValueExpr:
		return e.expr(value.Value)
	case *ast.Ellipsis:
		e.unsupported(value, "array ellipsis is not yet in the portable Java subset")
		return "0"
	default:
		e.unsupported(node, "Go expression %T is not yet in the portable Java subset", node)
		return "null"
	}
}

func (e *emitter) basicLiteral(lit *ast.BasicLit) string {
	tv := e.pkg.TypesInfo.Types[lit]
	if lit.Kind == token.STRING {
		text, err := strconv.Unquote(lit.Value)
		if err != nil {
			e.unsupported(lit, "invalid Go string literal: %v", err)
			return "GpString.EMPTY"
		}
		return "GpString.fromBase64(" + strconv.Quote(base64.StdEncoding.EncodeToString([]byte(text))) + ")"
	}
	if lit.Kind == token.CHAR {
		text, _, _, err := strconv.UnquoteChar(strings.TrimSuffix(strings.TrimPrefix(lit.Value, "'"), "'"), '\'')
		if err != nil || text == 0 {
			if tv.Value != nil {
				return tv.Value.ExactString()
			}
			return "0"
		}
		return strconv.FormatInt(int64(text), 10)
	}
	if tv.Value == nil {
		return strings.ReplaceAll(lit.Value, "_", "")
	}
	t := e.pkg.TypesInfo.TypeOf(lit)
	if isUnsigned(t) {
		if n, ok := constant.Uint64Val(tv.Value); ok && n > uint64(^uint64(0)>>1) {
			return "Long.parseUnsignedLong(" + strconv.Quote(strconv.FormatUint(n, 10)) + ")"
		}
	}
	if basic, ok := types.Unalias(t).(*types.Basic); ok && basic.Info()&types.IsInteger != 0 {
		text := tv.Value.ExactString()
		if basicJavaType(basic, false) == "long" {
			return text + "L"
		}
		return text
	}
	if _, ok := constant.Float64Val(tv.Value); ok {
		text := tv.Value.String()
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
		return text
	}
	return tv.Value.ExactString()
}

func (e *emitter) ident(ident *ast.Ident) string {
	switch ident.Name {
	case "nil":
		return "null"
	case "true", "false":
		return ident.Name
	case "iota":
		if tv := e.pkg.TypesInfo.Types[ident]; tv.Value != nil {
			return tv.Value.ExactString()
		}
	}
	obj := e.pkg.TypesInfo.Uses[ident]
	if obj == nil {
		obj = e.pkg.TypesInfo.Defs[ident]
	}
	if obj == nil {
		return javaIdent(ident.Name, ast.IsExported(ident.Name))
	}
	name := e.objectName(obj)
	if _, ok := obj.(*types.Func); ok && obj.Pkg() != nil && obj.Pkg().Path() == e.pkg.PkgPath && obj.Parent() == e.pkg.Types.Scope() {
		return "GpPackage::" + name
	}
	if obj.Pkg() != nil && obj.Pkg().Path() == e.pkg.PkgPath && obj.Parent() == e.pkg.Types.Scope() {
		switch obj.(type) {
		case *types.Var, *types.Const, *types.Func:
			return "GpPackage." + name
		}
	}
	return name
}

func (e *emitter) binary(expr *ast.BinaryExpr) string {
	left, right := e.expr(expr.X), e.expr(expr.Y)
	if expr.Op == token.EQL || expr.Op == token.NEQ {
		var check string
		switch {
		case isNilLiteral(expr.X):
			check = e.nilCheck(right, e.pkg.TypesInfo.TypeOf(expr.Y))
		case isNilLiteral(expr.Y):
			check = e.nilCheck(left, e.pkg.TypesInfo.TypeOf(expr.X))
		}
		if check != "" {
			if expr.Op == token.NEQ {
				return "!(" + check + ")"
			}
			return check
		}
	}
	t := e.pkg.TypesInfo.TypeOf(expr.X)
	if isStringType(t) {
		switch expr.Op {
		case token.ADD:
			return left + ".concat(" + right + ")"
		case token.EQL:
			return left + ".equals(" + right + ")"
		case token.NEQ:
			return "!" + left + ".equals(" + right + ")"
		case token.LSS, token.LEQ, token.GTR, token.GEQ:
			op := map[token.Token]string{token.LSS: "<", token.LEQ: "<=", token.GTR: ">", token.GEQ: ">="}[expr.Op]
			return left + ".compareTo(" + right + ") " + op + " 0"
		}
	}
	if isUnsigned(t) {
		switch expr.Op {
		case token.QUO:
			return e.normalizeNumeric("Long.divideUnsigned("+left+", "+right+")", t)
		case token.REM:
			return e.normalizeNumeric("Long.remainderUnsigned("+left+", "+right+")", t)
		case token.SHR:
			return e.normalizeNumeric("GpRuntime.shiftRightUnsigned("+left+", "+right+")", t)
		case token.LSS, token.LEQ, token.GTR, token.GEQ:
			op := map[token.Token]string{token.LSS: "<", token.LEQ: "<=", token.GTR: ">", token.GEQ: ">="}[expr.Op]
			return "Long.compareUnsigned(" + left + ", " + right + ") " + op + " 0"
		}
	}
	switch expr.Op {
	case token.SHL:
		return e.normalizeNumeric("GpRuntime.shiftLeft("+left+", "+right+")", t)
	case token.SHR:
		return e.normalizeNumeric("GpRuntime.shiftRight("+left+", "+right+")", t)
	case token.AND_NOT:
		return e.normalizeNumeric("("+left+" & ~("+right+"))", t)
	case token.EQL, token.NEQ:
		if _, pointer := types.Unalias(t).(*types.Pointer); pointer {
			return "(" + left + " " + expr.Op.String() + " " + right + ")"
		}
		if !isBasicType(t) {
			eq := "GpRuntime.equal(" + left + ", " + right + ")"
			if expr.Op == token.NEQ {
				return "!" + eq
			}
			return eq
		}
	}
	raw := "(" + left + " " + expr.Op.String() + " " + right + ")"
	switch expr.Op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
		return raw
	default:
		return e.normalizeNumeric(raw, t)
	}
}

func (e *emitter) unary(expr *ast.UnaryExpr) string {
	value := e.expr(expr.X)
	switch expr.Op {
	case token.ARROW:
		return value + ".receive()"
	case token.XOR:
		return e.normalizeNumeric("~("+value+")", e.pkg.TypesInfo.TypeOf(expr))
	case token.AND:
		if _, ok := expr.X.(*ast.CompositeLit); ok {
			return value
		}
		e.unsupported(expr, "taking the address of a variable needs escape-cell lowering and is not yet portable")
		return "new GpRef<>(" + value + ")"
	case token.ADD:
		return value
	default:
		return e.normalizeNumeric(expr.Op.String()+"("+value+")", e.pkg.TypesInfo.TypeOf(expr))
	}
}

func (e *emitter) call(call *ast.CallExpr) string {
	if constructor, ok := e.javaConstructor(call); ok {
		return constructor
	}
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if obj, ok := e.pkg.TypesInfo.Uses[ident].(*types.Builtin); ok {
			return e.builtinCall(obj.Name(), call)
		}
	}
	if tv, ok := e.pkg.TypesInfo.Types[call.Fun]; ok && tv.IsType() {
		return e.conversion(e.pkg.TypesInfo.TypeOf(call), call.Args)
	}
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if pkgIdent, ok := selector.X.(*ast.Ident); ok {
			if pkgName, ok := e.pkg.TypesInfo.Uses[pkgIdent].(*types.PkgName); ok {
				if adapted, ok := e.stdCall(pkgName.Imported().Path(), selector.Sel.Name, call.Args); ok {
					return adapted
				}
			}
		}
	}
	var args []string
	sig, _ := underlyingType(e.pkg.TypesInfo.TypeOf(call.Fun)).(*types.Signature)
	if sig != nil {
		args = e.callArguments(call, sig)
	} else {
		for _, arg := range call.Args {
			args = append(args, e.expr(arg))
		}
	}
	direct := isDirectCall(e, call.Fun)
	fun := e.expr(call.Fun)
	if direct {
		fun = e.directCallName(call.Fun)
	}
	if sig != nil && !direct {
		return fun + ".apply(" + strings.Join(args, ", ") + ")"
	}
	return fun + "(" + strings.Join(args, ", ") + ")"
}

func (e *emitter) javaConstructor(call *ast.CallExpr) (string, bool) {
	base := call.Fun
	switch value := base.(type) {
	case *ast.IndexExpr:
		base = value.X
	case *ast.IndexListExpr:
		base = value.X
	}
	ident, ok := base.(*ast.Ident)
	if !ok || ident.Name != "__goplusJavaNew" {
		return "", false
	}
	t := types.Unalias(e.pkg.TypesInfo.TypeOf(call))
	named, ok := t.(*types.Named)
	if !ok {
		return "", false
	}
	owner := e.javaTypes[named.Obj().Name()]
	if owner == "" {
		return "", false
	}
	var args []string
	for _, arg := range call.Args {
		text := e.expr(arg)
		if kind, ok := basicKind(e.pkg.TypesInfo.TypeOf(arg)); ok && kind == types.Int {
			text = "GpRuntime.index(" + text + ")"
		}
		args = append(args, text)
	}
	javaType := e.javaType(t, false)
	if named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
		javaType = owner + "<>"
	}
	return "new " + javaType + "(" + strings.Join(args, ", ") + ")", true
}

func (e *emitter) callArguments(call *ast.CallExpr, sig *types.Signature) []string {
	var out []string
	fixed := sig.Params().Len()
	if sig.Variadic() {
		fixed--
	}
	for i := 0; i < len(call.Args) && i < fixed; i++ {
		out = append(out, e.copyValue(e.expr(call.Args[i]), sig.Params().At(i).Type()))
	}
	if !sig.Variadic() {
		for i := len(out); i < len(call.Args); i++ {
			out = append(out, e.copyValue(e.expr(call.Args[i]), sig.Params().At(i).Type()))
		}
		return out
	}
	if call.Ellipsis.IsValid() {
		if len(call.Args) > fixed {
			out = append(out, e.copyValue(e.expr(call.Args[fixed]), sig.Params().At(fixed).Type()))
		}
		return out
	}
	var rest []string
	for i := fixed; i < len(call.Args); i++ {
		rest = append(rest, e.copyValue(e.expr(call.Args[i]), elementType(sig.Params().At(fixed).Type())))
	}
	elemZero := "null"
	if elem := elementType(sig.Params().At(fixed).Type()); elem != types.Typ[types.Invalid] {
		elemZero = e.zeroValue(elem)
	}
	out = append(out, "GpSlice.ofZero(() -> "+elemZero+", "+e.elementCopier(elementType(sig.Params().At(fixed).Type()))+optionalComma(rest)+strings.Join(rest, ", ")+")")
	return out
}

func (e *emitter) builtinCall(name string, call *ast.CallExpr) string {
	args := make([]string, len(call.Args))
	start := 0
	if name == "make" || name == "new" {
		start = 1 // the first argument is a Go type, not a runtime value
	}
	for i, arg := range call.Args[start:] {
		i += start
		args[i] = e.expr(arg)
	}
	switch name {
	case "len":
		return "GpRuntime.length(" + args[0] + ")"
	case "cap":
		return "GpRuntime.capacity(" + args[0] + ")"
	case "append":
		if call.Ellipsis.IsValid() {
			return args[0] + ".appendedAll(" + args[1] + ")"
		}
		out := args[0]
		elem := elementType(e.pkg.TypesInfo.TypeOf(call.Args[0]))
		for _, arg := range args[1:] {
			out += ".appended(" + e.copyValue(arg, elem) + ")"
		}
		return out
	case "make":
		t := types.Unalias(e.pkg.TypesInfo.TypeOf(call))
		switch value := t.(type) {
		case *types.Slice:
			length := "0"
			capacity := length
			if len(args) > 1 {
				length = args[1]
				capacity = length
			}
			if len(args) > 2 {
				capacity = args[2]
			}
			return "GpSlice.make(GpRuntime.index(" + length + "), GpRuntime.index(" + capacity + "), () -> " + e.zeroValue(value.Elem()) + ", " + e.elementCopier(value.Elem()) + ")"
		case *types.Map:
			return "GpMap.make(() -> " + e.zeroValue(value.Elem()) + ")"
		case *types.Chan:
			capacity := "0"
			if len(args) > 1 {
				capacity = "GpRuntime.index(" + args[1] + ")"
			}
			return "GpChan.make(" + capacity + ", () -> " + e.zeroValue(value.Elem()) + ")"
		default:
			e.unsupported(call, "make of %s is not in the portable Java subset", types.TypeString(t, nil))
			return "null"
		}
	case "new":
		ptr, _ := types.Unalias(e.pkg.TypesInfo.TypeOf(call)).(*types.Pointer)
		if ptr == nil {
			return "null"
		}
		if named, ok := types.Unalias(ptr.Elem()).(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok {
				return e.zeroValue(named)
			}
		}
		return "new GpRef<>(" + e.zeroValue(ptr.Elem()) + ")"
	case "delete":
		return args[0] + ".delete(" + args[1] + ")"
	case "close":
		return args[0] + ".close()"
	case "panic":
		return "GpRuntime.panicExpr(" + args[0] + ")"
	case "print":
		for i, raw := range call.Args {
			args[i] = e.printValue(args[i], e.pkg.TypesInfo.TypeOf(raw))
		}
		return "GpRuntime.print(" + strings.Join(args, ", ") + ")"
	case "println":
		for i, raw := range call.Args {
			args[i] = e.printValue(args[i], e.pkg.TypesInfo.TypeOf(raw))
		}
		return "GpRuntime.println(" + strings.Join(args, ", ") + ")"
	case "copy":
		return "GpRuntime.copySlice(" + strings.Join(args, ", ") + ")"
	case "clear":
		return "GpRuntime.clear(" + strings.Join(args, ", ") + ")"
	case "min", "max":
		t := e.pkg.TypesInfo.TypeOf(call)
		method := "Math." + name
		if isStringType(t) {
			method = "GpRuntime." + name + "String"
		} else if isUnsigned(t) {
			method = "GpRuntime." + name + "Unsigned"
		}
		out := args[0]
		for _, arg := range args[1:] {
			out = method + "(" + out + ", " + arg + ")"
		}
		return e.normalizeNumeric(out, t)
	case "recover":
		e.unsupported(call, "recover requires panic-frame lowering and is not yet in the portable Java subset")
		return "null"
	default:
		e.unsupported(call, "builtin %s is not yet in the portable Java subset", name)
		return "null"
	}
}

func (e *emitter) conversion(target types.Type, args []ast.Expr) string {
	if len(args) != 1 {
		e.unsupported(nil, "Java conversion expects one operand")
		return "null"
	}
	arg := e.expr(args[0])
	source := e.pkg.TypesInfo.TypeOf(args[0])
	if basic, ok := types.Unalias(source).(*types.Basic); ok && basic.Kind() == types.UntypedNil {
		return e.zeroValue(target)
	}
	if isStringType(target) {
		if isStringType(source) {
			return arg
		}
		if kind, ok := basicKind(source); ok && types.Typ[kind].Info()&types.IsInteger != 0 {
			return "GpRuntime.stringFromRune(" + arg + ")"
		}
		if slice, ok := underlyingType(source).(*types.Slice); ok {
			if kind, _ := basicKind(slice.Elem()); kind == types.Uint8 {
				return "GpRuntime.stringFromBytes(" + arg + ")"
			} else if kind == types.Int32 {
				return "GpRuntime.stringFromRunes(" + arg + ")"
			}
		}
		e.unsupported(args[0], "conversion from %s to string is not in the portable Java subset", types.TypeString(source, nil))
		return "GpString.EMPTY"
	}
	if slice, ok := underlyingType(target).(*types.Slice); ok && isStringType(source) {
		if kind, _ := basicKind(slice.Elem()); kind == types.Uint8 {
			return "GpRuntime.bytesFromString(" + arg + ")"
		} else if kind == types.Int32 {
			return "GpRuntime.runesFromString(" + arg + ")"
		}
		e.unsupported(args[0], "conversion from string to %s is not in the portable Java subset", types.TypeString(target, nil))
		return e.zeroValue(target)
	}
	if sameBasicKind(target, source) {
		return arg
	}
	if normalized := e.normalizeNumeric(arg, target); normalized != arg {
		return normalized
	}
	return "((" + e.javaType(target, false) + ") (" + arg + "))"
}

func (e *emitter) selector(selector *ast.SelectorExpr) string {
	if selection := e.pkg.TypesInfo.Selections[selector]; selection != nil {
		separator := "."
		if _, ok := selection.Obj().(*types.Func); ok {
			separator = "::"
		}
		return e.expr(selector.X) + separator + javaIdent(selector.Sel.Name, selection.Obj().Exported())
	}
	if ident, ok := selector.X.(*ast.Ident); ok {
		if pkgName, ok := e.pkg.TypesInfo.Uses[ident].(*types.PkgName); ok {
			obj := e.pkg.TypesInfo.Uses[selector.Sel]
			pkg := e.mapper.javaPackage(pkgName.Imported().Path())
			name := javaIdent(selector.Sel.Name, ast.IsExported(selector.Sel.Name))
			switch obj.(type) {
			case *types.TypeName:
				return pkg + "." + name
			case *types.Func:
				return pkg + ".GpPackage::" + name
			default:
				return pkg + ".GpPackage." + name
			}
		}
	}
	return e.expr(selector.X) + "." + javaIdent(selector.Sel.Name, ast.IsExported(selector.Sel.Name))
}

func (e *emitter) directCallName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		obj := e.pkg.TypesInfo.Uses[value]
		if obj == nil {
			obj = e.pkg.TypesInfo.Defs[value]
		}
		name := e.objectName(obj)
		if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == e.pkg.PkgPath && obj.Parent() == e.pkg.Types.Scope() {
			return "GpPackage." + name
		}
		return name
	case *ast.SelectorExpr:
		name := javaIdent(value.Sel.Name, ast.IsExported(value.Sel.Name))
		if selection := e.pkg.TypesInfo.Selections[value]; selection != nil {
			return e.expr(value.X) + "." + javaIdent(value.Sel.Name, selection.Obj().Exported())
		}
		if ident, ok := value.X.(*ast.Ident); ok {
			if pkgName, ok := e.pkg.TypesInfo.Uses[ident].(*types.PkgName); ok {
				return e.mapper.javaPackage(pkgName.Imported().Path()) + ".GpPackage." + name
			}
		}
		return e.expr(value.X) + "." + name
	case *ast.IndexExpr:
		return e.directCallName(value.X)
	case *ast.IndexListExpr:
		return e.directCallName(value.X)
	default:
		return e.expr(expr)
	}
}

func (e *emitter) index(container, index ast.Expr) string {
	value, idx := e.expr(container), e.expr(index)
	t := types.Unalias(e.pkg.TypesInfo.TypeOf(container))
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	switch t.(type) {
	case *types.Map:
		return value + ".get(" + idx + ")"
	case *types.Slice, *types.Array:
		return value + ".get(GpRuntime.index(" + idx + "))"
	case *types.Basic:
		if isStringType(t) {
			return value + ".byteAt(GpRuntime.index(" + idx + "))"
		}
	}
	e.unsupported(container, "indexing %s is not in the portable Java subset", types.TypeString(t, nil))
	return "null"
}

func (e *emitter) composite(lit *ast.CompositeLit) string {
	t := types.Unalias(e.pkg.TypesInfo.TypeOf(lit))
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	if named, ok := t.(*types.Named); ok {
		if strct, ok := named.Underlying().(*types.Struct); ok {
			values := make([]string, strct.NumFields())
			for i := range values {
				values[i] = e.zeroValue(strct.Field(i).Type())
			}
			next := 0
			for _, elt := range lit.Elts {
				if keyed, ok := elt.(*ast.KeyValueExpr); ok {
					ident, _ := keyed.Key.(*ast.Ident)
					if ident == nil {
						e.unsupported(keyed, "non-field struct literal key is not portable")
						continue
					}
					for i := 0; i < strct.NumFields(); i++ {
						if strct.Field(i).Name() == ident.Name {
							values[i] = e.expr(keyed.Value)
							break
						}
					}
				} else if next < len(values) {
					values[next] = e.expr(elt)
					next++
				}
			}
			return "new " + e.javaType(named, false) + "(" + strings.Join(values, ", ") + ")"
		}
		t = named.Underlying()
	}
	switch value := t.(type) {
	case *types.Slice:
		var values []string
		for _, elt := range lit.Elts {
			if _, keyed := elt.(*ast.KeyValueExpr); keyed {
				e.unsupported(elt, "keyed slice/array literals are not yet portable")
			}
			values = append(values, e.copyValue(e.expr(elt), value.Elem()))
		}
		return "GpSlice.ofZero(() -> " + e.zeroValue(value.Elem()) + ", " + e.elementCopier(value.Elem()) + optionalComma(values) + strings.Join(values, ", ") + ")"
	case *types.Array:
		values := make([]string, value.Len())
		for index := range values {
			values[index] = e.zeroValue(value.Elem())
		}
		next := 0
		for _, elt := range lit.Elts {
			if _, keyed := elt.(*ast.KeyValueExpr); keyed {
				e.unsupported(elt, "keyed array literals are not yet portable")
				continue
			}
			if next < len(values) {
				values[next] = e.copyValue(e.expr(elt), value.Elem())
				next++
			}
		}
		return "GpSlice.ofArray(() -> " + e.zeroValue(value.Elem()) + ", " + e.elementCopier(value.Elem()) + optionalComma(values) + strings.Join(values, ", ") + ")"
	case *types.Map:
		var entries []string
		for _, elt := range lit.Elts {
			keyed, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			entries = append(entries, "new GpTuple2<>("+e.copyValue(e.expr(keyed.Key), value.Key())+", "+e.copyValue(e.expr(keyed.Value), value.Elem())+")")
		}
		return "GpMap.of(() -> " + e.zeroValue(value.Elem()) + optionalComma(entries) + strings.Join(entries, ", ") + ")"
	default:
		e.unsupported(lit, "composite literal of %s is not yet portable", types.TypeString(value, nil))
		return "null"
	}
}

func isNilLiteral(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func (e *emitter) nilCheck(value string, t types.Type) string {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		t = types.Unalias(named.Underlying())
	}
	switch t.(type) {
	case *types.Slice, *types.Map, *types.Chan:
		return value + ".isNil()"
	case *types.Pointer, *types.Signature, *types.Interface:
		return value + " == null"
	default:
		return ""
	}
}

func optionalComma(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return ", "
}

func (e *emitter) functionLiteral(lit *ast.FuncLit) string {
	sig, _ := types.Unalias(e.pkg.TypesInfo.TypeOf(lit)).(*types.Signature)
	if sig == nil {
		return "null"
	}
	if sig.Params().Len() > 3 {
		e.unsupported(lit, "function literals with more than three parameters are not yet portable")
		return "null"
	}
	var params []string
	type previousName struct {
		obj     types.Object
		name    string
		existed bool
	}
	var previous []previousName
	for i := 0; i < sig.Params().Len(); i++ {
		param := sig.Params().At(i)
		name := param.Name()
		if name == "" || name == "_" {
			name = fmt.Sprintf("arg%d", i)
		}
		name = javaIdent(name, false) + e.nextTemp("lambda")
		old, existed := e.renames[param]
		previous = append(previous, previousName{obj: param, name: old, existed: existed})
		e.renames[param] = name
		params = append(params, name)
	}
	restoreNames := func() {
		for _, item := range previous {
			if item.existed {
				e.renames[item.obj] = item.name
			} else {
				delete(e.renames, item.obj)
			}
		}
	}
	if len(lit.Body.List) == 1 {
		if ret, ok := lit.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
			out := "(" + strings.Join(params, ", ") + ") -> " + e.expr(ret.Results[0])
			restoreNames()
			return out
		}
	}
	w := newJavaWriter()
	w.line("(%s) -> {", strings.Join(params, ", "))
	w.indent++
	previousSig, previousLambdaVoid := e.currentSig, e.lambdaVoid
	e.currentSig, e.lambdaVoid = sig, sig.Results().Len() == 0
	if containsDefer(lit.Body) {
		w.line("try (var __defer = GpRuntime.deferScope()) {")
		w.indent++
		w.line("__defer.touch();")
		e.block(w, lit.Body)
		w.indent--
		w.line("}")
	} else {
		e.block(w, lit.Body)
	}
	if e.lambdaVoid && (len(lit.Body.List) == 0 || !terminates(lit.Body.List[len(lit.Body.List)-1])) {
		w.line("return null;")
	}
	e.currentSig, e.lambdaVoid = previousSig, previousLambdaVoid
	restoreNames()
	w.indent--
	w.line("}")
	return strings.TrimSpace(string(w.bytes()))
}

func (e *emitter) isInstantiation(expr ast.Expr) bool {
	tv, ok := e.pkg.TypesInfo.Types[expr]
	return ok && tv.IsValue() && tv.Type != nil && isSignature(tv.Type)
}

func isSignature(t types.Type) bool {
	_, ok := types.Unalias(t).(*types.Signature)
	return ok
}

func (e *emitter) stdCall(path, name string, raw []ast.Expr) (string, bool) {
	args := make([]string, len(raw))
	for i, arg := range raw {
		args[i] = e.expr(arg)
	}
	switch path + "." + name {
	case "fmt.Print":
		for i, arg := range raw {
			args[i] = e.printValue(args[i], e.pkg.TypesInfo.TypeOf(arg))
		}
		return "GpRuntime.print(" + strings.Join(args, ", ") + ")", true
	case "fmt.Println":
		for i, arg := range raw {
			args[i] = e.printValue(args[i], e.pkg.TypesInfo.TypeOf(arg))
		}
		return "GpRuntime.println(" + strings.Join(args, ", ") + ")", true
	case "strings.Contains":
		return args[0] + ".toJava().contains(" + args[1] + ".toJava())", true
	case "strings.HasPrefix":
		return args[0] + ".toJava().startsWith(" + args[1] + ".toJava())", true
	case "strings.HasSuffix":
		return args[0] + ".toJava().endsWith(" + args[1] + ".toJava())", true
	case "strings.TrimSpace":
		return "GpString.fromJava(" + args[0] + ".toJava().strip())", true
	case "strings.ToLower":
		return "GpString.fromJava(" + args[0] + ".toJava().toLowerCase(java.util.Locale.ROOT))", true
	case "strings.ToUpper":
		return "GpString.fromJava(" + args[0] + ".toJava().toUpperCase(java.util.Locale.ROOT))", true
	case "strconv.Itoa":
		return "GpString.fromJava(Long.toString(" + args[0] + "))", true
	}
	if path == "fmt" || path == "strings" || path == "strconv" {
		e.unsupported(nil, "standard-library adapter %s.%s is not implemented for Java", path, name)
		return "null", true
	}
	return "", false
}

func (e *emitter) printValue(value string, t types.Type) string {
	if isUnsigned(t) {
		return "Long.toUnsignedString(" + value + ")"
	}
	return value
}
