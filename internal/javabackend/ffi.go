package javabackend

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"
	"strings"
	"unicode"
)

type javaBinding struct {
	Kind       string
	Owner      string
	Member     string
	Descriptor string
	Null       string
	Throws     string
	String     string
}

type descriptorType struct {
	Java string
	Void bool
}

func parseJavaBinding(decl *ast.FuncDecl) (*javaBinding, error) {
	if decl.Doc == nil {
		return nil, nil
	}
	var raw string
	for _, comment := range decl.Doc.List {
		text := strings.TrimSpace(comment.Text)
		if rest, ok := strings.CutPrefix(text, "//goplus:java"); ok {
			if raw != "" {
				return nil, fmt.Errorf("function has more than one //goplus:java directive")
			}
			raw = strings.TrimSpace(rest)
		}
	}
	if raw == "" {
		return nil, nil
	}
	values := map[string]string{}
	for _, field := range strings.Fields(raw) {
		key, value, ok := strings.Cut(field, "=")
		if !ok || key == "" || value == "" {
			return nil, fmt.Errorf("invalid //goplus:java field %q", field)
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf("duplicate //goplus:java field %q", key)
		}
		if strings.HasPrefix(value, `"`) {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			value = unquoted
		}
		values[key] = value
	}
	for key := range values {
		switch key {
		case "kind", "owner", "member", "descriptor", "null", "throws", "string":
		default:
			return nil, fmt.Errorf("unknown //goplus:java field %q", key)
		}
	}
	binding := &javaBinding{
		Kind: values["kind"], Owner: values["owner"], Member: values["member"],
		Descriptor: values["descriptor"], Null: values["null"], Throws: values["throws"],
		String: values["string"],
	}
	if binding.Null == "" {
		binding.Null = "nullable"
	}
	if binding.Throws == "" {
		binding.Throws = "none"
	}
	if binding.String == "" {
		binding.String = "go"
	}
	if binding.Kind != "static" && binding.Kind != "virtual" && binding.Kind != "constructor" {
		return nil, fmt.Errorf("kind must be static, virtual, or constructor")
	}
	if !javaQualifiedName(binding.Owner) {
		return nil, fmt.Errorf("invalid Java owner %q", binding.Owner)
	}
	if binding.Kind != "constructor" && !javaMemberName(binding.Member) {
		return nil, fmt.Errorf("invalid Java member %q", binding.Member)
	}
	if binding.Descriptor == "" {
		return nil, fmt.Errorf("descriptor is required")
	}
	if binding.Null != "nonnull" && binding.Null != "nullable" {
		return nil, fmt.Errorf("null must be nonnull or nullable")
	}
	if binding.Throws != "none" && binding.Throws != "panic" && binding.Throws != "result" {
		return nil, fmt.Errorf("throws must be none, panic, or result")
	}
	if binding.String != "go" && binding.String != "java" {
		return nil, fmt.Errorf("string must be go or java")
	}
	return binding, nil
}

func (e *emitter) emitJavaBinding(w *javaWriter, decl *ast.FuncDecl, sig *types.Signature, name string, binding *javaBinding) {
	if decl.Body != nil {
		e.unsupported(decl, "//goplus:java function %s must be bodyless", decl.Name.Name)
		return
	}
	params, result, err := parseMethodDescriptor(binding.Descriptor)
	if err != nil {
		e.unsupported(decl, "invalid //goplus:java descriptor: %v", err)
		return
	}
	if binding.Kind == "constructor" {
		if !result.Void {
			e.unsupported(decl, "Java constructor descriptor must return void")
			return
		}
		result = descriptorType{Java: binding.Owner}
	}
	goOffset := 0
	if binding.Kind == "virtual" {
		goOffset = 1
	}
	if sig.Params().Len() != len(params)+goOffset {
		e.unsupported(decl, "Java descriptor has %d parameters but Go binding has %d", len(params), sig.Params().Len())
		return
	}
	if binding.Throws == "result" {
		if sig.Results().Len() != 2 {
			e.unsupported(decl, "throws=result requires a two-result Go binding")
			return
		}
		if !isBuiltinError(sig.Results().At(1).Type()) {
			e.unsupported(decl, "throws=result requires error as the second Go result")
			return
		}
	} else if (result.Void && sig.Results().Len() != 0) || (!result.Void && sig.Results().Len() != 1) {
		e.unsupported(decl, "Java descriptor result does not match Go binding result count")
		return
	}
	if !result.Void && result.Java == "java.lang.String" && binding.Null == "nullable" &&
		sig.Results().Len() > 0 && isStringType(sig.Results().At(0).Type()) {
		e.unsupported(decl, "nullable java.lang.String cannot bind to Go string; use null=nonnull or an explicit nullable result type")
		return
	}
	w.line("%s {", e.signature(name, sig, true, ast.IsExported(decl.Name.Name)))
	w.indent++
	var args []string
	for i, param := range params {
		goParam := sig.Params().At(i + goOffset)
		args = append(args, e.javaArgument(e.objectName(goParam), goParam.Type(), param, binding))
	}
	var invocation string
	switch binding.Kind {
	case "static":
		invocation = binding.Owner + "." + binding.Member + "(" + strings.Join(args, ", ") + ")"
	case "constructor":
		invocation = "new " + binding.Owner + "(" + strings.Join(args, ", ") + ")"
	case "virtual":
		receiver := e.objectName(sig.Params().At(0))
		if isStringType(sig.Params().At(0).Type()) && binding.String == "java" {
			receiver += ".toJava()"
		} else {
			receiver = "((" + binding.Owner + ") " + receiver + ")"
		}
		invocation = receiver + "." + binding.Member + "(" + strings.Join(args, ", ") + ")"
	}
	success := e.javaResult(invocation, result, sig, binding)
	if binding.Throws == "none" {
		w.line("%s", success)
	} else {
		w.line("try {")
		w.indent++
		w.line("%s", success)
		w.indent--
		w.line("} catch (Exception __error) {")
		w.indent++
		if binding.Throws == "panic" {
			w.line("throw GpRuntime.panicValue(__error);")
		} else {
			w.line("return new GpTuple2<>(%s, GpRuntime.javaError(__error));", e.zeroValue(sig.Results().At(0).Type()))
		}
		w.indent--
		w.line("}")
	}
	w.indent--
	w.line("}")
}

func (e *emitter) javaArgument(name string, goType types.Type, descriptor descriptorType, binding *javaBinding) string {
	if descriptor.Java == "java.lang.String" && isStringType(goType) {
		if binding.String != "java" {
			e.unsupported(nil, "GpString crossing java.lang.String requires string=java")
			return name
		}
		return name + ".toJava()"
	}
	goJava := e.javaType(goType, false)
	if goJava == descriptor.Java {
		return name
	}
	return "((" + descriptor.Java + ") (" + name + "))"
}

func (e *emitter) javaResult(invocation string, descriptor descriptorType, sig *types.Signature, binding *javaBinding) string {
	if descriptor.Void {
		if binding.Throws == "result" {
			return invocation + "; return new GpTuple2<>(" + e.zeroValue(sig.Results().At(0).Type()) + ", null);"
		}
		return invocation + ";"
	}
	expr := invocation
	goResult := sig.Results().At(0).Type()
	if descriptor.Java == "java.lang.String" && isStringType(goResult) {
		if binding.String != "java" {
			e.unsupported(nil, "java.lang.String result requires string=java")
		} else if binding.Null == "nonnull" {
			expr = "GpString.fromJava(GpRuntime.requireNonNull(" + expr + ", \"" + binding.Owner + "." + binding.Member + "\"))"
		}
	} else if binding.Null == "nonnull" && !descriptorPrimitive(descriptor.Java) {
		expr = "GpRuntime.requireNonNull(" + expr + ", \"" + binding.Owner + "." + binding.Member + "\")"
	} else if descriptorPrimitive(descriptor.Java) {
		expr = e.normalizeNumeric(expr, goResult)
	}
	if binding.Throws == "result" {
		return "return new GpTuple2<>(" + expr + ", null);"
	}
	return "return " + expr + ";"
}

func parseMethodDescriptor(text string) ([]descriptorType, descriptorType, error) {
	if len(text) < 3 || text[0] != '(' {
		return nil, descriptorType{}, fmt.Errorf("method descriptor must start with (")
	}
	i := 1
	var params []descriptorType
	for i < len(text) && text[i] != ')' {
		t, next, err := parseDescriptorType(text, i, false)
		if err != nil {
			return nil, descriptorType{}, err
		}
		params, i = append(params, t), next
	}
	if i >= len(text) || text[i] != ')' {
		return nil, descriptorType{}, fmt.Errorf("missing )")
	}
	result, next, err := parseDescriptorType(text, i+1, true)
	if err != nil {
		return nil, descriptorType{}, err
	}
	if next != len(text) {
		return nil, descriptorType{}, fmt.Errorf("trailing descriptor text")
	}
	return params, result, nil
}

func parseDescriptorType(text string, at int, allowVoid bool) (descriptorType, int, error) {
	if at >= len(text) {
		return descriptorType{}, at, fmt.Errorf("missing descriptor type")
	}
	primitive := map[byte]string{'B': "byte", 'C': "char", 'D': "double", 'F': "float", 'I': "int", 'J': "long", 'S': "short", 'Z': "boolean"}
	if java := primitive[text[at]]; java != "" {
		return descriptorType{Java: java}, at + 1, nil
	}
	if text[at] == 'V' && allowVoid {
		return descriptorType{Java: "void", Void: true}, at + 1, nil
	}
	if text[at] == 'L' {
		end := strings.IndexByte(text[at:], ';')
		if end < 0 {
			return descriptorType{}, at, fmt.Errorf("unterminated object descriptor")
		}
		end += at
		name := strings.ReplaceAll(text[at+1:end], "/", ".")
		if !javaQualifiedName(name) {
			return descriptorType{}, at, fmt.Errorf("invalid object descriptor")
		}
		return descriptorType{Java: name}, end + 1, nil
	}
	return descriptorType{}, at, fmt.Errorf("unsupported descriptor type %q", text[at])
}

func javaQualifiedName(name string) bool {
	if name == "" {
		return false
	}
	for _, part := range strings.Split(name, ".") {
		if !javaMemberName(part) {
			return false
		}
	}
	return true
}

func javaMemberName(name string) bool {
	for i, r := range name {
		if !(r == '_' || r == '$' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r))) {
			return false
		}
	}
	return name != ""
}

func descriptorPrimitive(name string) bool {
	switch name {
	case "byte", "char", "double", "float", "int", "long", "short", "boolean":
		return true
	}
	return false
}
