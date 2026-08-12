package resolve

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestTerminatingMatchWithFollowingStatementNeedsNoSyntheticPanic(t *testing.T) {
	const source = `package sample
func choose(value any) int {
	switch value.(type) {
	case string:
		return 1
	case int:
		return 2
	}
	return 3
}`
	file, err := parser.ParseFile(token.NewFileSet(), "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var match *ast.TypeSwitchStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if statement, ok := node.(*ast.TypeSwitchStmt); ok {
			match = statement
			return false
		}
		return true
	})
	if match == nil {
		t.Fatal("type switch not found")
	}
	resolver := &fileResolver{file: file}
	if !matchArmsTerminate(match) {
		t.Fatal("returning match arms were not recognized as terminating")
	}
	if !resolver.matchHasFollowingStatement(match) {
		t.Fatal("following return was not detected")
	}
}

func TestNonTerminatingMatchStillRequiresBackstop(t *testing.T) {
	const source = `package sample
func choose(value any) int {
	switch value.(type) {
	case string:
		println("string")
	}
	return 3
}`
	file, err := parser.ParseFile(token.NewFileSet(), "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var match *ast.TypeSwitchStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if statement, ok := node.(*ast.TypeSwitchStmt); ok {
			match = statement
			return false
		}
		return true
	})
	if match == nil || matchArmsTerminate(match) {
		t.Fatal("non-terminating match was accepted")
	}
}
