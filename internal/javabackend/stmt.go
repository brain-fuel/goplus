package javabackend

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

func (e *emitter) block(w *javaWriter, block *ast.BlockStmt) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		e.stmt(w, stmt)
	}
}

func (e *emitter) stmt(w *javaWriter, node ast.Stmt) {
	switch value := node.(type) {
	case *ast.EmptyStmt:
		return
	case *ast.BlockStmt:
		w.line("{")
		w.indent++
		e.block(w, value)
		w.indent--
		w.line("}")
	case *ast.ExprStmt:
		if call, ok := value.X.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok {
				if builtin, ok := e.pkg.TypesInfo.Uses[ident].(*types.Builtin); ok && builtin.Name() == "panic" && len(call.Args) == 1 {
					w.line("throw GpRuntime.panicValue(%s);", e.expr(call.Args[0]))
					return
				}
			}
		}
		w.line("%s;", e.expr(value.X))
	case *ast.DeclStmt:
		e.localDecl(w, value.Decl)
	case *ast.AssignStmt:
		e.assignment(w, value)
	case *ast.ReturnStmt:
		e.returnStmt(w, value)
	case *ast.IfStmt:
		e.ifStmt(w, value)
	case *ast.ForStmt:
		e.forStmt(w, value)
	case *ast.RangeStmt:
		e.rangeStmt(w, value)
	case *ast.IncDecStmt:
		e.incDec(w, value)
	case *ast.BranchStmt:
		switch value.Tok {
		case token.BREAK, token.CONTINUE:
			label := ""
			if value.Label != nil {
				label = " " + javaIdent(value.Label.Name, false)
			}
			w.line("%s%s;", value.Tok.String(), label)
		default:
			e.unsupported(value, "%s is not yet in the portable Java subset", value.Tok)
		}
	case *ast.SwitchStmt:
		e.switchStmt(w, value)
	case *ast.TypeSwitchStmt:
		e.typeSwitchStmt(w, value)
	case *ast.SendStmt:
		w.line("%s.send(%s);", e.expr(value.Chan), e.copyValue(e.expr(value.Value), elementType(e.pkg.TypesInfo.TypeOf(value.Chan))))
	case *ast.GoStmt:
		e.asyncCall(w, "GpRuntime.go", value.Call)
	case *ast.DeferStmt:
		e.asyncCall(w, "GpRuntime.defer", value.Call)
	case *ast.SelectStmt:
		e.selectStmt(w, value)
	case *ast.LabeledStmt:
		w.line("%s:", javaIdent(value.Label.Name, false))
		e.stmt(w, value.Stmt)
	default:
		e.unsupported(node, "Go statement %T is not yet in the portable Java subset", node)
	}
}

func (e *emitter) typeSwitchStmt(w *javaWriter, stmt *ast.TypeSwitchStmt) {
	if stmt.Init != nil {
		e.stmt(w, stmt.Init)
	}
	var asserted ast.Expr
	var binding *ast.Ident
	switch assign := stmt.Assign.(type) {
	case *ast.AssignStmt:
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			e.unsupported(stmt, "type-switch binding must have one value")
			return
		}
		binding, _ = assign.Lhs[0].(*ast.Ident)
		assertion, _ := assign.Rhs[0].(*ast.TypeAssertExpr)
		if assertion != nil {
			asserted = assertion.X
		}
	case *ast.ExprStmt:
		assertion, _ := assign.X.(*ast.TypeAssertExpr)
		if assertion != nil {
			asserted = assertion.X
		}
	}
	if asserted == nil {
		e.unsupported(stmt, "invalid type-switch assertion")
		return
	}
	temp := e.nextTemp("typeSwitch")
	w.line("Object %s = %s;", temp, e.expr(asserted))
	first := true
	var defaultClause *ast.CaseClause
	for _, raw := range stmt.Body.List {
		clause := raw.(*ast.CaseClause)
		if len(clause.List) == 0 {
			defaultClause = clause
			continue
		}
		if len(clause.List) != 1 {
			e.unsupported(clause, "multi-type switch cases are not yet portable")
			continue
		}
		caseType := clause.List[0]
		condition := ""
		decl := ""
		if ident, ok := caseType.(*ast.Ident); ok && ident.Name == "nil" {
			condition = temp + " == null"
		} else {
			javaType := e.javaType(e.pkg.TypesInfo.TypeOf(caseType), false)
			condition = temp + " instanceof " + reifiableJavaType(javaType)
			if binding != nil && binding.Name != "_" {
				if javaType != reifiableJavaType(javaType) {
					decl = "@SuppressWarnings(\"unchecked\") "
				}
				decl += javaType + " " + javaIdent(binding.Name, false) + " = (" + javaType + ") " + temp + ";"
			}
		}
		if first {
			w.line("if (%s) {", condition)
			first = false
		} else {
			w.line("else if (%s) {", condition)
		}
		w.indent++
		if decl != "" {
			w.line("%s", decl)
		}
		e.withTypeSwitchBinding(binding, clause, func() {
			for _, body := range clause.Body {
				e.stmt(w, body)
			}
		})
		w.indent--
		w.line("}")
	}
	if defaultClause != nil {
		if first {
			w.line("{")
		} else {
			w.line("else {")
		}
		w.indent++
		if binding != nil && binding.Name != "_" {
			w.line("Object %s = %s;", javaIdent(binding.Name, false), temp)
		}
		e.withTypeSwitchBinding(binding, defaultClause, func() {
			for _, body := range defaultClause.Body {
				e.stmt(w, body)
			}
		})
		w.indent--
		w.line("}")
	}
}

func reifiableJavaType(value string) string {
	start := strings.IndexByte(value, '<')
	if start < 0 {
		return value
	}
	end := strings.LastIndexByte(value, '>')
	if end < start {
		return value[:start]
	}
	count := strings.Count(value[start+1:end], ",") + 1
	return value[:start] + "<" + strings.TrimSuffix(strings.Repeat("?, ", count), ", ") + ">"
}

func (e *emitter) withTypeSwitchBinding(binding *ast.Ident, clause *ast.CaseClause, emit func()) {
	if binding == nil {
		emit()
		return
	}
	obj := e.pkg.TypesInfo.Implicits[clause]
	if obj == nil {
		emit()
		return
	}
	old, existed := e.renames[obj]
	e.renames[obj] = javaIdent(binding.Name, false)
	emit()
	if existed {
		e.renames[obj] = old
	} else {
		delete(e.renames, obj)
	}
}

func (e *emitter) localDecl(w *javaWriter, decl ast.Decl) {
	gen, ok := decl.(*ast.GenDecl)
	if !ok || (gen.Tok != token.VAR && gen.Tok != token.CONST) {
		e.unsupported(decl, "local declaration %T is not yet portable", decl)
		return
	}
	for _, raw := range gen.Specs {
		spec, ok := raw.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if len(spec.Values) == 1 && len(spec.Names) > 1 {
			e.emitTupleBinding(w, spec.Names, spec.Values[0], true, gen.Tok == token.CONST)
			continue
		}
		for i, name := range spec.Names {
			if name.Name == "_" {
				continue
			}
			obj := e.pkg.TypesInfo.Defs[name]
			if obj == nil {
				continue
			}
			init := e.zeroValue(obj.Type())
			if i < len(spec.Values) {
				init = e.expr(spec.Values[i])
			}
			modifier := ""
			if gen.Tok == token.CONST {
				modifier = "final "
			}
			w.line("%s%s %s = %s;", modifier, e.javaType(obj.Type(), false), e.objectName(obj), e.copyValue(init, obj.Type()))
		}
	}
}

func (e *emitter) assignment(w *javaWriter, stmt *ast.AssignStmt) {
	if len(stmt.Rhs) == 1 && len(stmt.Lhs) > 1 {
		var names []*ast.Ident
		for _, lhs := range stmt.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				e.unsupported(stmt, "multi-result assignment requires identifier destinations")
				return
			}
			names = append(names, ident)
		}
		e.emitTupleBinding(w, names, stmt.Rhs[0], stmt.Tok == token.DEFINE, false)
		return
	}
	if len(stmt.Lhs) != len(stmt.Rhs) {
		e.unsupported(stmt, "assignment arity %d = %d is not portable", len(stmt.Lhs), len(stmt.Rhs))
		return
	}
	// Evaluate every RHS before changing an LHS, matching Go's parallel
	// assignment rule (and making swaps deterministic).
	temps := make([]string, len(stmt.Rhs))
	for i, rhs := range stmt.Rhs {
		temps[i] = e.nextTemp("assign")
		t := e.pkg.TypesInfo.TypeOf(rhs)
		w.line("%s %s = %s;", e.javaType(t, false), temps[i], e.copyValue(e.expr(rhs), t))
	}
	for i, lhs := range stmt.Lhs {
		e.assignOne(w, lhs, temps[i], stmt.Tok, stmt.Tok == token.DEFINE)
	}
}

func (e *emitter) emitTupleBinding(w *javaWriter, names []*ast.Ident, rhs ast.Expr, define, constant bool) {
	tuple, _ := types.Unalias(e.pkg.TypesInfo.TypeOf(rhs)).(*types.Tuple)
	if tuple == nil || tuple.Len() != len(names) || tuple.Len() < 2 || tuple.Len() > 3 {
		e.unsupported(rhs, "multi-result binding requires a two- or three-value function result")
		return
	}
	temp := e.nextTemp("tuple")
	w.line("var %s = %s;", temp, e.tupleExpr(rhs))
	accessors := []string{"first()", "second()", "third()"}
	for i, name := range names {
		if name.Name == "_" {
			continue
		}
		value := e.copyValue(temp+"."+accessors[i], tuple.At(i).Type())
		if define {
			obj := e.pkg.TypesInfo.Defs[name]
			if obj == nil {
				obj = e.pkg.TypesInfo.Uses[name]
			}
			modifier := ""
			if constant {
				modifier = "final "
			}
			w.line("%s%s %s = %s;", modifier, e.javaType(tuple.At(i).Type(), false), javaIdent(name.Name, false), value)
		} else {
			w.line("%s = %s;", e.expr(name), value)
		}
	}
}

func (e *emitter) tupleExpr(rhs ast.Expr) string {
	if index, ok := rhs.(*ast.IndexExpr); ok {
		if _, ok := underlyingType(e.pkg.TypesInfo.TypeOf(index.X)).(*types.Map); ok {
			return e.expr(index.X) + ".lookup(" + e.expr(index.Index) + ")"
		}
	}
	if receive, ok := rhs.(*ast.UnaryExpr); ok && receive.Op == token.ARROW {
		return e.expr(receive.X) + ".receiveOk()"
	}
	return e.expr(rhs)
}

func (e *emitter) assignOne(w *javaWriter, lhs ast.Expr, rhs string, tok token.Token, define bool) {
	if ident, ok := lhs.(*ast.Ident); ok {
		if ident.Name == "_" {
			return
		}
		if define {
			if obj := e.pkg.TypesInfo.Defs[ident]; obj != nil {
				w.line("%s %s = %s;", e.javaType(obj.Type(), false), e.objectName(obj), rhs)
				return
			}
		}
		if tok == token.ASSIGN || tok == token.DEFINE {
			w.line("%s = %s;", e.expr(ident), rhs)
		} else {
			left := e.expr(ident)
			w.line("%s = %s;", left, e.compound(left, rhs, e.pkg.TypesInfo.TypeOf(lhs), tok))
		}
		return
	}
	if index, ok := lhs.(*ast.IndexExpr); ok {
		container, key := e.expr(index.X), e.expr(index.Index)
		t := types.Unalias(e.pkg.TypesInfo.TypeOf(index.X))
		if named, ok := t.(*types.Named); ok {
			t = named.Underlying()
		}
		current := e.index(index.X, index.Index)
		if tok != token.ASSIGN {
			rhs = e.compound(current, rhs, e.pkg.TypesInfo.TypeOf(lhs), tok)
		}
		switch t.(type) {
		case *types.Map:
			mapType := t.(*types.Map)
			w.line("%s.put(%s, %s);", container, e.copyValue(key, mapType.Key()), rhs)
		case *types.Slice, *types.Array:
			w.line("%s.set(GpRuntime.index(%s), %s);", container, key, rhs)
		default:
			e.unsupported(lhs, "indexed assignment to %s is not portable", types.TypeString(t, nil))
		}
		return
	}
	if star, ok := lhs.(*ast.StarExpr); ok {
		w.line("%s.set(%s);", e.expr(star.X), rhs)
		return
	}
	if tok == token.ASSIGN {
		w.line("%s = %s;", e.expr(lhs), rhs)
		return
	}
	left := e.expr(lhs)
	w.line("%s = %s;", left, e.compound(left, rhs, e.pkg.TypesInfo.TypeOf(lhs), tok))
}

func (e *emitter) compound(left, right string, t types.Type, tok token.Token) string {
	op := strings.TrimSuffix(tok.String(), "=")
	if tok == token.ADD_ASSIGN && isStringType(t) {
		return left + ".concat(" + right + ")"
	}
	if isUnsigned(t) {
		switch tok {
		case token.QUO_ASSIGN:
			return e.normalizeNumeric("Long.divideUnsigned("+left+", "+right+")", t)
		case token.REM_ASSIGN:
			return e.normalizeNumeric("Long.remainderUnsigned("+left+", "+right+")", t)
		case token.SHR_ASSIGN:
			return e.normalizeNumeric("GpRuntime.shiftRightUnsigned("+left+", "+right+")", t)
		}
	}
	switch tok {
	case token.SHL_ASSIGN:
		return e.normalizeNumeric("GpRuntime.shiftLeft("+left+", "+right+")", t)
	case token.SHR_ASSIGN:
		return e.normalizeNumeric("GpRuntime.shiftRight("+left+", "+right+")", t)
	case token.AND_NOT_ASSIGN:
		return e.normalizeNumeric("("+left+" & ~("+right+"))", t)
	default:
		return e.normalizeNumeric("("+left+" "+op+" "+right+")", t)
	}
}

func (e *emitter) returnStmt(w *javaWriter, stmt *ast.ReturnStmt) {
	if len(stmt.Results) == 0 {
		if e.currentSig != nil && e.currentSig.Results().Len() > 0 {
			var results []string
			for i := 0; i < e.currentSig.Results().Len(); i++ {
				result := e.currentSig.Results().At(i)
				if result.Name() == "" {
					e.unsupported(stmt, "naked return without named results")
					return
				}
				results = append(results, e.copyValue(e.objectName(result), result.Type()))
			}
			if len(results) == 1 {
				w.line("return %s;", results[0])
			} else {
				w.line("return new GpTuple%d<>(%s);", len(results), strings.Join(results, ", "))
			}
			return
		}
		if e.lambdaVoid {
			w.line("return null;")
		} else {
			w.line("return;")
		}
		return
	}
	if len(stmt.Results) == 1 {
		resultType := e.pkg.TypesInfo.TypeOf(stmt.Results[0])
		if e.currentSig != nil && e.currentSig.Results().Len() == 1 {
			resultType = e.currentSig.Results().At(0).Type()
		}
		w.line("return %s;", e.copyValue(e.expr(stmt.Results[0]), resultType))
		return
	}
	var values []string
	for _, result := range stmt.Results {
		resultType := e.pkg.TypesInfo.TypeOf(result)
		if e.currentSig != nil && len(values) < e.currentSig.Results().Len() {
			resultType = e.currentSig.Results().At(len(values)).Type()
		}
		values = append(values, e.copyValue(e.expr(result), resultType))
	}
	w.line("return new GpTuple%d<>(%s);", len(values), strings.Join(values, ", "))
}

func (e *emitter) ifStmt(w *javaWriter, stmt *ast.IfStmt) {
	wrapped := stmt.Init != nil
	if wrapped {
		w.line("{")
		w.indent++
		e.stmt(w, stmt.Init)
	}
	w.line("if (%s) {", e.expr(stmt.Cond))
	w.indent++
	e.block(w, stmt.Body)
	w.indent--
	if stmt.Else == nil {
		w.line("}")
	} else {
		w.line("} else {")
		w.indent++
		if block, ok := stmt.Else.(*ast.BlockStmt); ok {
			e.block(w, block)
		} else {
			e.stmt(w, stmt.Else)
		}
		w.indent--
		w.line("}")
	}
	if wrapped {
		w.indent--
		w.line("}")
	}
}

func (e *emitter) forStmt(w *javaWriter, stmt *ast.ForStmt) {
	init := ""
	if stmt.Init != nil {
		init = e.inlineStmt(stmt.Init)
	}
	cond := "true"
	if stmt.Cond != nil {
		cond = e.expr(stmt.Cond)
	}
	post := ""
	if stmt.Post != nil {
		post = e.inlineStmt(stmt.Post)
	}
	w.line("for (%s; %s; %s) {", init, cond, post)
	w.indent++
	e.block(w, stmt.Body)
	w.indent--
	w.line("}")
}

func (e *emitter) inlineStmt(stmt ast.Stmt) string {
	switch value := stmt.(type) {
	case *ast.AssignStmt:
		if len(value.Lhs) != 1 || len(value.Rhs) != 1 {
			e.unsupported(stmt, "multi-assignment is not valid in a Java for clause")
			return ""
		}
		if value.Tok == token.DEFINE {
			ident, _ := value.Lhs[0].(*ast.Ident)
			obj := e.pkg.TypesInfo.Defs[ident]
			if ident == nil || obj == nil {
				return ""
			}
			return e.javaType(obj.Type(), false) + " " + e.objectName(obj) + " = " + e.expr(value.Rhs[0])
		}
		return e.expr(value.Lhs[0]) + " " + value.Tok.String() + " " + e.expr(value.Rhs[0])
	case *ast.IncDecStmt:
		tok := token.ADD_ASSIGN
		if value.Tok == token.DEC {
			tok = token.SUB_ASSIGN
		}
		left := e.expr(value.X)
		return left + " = " + e.compound(left, "1", e.pkg.TypesInfo.TypeOf(value.X), tok)
	case *ast.ExprStmt:
		return e.expr(value.X)
	default:
		e.unsupported(stmt, "statement %T is not valid in a portable Java for clause", stmt)
		return ""
	}
}

func (e *emitter) rangeStmt(w *javaWriter, stmt *ast.RangeStmt) {
	originalType := e.pkg.TypesInfo.TypeOf(stmt.X)
	t := types.Unalias(originalType)
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	container := e.nextTemp("range")
	w.line("{")
	w.indent++
	w.line("var %s = %s;", container, e.copyValue(e.expr(stmt.X), originalType))
	switch value := t.(type) {
	case *types.Slice, *types.Array:
		idx := e.nextTemp("index")
		w.line("for (long %s = 0; %s < %s.length(); %s++) {", idx, idx, container, idx)
		w.indent++
		e.rangeBind(w, stmt.Key, idx, stmt.Tok)
		e.rangeBind(w, stmt.Value, container+".get(GpRuntime.index("+idx+"))", stmt.Tok)
		e.block(w, stmt.Body)
		w.indent--
		w.line("}")
	case *types.Map:
		entry := e.nextTemp("entry")
		w.line("for (var %s : %s.entries()) {", entry, container)
		w.indent++
		e.rangeBind(w, stmt.Key, entry+".getKey()", stmt.Tok)
		e.rangeBind(w, stmt.Value, entry+".getValue()", stmt.Tok)
		e.block(w, stmt.Body)
		w.indent--
		w.line("}")
	case *types.Chan:
		if stmt.Key != nil && stmt.Value != nil {
			e.unsupported(stmt, "channel range has one value")
		}
		valueName := e.nextTemp("value")
		w.line("for (var %s : %s) {", valueName, container)
		w.indent++
		dest := stmt.Key
		if stmt.Value != nil {
			dest = stmt.Value
		}
		e.rangeBind(w, dest, valueName, stmt.Tok)
		e.block(w, stmt.Body)
		w.indent--
		w.line("}")
	default:
		e.unsupported(stmt, "range over %s is not yet in the portable Java subset", types.TypeString(value, nil))
	}
	w.indent--
	w.line("}")
}

func (e *emitter) rangeBind(w *javaWriter, destination ast.Expr, value string, tok token.Token) {
	if destination == nil {
		return
	}
	ident, ok := destination.(*ast.Ident)
	if ok && ident.Name == "_" {
		return
	}
	if tok == token.DEFINE && ok {
		obj := e.pkg.TypesInfo.Defs[ident]
		if obj != nil {
			w.line("%s %s = %s;", e.javaType(obj.Type(), false), e.objectName(obj), e.copyValue(value, obj.Type()))
			return
		}
	}
	w.line("%s = %s;", e.expr(destination), e.copyValue(value, e.pkg.TypesInfo.TypeOf(destination)))
}

func (e *emitter) incDec(w *javaWriter, stmt *ast.IncDecStmt) {
	op := token.ADD_ASSIGN
	if stmt.Tok == token.DEC {
		op = token.SUB_ASSIGN
	}
	e.assignOne(w, stmt.X, "1", op, false)
}

func (e *emitter) switchStmt(w *javaWriter, stmt *ast.SwitchStmt) {
	w.line("{")
	w.indent++
	if stmt.Init != nil {
		e.stmt(w, stmt.Init)
	}
	tag := "true"
	if stmt.Tag != nil {
		tag = e.nextTemp("switch")
		w.line("var %s = %s;", tag, e.expr(stmt.Tag))
	}
	w.line("do {")
	w.indent++
	clauses := append([]ast.Stmt(nil), stmt.Body.List...)
	sort.SliceStable(clauses, func(i, j int) bool {
		left := clauses[i].(*ast.CaseClause)
		right := clauses[j].(*ast.CaseClause)
		return len(left.List) > 0 && len(right.List) == 0
	})
	first := true
	for _, raw := range clauses {
		clause := raw.(*ast.CaseClause)
		condition := "true"
		if len(clause.List) > 0 {
			var tests []string
			for _, item := range clause.List {
				if stmt.Tag == nil {
					tests = append(tests, e.expr(item))
				} else {
					tests = append(tests, "GpRuntime.equal("+tag+", "+e.expr(item)+")")
				}
			}
			condition = "(" + strings.Join(tests, " || ") + ")"
		}
		prefix := "if"
		if !first {
			prefix = "else if"
		}
		w.line("%s (%s) {", prefix, condition)
		w.indent++
		for _, body := range clause.Body {
			e.stmt(w, body)
		}
		w.indent--
		w.line("}")
		first = false
	}
	w.indent--
	w.line("} while (false);")
	w.indent--
	w.line("}")
}

func (e *emitter) asyncCall(w *javaWriter, helper string, call *ast.CallExpr) {
	if call == nil {
		return
	}
	sig, _ := types.Unalias(e.pkg.TypesInfo.TypeOf(call.Fun)).(*types.Signature)
	if sig == nil {
		e.unsupported(call, "cannot recover deferred/asynchronous call signature")
		return
	}
	direct := isDirectCall(e, call.Fun)
	fun := ""
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if selection := e.pkg.TypesInfo.Selections[selector]; selection != nil && selection.Kind() == types.MethodVal {
			receiverType := e.pkg.TypesInfo.TypeOf(selector.X)
			receiver := e.nextTemp("receiver")
			w.line("%s %s = %s;", e.javaType(receiverType, false), receiver, e.copyValue(e.expr(selector.X), receiverType))
			fun = receiver + "." + javaIdent(selector.Sel.Name, selection.Obj().Exported())
		}
	}
	if fun == "" && direct {
		fun = e.directCallName(call.Fun)
	} else if fun == "" {
		name := e.nextTemp("fun")
		w.line("%s %s = %s;", e.functionType(sig), name, e.expr(call.Fun))
		fun = name + ".apply"
	}
	callArgs := e.callArguments(call, sig)
	var args []string
	for i, arg := range callArgs {
		name := e.nextTemp("arg")
		argType := sig.Params().At(i).Type()
		w.line("%s %s = %s;", e.javaType(argType, false), name, arg)
		args = append(args, name)
	}
	w.line("%s(() -> %s(%s));", helper, fun, strings.Join(args, ", "))
}

func isDirectCall(e *emitter, fun ast.Expr) bool {
	switch value := fun.(type) {
	case *ast.Ident:
		_, ok := e.pkg.TypesInfo.Uses[value].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if selection := e.pkg.TypesInfo.Selections[value]; selection != nil {
			return selection.Kind() == types.MethodVal
		}
		_, ok := e.pkg.TypesInfo.Uses[value.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return isDirectCall(e, value.X)
	case *ast.IndexListExpr:
		return isDirectCall(e, value.X)
	default:
		return false
	}
}

func (e *emitter) selectStmt(w *javaWriter, stmt *ast.SelectStmt) {
	type selectCase struct {
		clause  *ast.CommClause
		kind    string
		channel ast.Expr
		value   ast.Expr
		assign  *ast.AssignStmt
	}
	var cases []selectCase
	var constructors []string
	for _, raw := range stmt.Body.List {
		clause := raw.(*ast.CommClause)
		entry := selectCase{clause: clause}
		switch comm := clause.Comm.(type) {
		case nil:
			entry.kind = "default"
			constructors = append(constructors, "new GpSelect.Default<>()")
		case *ast.SendStmt:
			entry.kind, entry.channel, entry.value = "send", comm.Chan, comm.Value
			constructors = append(constructors, "new GpSelect.Send<>("+e.expr(comm.Chan)+", "+e.copyValue(e.expr(comm.Value), elementType(e.pkg.TypesInfo.TypeOf(comm.Chan)))+")")
		case *ast.ExprStmt:
			recv, ok := comm.X.(*ast.UnaryExpr)
			if !ok || recv.Op != token.ARROW {
				e.unsupported(comm, "select expression is not a receive")
				continue
			}
			entry.kind, entry.channel = "receive", recv.X
			constructors = append(constructors, "new GpSelect.Receive<>("+e.expr(recv.X)+")")
		case *ast.AssignStmt:
			if len(comm.Rhs) != 1 {
				e.unsupported(comm, "select receive assignment has multiple RHS values")
				continue
			}
			recv, ok := comm.Rhs[0].(*ast.UnaryExpr)
			if !ok || recv.Op != token.ARROW {
				e.unsupported(comm, "select assignment is not a receive")
				continue
			}
			entry.kind, entry.channel, entry.assign = "receive", recv.X, comm
			constructors = append(constructors, "new GpSelect.Receive<>("+e.expr(recv.X)+")")
		default:
			e.unsupported(comm, "select communication %T is not portable", comm)
			continue
		}
		cases = append(cases, entry)
	}
	selection := e.nextTemp("select")
	w.line("{")
	w.indent++
	w.line("var %s = GpSelect.select(java.util.List.of(%s));", selection, strings.Join(constructors, ", "))
	w.line("switch (%s.index()) {", selection)
	w.indent++
	for i, entry := range cases {
		w.line("case %d -> {", i)
		w.indent++
		if entry.assign != nil {
			tuple, _ := types.Unalias(e.pkg.TypesInfo.TypeOf(entry.assign.Rhs[0])).(*types.Tuple)
			for j, lhs := range entry.assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name == "_" {
					continue
				}
				rhs := selection + (map[bool]string{true: ".ok()", false: ".value()"}[j == 1])
				if j == 0 {
					t := e.pkg.TypesInfo.TypeOf(entry.assign.Rhs[0])
					if tuple != nil && tuple.Len() > 0 {
						t = tuple.At(0).Type()
					}
					rhs = "((" + e.javaType(t, true) + ") " + selection + ".value())"
				}
				if entry.assign.Tok == token.DEFINE {
					obj := e.pkg.TypesInfo.Defs[ident]
					w.line("%s %s = %s;", e.javaType(obj.Type(), false), e.objectName(obj), rhs)
				} else {
					w.line("%s = %s;", e.expr(ident), rhs)
				}
			}
		}
		for _, body := range entry.clause.Body {
			e.stmt(w, body)
		}
		w.indent--
		w.line("}")
	}
	w.line("default -> throw new IllegalStateException(\"invalid select result\");")
	w.indent--
	w.line("}")
	w.indent--
	w.line("}")
}

func (e *emitter) nextTemp(kind string) string {
	e.temp++
	return fmt.Sprintf("__%s%d", kind, e.temp)
}

func terminates(stmt ast.Stmt) bool {
	switch value := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BlockStmt:
		return len(value.List) > 0 && terminates(value.List[len(value.List)-1])
	case *ast.IfStmt:
		return value.Else != nil && terminates(value.Body) && terminates(value.Else)
	default:
		return false
	}
}
