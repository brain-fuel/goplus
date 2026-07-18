package syntax

import (
	"go/ast"
	"go/token"
	"reflect"
)

var tokenPosType = reflect.TypeOf(token.Pos(0))

// rewriteNodePositions rewrites every token.Pos-typed struct field of the
// node itself (not its children — ast.Inspect visits those separately).
// Reflection keeps this exhaustive over all node kinds that can appear in a
// type parameter list, including every constraint expression form.
func rewriteNodePositions(node ast.Node, fn func(token.Pos) token.Pos) {
	v := reflect.ValueOf(node)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type() == tokenPosType && f.CanSet() {
			if p := token.Pos(f.Int()); p.IsValid() {
				f.SetInt(int64(fn(p)))
			}
		}
	}
}
