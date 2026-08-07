package javabackend

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

func (e *emitter) javaType(t types.Type, boxed bool) string {
	if t == nil {
		return "Object"
	}
	t = types.Unalias(t)
	switch value := t.(type) {
	case *types.Basic:
		if value.Info()&types.IsComplex != 0 {
			e.unsupported(nil, "complex numbers are not in the portable Java subset")
			return "Object"
		}
		if value.Kind() == types.UnsafePointer {
			e.unsupported(nil, "unsafe.Pointer is not in the portable Java subset")
			return "Object"
		}
		return basicJavaType(value, boxed)
	case *types.Named:
		if isBuiltinError(value) {
			return "GpError"
		}
		if value.Obj().Pkg() != nil && value.Obj().Pkg().Path() == "testing" && (value.Obj().Name() == "T" || value.Obj().Name() == "TB") {
			return "GpTest"
		}
		if value.Obj().Pkg() != nil && value.Obj().Pkg().Path() == e.pkg.PkgPath {
			if owner := e.javaTypes[value.Obj().Name()]; owner != "" {
				if value.TypeArgs() != nil && value.TypeArgs().Len() > 0 {
					var args []string
					for i := 0; i < value.TypeArgs().Len(); i++ {
						args = append(args, e.javaType(value.TypeArgs().At(i), true))
					}
					return owner + "<" + strings.Join(args, ", ") + ">"
				}
				return owner
			}
		}
		under := types.Unalias(value.Underlying())
		switch under.(type) {
		case *types.Basic, *types.Slice, *types.Array, *types.Map, *types.Chan, *types.Pointer, *types.Signature:
			return e.javaType(under, boxed)
		}
		if pkg := value.Obj().Pkg(); pkg != nil && pkg.Path() != e.pkg.PkgPath &&
			pkg.Path() != e.mapper.modulePath && !strings.HasPrefix(pkg.Path(), e.mapper.modulePath+"/") {
			e.unsupported(nil, "Go type %s needs a Java-target adapter", types.TypeString(value, nil))
			return "Object"
		}
		name := javaIdent(value.Obj().Name(), ast.IsExported(value.Obj().Name()))
		if pkg := value.Obj().Pkg(); pkg != nil && pkg.Path() != e.pkg.PkgPath {
			name = e.mapper.javaPackage(pkg.Path()) + "." + name
		}
		if value.TypeArgs() != nil && value.TypeArgs().Len() > 0 {
			var args []string
			for i := 0; i < value.TypeArgs().Len(); i++ {
				args = append(args, e.javaType(value.TypeArgs().At(i), true))
			}
			name += "<" + strings.Join(args, ", ") + ">"
		}
		return name
	case *types.TypeParam:
		return javaIdent(value.Obj().Name(), true)
	case *types.Pointer:
		elem := types.Unalias(value.Elem())
		if named, ok := elem.(*types.Named); ok {
			switch named.Underlying().(type) {
			case *types.Struct, *types.Interface:
				return e.javaType(named, true)
			}
		}
		return "GpRef<" + e.javaType(value.Elem(), true) + ">"
	case *types.Slice:
		return "GpSlice<" + e.javaType(value.Elem(), true) + ">"
	case *types.Array:
		return "GpSlice<" + e.javaType(value.Elem(), true) + ">"
	case *types.Map:
		return "GpMap<" + e.javaType(value.Key(), true) + ", " + e.javaType(value.Elem(), true) + ">"
	case *types.Chan:
		return "GpChan<" + e.javaType(value.Elem(), true) + ">"
	case *types.Signature:
		return e.functionType(value)
	case *types.Interface:
		if value.NumExplicitMethods() == 0 && value.NumEmbeddeds() == 0 {
			return "Object"
		}
		e.unsupported(nil, "anonymous non-empty interfaces are not in the portable Java subset")
		return "Object"
	case *types.Tuple:
		return e.tupleType(value)
	case *types.Struct:
		e.unsupported(nil, "anonymous struct types are not in the portable Java subset")
		return "Object"
	default:
		e.unsupported(nil, "Go type %s is not in the portable Java subset", types.TypeString(t, nil))
		return "Object"
	}
}

func basicJavaType(basic *types.Basic, boxed bool) string {
	kind := basic.Kind()
	if basic.Info()&types.IsUntyped != 0 {
		switch kind {
		case types.UntypedBool:
			kind = types.Bool
		case types.UntypedInt, types.UntypedRune:
			kind = types.Int
		case types.UntypedFloat:
			kind = types.Float64
		case types.UntypedString:
			kind = types.String
		case types.UntypedNil:
			return "Object"
		}
	}
	switch kind {
	case types.Bool:
		if boxed {
			return "Boolean"
		}
		return "boolean"
	case types.Int8:
		if boxed {
			return "Byte"
		}
		return "byte"
	case types.Int16:
		if boxed {
			return "Short"
		}
		return "short"
	case types.Int32:
		if boxed {
			return "Integer"
		}
		return "int"
	case types.Int, types.Int64, types.Uint, types.Uint64, types.Uintptr:
		if boxed {
			return "Long"
		}
		return "long"
	case types.Uint8:
		if boxed {
			return "Integer"
		}
		return "int"
	case types.Uint16:
		if boxed {
			return "Integer"
		}
		return "int"
	case types.Uint32:
		if boxed {
			return "Long"
		}
		return "long"
	case types.Float32:
		if boxed {
			return "Float"
		}
		return "float"
	case types.Float64:
		if boxed {
			return "Double"
		}
		return "double"
	case types.String:
		return "GpString"
	case types.UnsafePointer:
		return "java.lang.foreign.MemorySegment"
	default:
		return "Object"
	}
}

func (e *emitter) functionType(sig *types.Signature) string {
	var args []string
	for i := 0; i < sig.Params().Len(); i++ {
		args = append(args, e.javaType(sig.Params().At(i).Type(), true))
	}
	result := "Void"
	if sig.Results().Len() == 1 {
		result = e.javaType(sig.Results().At(0).Type(), true)
	} else if sig.Results().Len() > 1 {
		result = e.tupleType(sig.Results())
	}
	args = append(args, result)
	if len(args) > 4 {
		e.unsupported(nil, "function values with more than three parameters are not yet portable to Java")
		return "Object"
	}
	return fmt.Sprintf("GpFn%d<%s>", len(args)-1, strings.Join(args, ", "))
}

func (e *emitter) tupleType(tuple *types.Tuple) string {
	if tuple.Len() < 2 || tuple.Len() > 3 {
		e.unsupported(nil, "Java tuples currently support two or three results, got %d", tuple.Len())
		return "Object"
	}
	var parts []string
	for i := 0; i < tuple.Len(); i++ {
		parts = append(parts, e.javaType(tuple.At(i).Type(), true))
	}
	return fmt.Sprintf("GpTuple%d<%s>", tuple.Len(), strings.Join(parts, ", "))
}

func (e *emitter) zeroValue(t types.Type) string {
	if t == nil {
		return "null"
	}
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		if isBuiltinError(named) {
			return "null"
		}
		if named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == e.pkg.PkgPath && e.javaTypes[named.Obj().Name()] != "" {
			return "null"
		}
		switch named.Underlying().(type) {
		case *types.Basic, *types.Slice, *types.Array, *types.Map, *types.Chan, *types.Pointer, *types.Signature:
			return e.zeroValue(named.Underlying())
		case *types.Interface:
			return "null"
		default:
			return "new " + e.javaType(named, false) + "()"
		}
	}
	switch value := t.(type) {
	case *types.Basic:
		switch {
		case value.Info()&types.IsBoolean != 0:
			return "false"
		case value.Info()&types.IsString != 0:
			return "GpString.EMPTY"
		case value.Info()&types.IsNumeric != 0:
			switch value.Kind() {
			case types.Int8:
				return "GpRuntime.int8(0)"
			case types.Int16:
				return "GpRuntime.int16(0)"
			case types.Int32:
				return "GpRuntime.int32(0)"
			case types.Uint8, types.Uint16:
				return "0"
			case types.Float32:
				return "0.0f"
			}
			if value.Info()&types.IsFloat != 0 {
				return "0.0"
			}
			return "0L"
		default:
			return "null"
		}
	case *types.Slice:
		return "GpSlice.nil(() -> " + e.zeroValue(value.Elem()) + ", " + e.elementCopier(value.Elem()) + ")"
	case *types.Array:
		return fmt.Sprintf("GpSlice.makeArray(%d, () -> %s, %s)", value.Len(), e.zeroValue(value.Elem()), e.elementCopier(value.Elem()))
	case *types.Map:
		return "GpMap.nil(() -> " + e.zeroValue(value.Elem()) + ")"
	case *types.Chan:
		return "GpChan.nil()"
	case *types.Pointer, *types.Signature, *types.Interface:
		return "null"
	case *types.TypeParam:
		return "null"
	default:
		return "null"
	}
}

func isUnsigned(t types.Type) bool {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	basic, ok := t.(*types.Basic)
	if !ok {
		return false
	}
	switch basic.Kind() {
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return true
	default:
		return false
	}
}

func basicKind(t types.Type) (types.BasicKind, bool) {
	t = underlyingType(t)
	basic, ok := t.(*types.Basic)
	if !ok {
		return types.Invalid, false
	}
	return basic.Kind(), true
}

func underlyingType(t types.Type) types.Type {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		return types.Unalias(named.Underlying())
	}
	return t
}

func elementType(t types.Type) types.Type {
	switch value := underlyingType(t).(type) {
	case *types.Slice:
		return value.Elem()
	case *types.Array:
		return value.Elem()
	case *types.Chan:
		return value.Elem()
	default:
		return types.Typ[types.Invalid]
	}
}

func sameBasicKind(left, right types.Type) bool {
	a, ok := basicKind(left)
	if !ok {
		return false
	}
	b, ok := basicKind(right)
	return ok && a == b
}

func (e *emitter) normalizeNumeric(value string, t types.Type) string {
	kind, ok := basicKind(t)
	if !ok {
		return value
	}
	switch kind {
	case types.Int8:
		return "GpRuntime.int8(" + value + ")"
	case types.Int16:
		return "GpRuntime.int16(" + value + ")"
	case types.Int32:
		return "GpRuntime.int32(" + value + ")"
	case types.Uint8:
		return "GpRuntime.uint8(" + value + ")"
	case types.Uint16:
		return "GpRuntime.uint16(" + value + ")"
	case types.Uint32:
		return "GpRuntime.uint32(" + value + ")"
	case types.Float32:
		return "GpRuntime.float32(" + value + ")"
	default:
		return value
	}
}

func (e *emitter) copyValue(value string, t types.Type) string {
	if _, ok := basicKind(t); ok {
		return e.normalizeNumeric(value, t)
	}
	if _, pointer := types.Unalias(t).(*types.Pointer); pointer {
		return value
	}
	return "GpRuntime.copy(" + value + ")"
}

func (e *emitter) elementCopier(t types.Type) string {
	name := e.nextTemp("element")
	return name + " -> " + e.copyValue(name, t)
}

func isStringType(t types.Type) bool {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	basic, ok := t.(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

func isBasicType(t types.Type) bool {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	_, ok := t.(*types.Basic)
	return ok
}

func isBuiltinError(t types.Type) bool {
	t = types.Unalias(t)
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() == nil && named.Obj().Name() == "error"
}
