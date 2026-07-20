package resolve

import (
	"fmt"
	"go/ast"
	"strings"

	"goforge.dev/goplus/internal/lower"
	"goforge.dev/goplus/internal/naming"
	"goforge.dev/goplus/internal/registry"
)

// erasedGADTFunction lowers a generic function containing an otherwise
// unspellable flat GADT match into a typed facade plus an erased companion.
// The companion is an implementation detail; authored parameter and result
// types remain on the facade.
func (r *fileResolver) erasedGADTFunction(sw *ast.TypeSwitchStmt, e *registry.Enum, arms []*armAnalysis, subjText string, hasWildcard bool) bool {
	fd := r.enclosingFuncDecl(sw)
	if fd == nil || fd.Recv != nil || fd.Type.TypeParams == nil || fd.Body == nil {
		if r.report {
			r.errorf(sw.Pos(), "this GADT match requires erased elimination, currently available in a generic top-level function")
		}
		return false
	}
	tparams := fieldNames(fd.Type.TypeParams)
	tset := map[string]bool{}
	for _, name := range tparams {
		tset[name] = true
	}
	params, args, ok := r.erasedParams(fd, tset)
	if !ok {
		return false
	}
	resultText, facadeResult, ok := r.erasedResult(fd, tset)
	if !ok {
		return false
	}
	helper := "__goplus_erased_" + fd.Name.Name
	call := helper + "(" + strings.Join(args, ", ") + ")"
	facadeHead := r.text(fd.Pos(), fd.Body.Lbrace+1)
	var facade string
	switch {
	case fd.Type.Results == nil:
		facade = facadeHead + "\n\t" + call + "\n}"
	case facadeResult != "":
		facade = facadeHead + "\n\treturn any(" + call + ").(" + facadeResult + ")\n}"
	default:
		facade = facadeHead + "\n\treturn " + call + "\n}"
	}

	erasedMatch := r.emitErasedMatch(e, arms, subjText, hasWildcard, helper)
	bodyStart, bodyEnd := r.off(fd.Body.Lbrace)+1, r.off(fd.Body.Rbrace)
	body := string(r.src[bodyStart:bodyEnd])
	matchStart, matchEnd := r.off(sw.Pos())-bodyStart, r.off(sw.End())-bodyStart
	if matchStart < 0 || matchEnd < matchStart || matchEnd > len(body) {
		return false
	}
	body = body[:matchStart] + erasedMatch + body[matchEnd:]
	companion := fmt.Sprintf("func %s(%s)%s {%s}", helper, params, resultText, body)
	r.edits = append(r.edits, lower.Edit{Start: r.off(fd.Pos()), End: r.off(fd.End()), New: facade + "\n\n" + companion})
	return true
}

func fieldNames(list *ast.FieldList) []string {
	var out []string
	if list != nil {
		for _, field := range list.List {
			for _, name := range field.Names {
				out = append(out, name.Name)
			}
		}
	}
	return out
}

func (r *fileResolver) erasedParams(fd *ast.FuncDecl, tset map[string]bool) (string, []string, bool) {
	var decls, args []string
	for _, field := range fd.Type.Params.List {
		if len(field.Names) == 0 {
			if r.report {
				r.errorf(field.Pos(), "erased GADT elimination requires named function parameters")
			}
			return "", nil, false
		}
		typeText := r.text(field.Type.Pos(), field.Type.End())
		erasedType := typeText
		if textHasTParam(typeText, tset) {
			if _, variadic := field.Type.(*ast.Ellipsis); variadic {
				erasedType = "...any"
			} else {
				erasedType = "any"
			}
		}
		for _, name := range field.Names {
			decls = append(decls, name.Name+" "+erasedType)
			arg := name.Name
			if _, variadic := field.Type.(*ast.Ellipsis); variadic {
				arg += "..."
			}
			args = append(args, arg)
		}
	}
	return strings.Join(decls, ", "), args, true
}

func (r *fileResolver) erasedResult(fd *ast.FuncDecl, tset map[string]bool) (companion, facadeAssertion string, ok bool) {
	if fd.Type.Results == nil {
		return "", "", true
	}
	count := 0
	var typ ast.Expr
	for _, field := range fd.Type.Results.List {
		n := len(field.Names)
		if n == 0 {
			n = 1
		}
		count += n
		typ = field.Type
	}
	if count != 1 {
		if r.report {
			r.errorf(fd.Type.Results.Pos(), "erased GADT elimination currently requires zero or one result")
		}
		return "", "", false
	}
	text := r.text(typ.Pos(), typ.End())
	if textHasTParam(text, tset) {
		return " any", text, true
	}
	return " " + text, "", true
}

func (r *fileResolver) emitErasedMatch(e *registry.Enum, arms []*armAnalysis, subjText string, hasWildcard bool, helper string) string {
	view := naming.ErasedViewAnyName(e.Name)
	if e.PkgPath != r.pkg.PkgPath {
		if alias, ok := r.importName(e.PkgPath); ok {
			view = alias + "." + view
		}
	}
	var b strings.Builder
	b.WriteString("{\n")
	fmt.Fprintf(&b, "__gp_tag, __gp_payload := %s(%s)\n", view, subjText)
	b.WriteString("switch __gp_tag {\n")
	for _, arm := range arms {
		if arm.pat.wild {
			b.WriteString("default:\n")
			b.WriteString(arm.body + "\n")
			continue
		}
		tag := variantIndex(e, arm.pat.variant)
		fmt.Fprintf(&b, "case %d:\n", tag)
		if arm.binderName != "" && identReferencedInText(arm.body, arm.binderName) {
			fmt.Fprintf(&b, "%s := %s\n", arm.binderName, subjText)
		}
		unknown := map[string]bool{}
		for i, arg := range arm.pat.args {
			if arg.binder == "" || !identReferencedInText(arm.body, arg.binder) {
				continue
			}
			if typ, ok := erasedFieldType(e, arm.pat.variant, arm.pat.variant.Params[i]); ok {
				fmt.Fprintf(&b, "%s := __gp_payload[%d].(%s)\n", arg.binder, i, typ)
			} else {
				fmt.Fprintf(&b, "%s := __gp_payload[%d]\n", arg.binder, i)
				unknown[arg.binder] = true
			}
		}
		body := arm.body
		for name := range unknown {
			body = strings.ReplaceAll(body, fdCallSpelling(helper, name), helper+"("+name+")")
		}
		b.WriteString(body + "\n")
	}
	if !hasWildcard {
		b.WriteString("default:\npanic(\"goplus: impossible enum value in match\")\n")
	}
	b.WriteString("}\n}")
	return b.String()
}

func fdCallSpelling(helper, arg string) string {
	name := strings.TrimPrefix(helper, "__goplus_erased_")
	return name + "(" + arg + ")"
}

func variantIndex(e *registry.Enum, want *registry.EnumVariant) int {
	for i, v := range e.Variants {
		if v.Name == want.Name {
			return i
		}
	}
	return -1
}

func erasedFieldType(e *registry.Enum, v *registry.EnumVariant, field registry.EnumParam) (string, bool) {
	tset := map[string]bool{}
	for _, name := range e.TParams {
		tset[name] = true
	}
	subst := map[string]string{}
	if v.ResultArgs != nil {
		for i, arg := range v.ResultArgs {
			if i < len(e.TParams) && !textHasTParam(arg, tset) {
				subst[e.TParams[i]] = arg
			}
		}
	}
	text, err := substTypeTextLite(field.Type, subst)
	if err != nil || textHasTParam(text, tset) {
		return "", false
	}
	return text, true
}
