package resolve

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestBlankUnusedImportsUsesDeclaredPackageName(t *testing.T) {
	const source = `package sample
import "go.yaml.in/yaml/v3"
var _ = yaml.Node{}
`
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Implicits: make(map[ast.Node]types.Object)}
	info.Implicits[file.Imports[0]] = types.NewPkgName(
		file.Imports[0].Pos(),
		nil,
		"yaml",
		types.NewPackage("go.yaml.in/yaml/v3", "yaml"),
	)
	if edits := blankUnusedImports(file, files, info, []byte(source)); len(edits) != 0 {
		t.Fatalf("used import was blanked: %#v", edits)
	}
}

func TestBlankUnusedImportsStillBlanksOrphan(t *testing.T) {
	const source = `package sample
import "go.yaml.in/yaml/v3"
`
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Implicits: make(map[ast.Node]types.Object)}
	info.Implicits[file.Imports[0]] = types.NewPkgName(
		file.Imports[0].Pos(),
		nil,
		"yaml",
		types.NewPackage("go.yaml.in/yaml/v3", "yaml"),
	)
	if edits := blankUnusedImports(file, files, info, []byte(source)); len(edits) != 1 || edits[0].New != "_ " {
		t.Fatalf("orphan import edits = %#v", edits)
	}
}
