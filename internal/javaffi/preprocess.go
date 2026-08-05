// Package javaffi prepares target-tagged java: imports for the ordinary Go+
// elaborator. The durable markers it adds are consumed by the Java backend;
// none of the synthetic declarations are exposed in Java artifacts.
package javaffi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"goforge.dev/goplus/internal/lower"
)

// Prepare returns an overlay containing rewritten .gp files. Existing overlay
// entries win over disk contents.
func Prepare(root string, input map[string][]byte) (map[string][]byte, error) {
	overlay := make(map[string][]byte, len(input))
	for path, data := range input {
		overlay[path] = data
	}
	paths := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "gen") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".gp") {
			paths[path] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for path := range input {
		if strings.HasSuffix(path, ".gp") {
			paths[path] = true
		}
	}
	var ordered []string
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	for _, path := range ordered {
		src, ok := overlay[path]
		if !ok {
			var err error
			src, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
		}
		prepared, changed, err := prepareFile(path, src)
		if err != nil {
			return nil, err
		}
		if changed {
			overlay[path] = prepared
		}
	}
	return overlay, nil
}

type importedType struct {
	Name, Owner string
	TParams     int
}

func prepareFile(path string, src []byte) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, false, err
	}
	tokenFile := fset.File(file.Pos())
	offset := func(pos token.Pos) int { return tokenFile.Offset(pos) }
	direct := map[string]*importedType{}
	packages := map[string]string{}
	var edits []lower.Edit
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		javaSpecs := 0
		for _, raw := range gen.Specs {
			spec := raw.(*ast.ImportSpec)
			value, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			kind, target, ok := strings.Cut(value, "/")
			if !ok || (kind != "java:type" && kind != "java:package") {
				continue
			}
			javaSpecs++
			alias := ""
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias == "" {
				parts := strings.Split(target, ".")
				alias = parts[len(parts)-1]
			}
			if alias == "_" || alias == "." {
				return nil, false, fmt.Errorf("%s: java: import needs a named alias", fset.Position(spec.Pos()))
			}
			if kind == "java:type" {
				direct[alias] = &importedType{Name: alias, Owner: target}
			} else {
				packages[alias] = target
			}
			if gen.Lparen.IsValid() {
				edits = append(edits, lower.Edit{Start: offset(spec.Pos()), End: offset(spec.End()), New: ""})
			}
		}
		if javaSpecs > 0 && javaSpecs == len(gen.Specs) && !gen.Lparen.IsValid() {
			edits = append(edits, lower.Edit{Start: offset(gen.Pos()), End: offset(gen.End()), New: ""})
		}
	}
	if len(direct) == 0 && len(packages) == 0 {
		return src, false, nil
	}

	selectorTypes := map[*ast.SelectorExpr]*importedType{}
	selectorModels := map[string]*importedType{}
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		javaPackage, ok := packages[ident.Name]
		if !ok {
			return true
		}
		stubName := "__goplusJava_" + ident.Name + "_" + selector.Sel.Name
		stub := selectorModels[stubName]
		if stub == nil {
			stub = &importedType{Name: stubName, Owner: javaPackage + "." + selector.Sel.Name}
			selectorModels[stubName] = stub
		}
		selectorTypes[selector] = stub
		return true
	})
	for selector, stub := range selectorTypes {
		stub.TParams = max(stub.TParams, typeArgumentCount(file, selector))
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.IndexExpr:
			if ident, ok := value.X.(*ast.Ident); ok && direct[ident.Name] != nil {
				direct[ident.Name].TParams = max(direct[ident.Name].TParams, 1)
			}
		case *ast.IndexListExpr:
			if ident, ok := value.X.(*ast.Ident); ok && direct[ident.Name] != nil {
				direct[ident.Name].TParams = max(direct[ident.Name].TParams, len(value.Indices))
			}
		}
		return true
	})

	javaTypes := map[string]*importedType{}
	for name, model := range direct {
		javaTypes[name] = model
	}
	for _, model := range selectorTypes {
		javaTypes[model.Name] = model
	}
	var rewrittenCalls map[*ast.CallExpr]bool = map[*ast.CallExpr]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "new" {
			return true
		}
		typeText, model, ok := javaTypeText(src, offset, call.Args[0], direct, selectorTypes)
		if !ok {
			return true
		}
		var args []string
		for _, arg := range call.Args[1:] {
			args = append(args, string(src[offset(arg.Pos()):offset(arg.End())]))
		}
		text := "__goplusJavaNew[" + typeText + "](" + strings.Join(args, ", ") + ")"
		edits = append(edits, lower.Edit{Start: offset(call.Pos()), End: offset(call.End()), New: text})
		rewrittenCalls[call] = true
		javaTypes[model.Name] = model
		return false
	})
	// Selector edits not covered by a whole constructor rewrite remain valid
	// type uses (fields, parameters, variables).
	for selector, model := range selectorTypes {
		covered := false
		for call := range rewrittenCalls {
			if selector.Pos() >= call.Pos() && selector.End() <= call.End() {
				covered = true
				break
			}
		}
		if !covered {
			edits = append(edits, lower.Edit{Start: offset(selector.Pos()), End: offset(selector.End()), New: model.Name})
		}
	}

	var models []*importedType
	for _, model := range javaTypes {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	var appendix strings.Builder
	appendix.WriteString("\n\n// Synthetic Go shadow for Java imports.\n")
	seen := map[string]bool{}
	for _, model := range models {
		if seen[model.Name] {
			continue
		}
		seen[model.Name] = true
		fmt.Fprintf(&appendix, "//goplus:java-type %s %s\n", model.Name, model.Owner)
		fmt.Fprintf(&appendix, "type %s", model.Name)
		if model.TParams > 0 {
			var params []string
			for i := 0; i < model.TParams; i++ {
				params = append(params, fmt.Sprintf("J%d any", i))
			}
			fmt.Fprintf(&appendix, "[%s]", strings.Join(params, ", "))
		}
		appendix.WriteString(" struct{}\n")
	}
	if len(rewrittenCalls) > 0 {
		appendix.WriteString("func __goplusJavaNew[T any](args ...any) T { var zero T; return zero }\n")
	}
	edits = append(edits, lower.Edit{Start: len(src), End: len(src), New: appendix.String()})
	out, err := lower.Apply(src, edits)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func javaTypeText(src []byte, offset func(token.Pos) int, expr ast.Expr, direct map[string]*importedType, selectors map[*ast.SelectorExpr]*importedType) (string, *importedType, bool) {
	switch value := expr.(type) {
	case *ast.Ident:
		model, ok := direct[value.Name]
		return value.Name, model, ok
	case *ast.SelectorExpr:
		model, ok := selectors[value]
		if !ok {
			return "", nil, false
		}
		return model.Name, model, true
	case *ast.IndexExpr:
		base, model, ok := javaTypeText(src, offset, value.X, direct, selectors)
		if !ok {
			return "", nil, false
		}
		return base + string(src[offset(value.Lbrack):offset(value.Rbrack)+1]), model, true
	case *ast.IndexListExpr:
		base, model, ok := javaTypeText(src, offset, value.X, direct, selectors)
		if !ok {
			return "", nil, false
		}
		return base + string(src[offset(value.Lbrack):offset(value.Rbrack)+1]), model, true
	default:
		return "", nil, false
	}
}

func typeArgumentCount(file *ast.File, target *ast.SelectorExpr) int {
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.IndexExpr:
			if value.X == target {
				count = max(count, 1)
			}
		case *ast.IndexListExpr:
			if value.X == target {
				count = max(count, len(value.Indices))
			}
		}
		return true
	})
	return count
}
