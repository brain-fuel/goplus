// Package syntax parses .gpp source. G++ v0.1.0 is Go grammar plus type
// parameters on method declarations, which stock go/parser fully parses for
// error recovery but then discards (setting TypeParams to nil) alongside a
// "method must have no type parameters" error. This package filters exactly
// those errors and recovers the discarded type parameter lists from source,
// grafting them back onto the AST with correct positions.
//
// This package is the designated fork boundary: when later milestones extend
// the grammar beyond what go/parser tolerates, a real parser replaces the
// recovery mechanism behind the same API.
package syntax

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"

	"goforge.dev/gpp/internal/directive"
)

// methodTParamErr is the exact go/parser message emitted at the '[' of a
// method's type parameter list (go/parser/parser.go, parseFuncDecl).
const methodTParamErr = "method must have no type parameters"

// File is one parsed .gpp file.
type File struct {
	Path    string
	Src     []byte
	Fset    *token.FileSet
	TokFile *token.File
	AST     *ast.File
	Methods []*GenericMethod
}

// GenericMethod is a method declaration carrying its own type parameters.
type GenericMethod struct {
	Decl *ast.FuncDecl // TypeParams grafted back in

	RecvName     string   // receiver identifier as written; "" if absent
	RecvTypeName string   // base named type, e.g. "Stack"
	RecvPointer  bool     // *Stack[T] receiver
	RecvTParams  []string // receiver type parameter names, e.g. ["T"]

	// LBrack and RBrack are byte offsets in Src of the method's type
	// parameter brackets (the recovered '[' and ']').
	LBrack, RBrack int

	NameOverride string // //gpp:name value, "" if absent
}

// Offset converts a token.Pos within f to a byte offset in f.Src.
func (f *File) Offset(pos token.Pos) int { return f.TokFile.Offset(pos) }

// ParseFile parses .gpp source. Genuine syntax errors (anything other than
// the filtered method-type-parameter errors) are returned as a
// scanner.ErrorList.
func ParseFile(fset *token.FileSet, path string, src []byte) (*File, error) {
	astFile, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if astFile == nil {
		return nil, parseErr
	}
	f := &File{
		Path:    path,
		Src:     src,
		Fset:    fset,
		TokFile: fset.File(astFile.Pos()),
		AST:     astFile,
	}

	// Candidate generic methods: method decls whose name is immediately
	// followed by '[' in the source (the parser discarded that list).
	candidates := map[int]*ast.FuncDecl{} // offset of '[' -> decl
	for _, decl := range astFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Type.TypeParams != nil {
			continue
		}
		off := f.Offset(fd.Name.End())
		if off < len(src) && src[off] == '[' {
			candidates[off] = fd
		}
	}

	// Filter the discard errors that match a candidate; keep the rest.
	var genuine scanner.ErrorList
	if parseErr != nil {
		list, ok := parseErr.(scanner.ErrorList)
		if !ok {
			return nil, parseErr
		}
		for _, e := range list {
			if e.Msg == methodTParamErr {
				if _, isCandidate := candidates[e.Pos.Offset]; isCandidate {
					continue
				}
			}
			genuine = append(genuine, e)
		}
	}
	if len(genuine) > 0 {
		return nil, genuine
	}

	for off, fd := range candidates {
		m, err := f.recoverMethod(fd, off)
		if err != nil {
			return nil, err
		}
		f.Methods = append(f.Methods, m)
	}
	// Deterministic order (candidates is a map).
	sortMethods(f.Methods)
	return f, nil
}

// recoverMethod re-scans the discarded type parameter list starting at the
// '[' at lbrack, re-parses it, and grafts it onto fd with original positions.
func (f *File) recoverMethod(fd *ast.FuncDecl, lbrack int) (*GenericMethod, error) {
	rbrack, err := matchBracket(f.Src, lbrack)
	if err != nil {
		return nil, fmt.Errorf("%s: recovering method type parameters of %s: %w", f.Path, fd.Name.Name, err)
	}
	segment := string(f.Src[lbrack : rbrack+1])

	const synthHeader = "package p\nfunc _"
	synthSrc := synthHeader + segment + "()"
	synthFset := token.NewFileSet()
	synthFile, err := parser.ParseFile(synthFset, "synthetic.go", []byte(synthSrc), parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%s: re-parsing type parameters of %s: %w", f.Path, fd.Name.Name, err)
	}
	synthDecl, ok := synthFile.Decls[0].(*ast.FuncDecl)
	if !ok || synthDecl.Type.TypeParams == nil {
		return nil, fmt.Errorf("%s: internal error: synthetic parse of %s produced no type parameters", f.Path, fd.Name.Name)
	}
	tparams := synthDecl.Type.TypeParams

	// Rewrite every position in the recovered subtree from synthetic-file
	// offsets to original-file offsets.
	synthTok := synthFset.File(synthFile.Pos())
	delta := lbrack - len(synthHeader)
	rewritePositions(tparams, func(p token.Pos) token.Pos {
		return f.TokFile.Pos(synthTok.Offset(p) + delta)
	})
	fd.Type.TypeParams = tparams

	m := &GenericMethod{Decl: fd, LBrack: lbrack, RBrack: rbrack}
	if err := m.fillReceiver(); err != nil {
		return nil, fmt.Errorf("%s: %w", f.Path, err)
	}
	if name, ok := directive.Name(fd.Doc); ok {
		m.NameOverride = name
	}
	return m, nil
}

// fillReceiver extracts receiver shape from the declaration.
func (m *GenericMethod) fillReceiver() error {
	fd := m.Decl
	if len(fd.Recv.List) != 1 {
		return fmt.Errorf("method %s: malformed receiver", fd.Name.Name)
	}
	field := fd.Recv.List[0]
	if len(field.Names) == 1 {
		m.RecvName = field.Names[0].Name
	}
	t := field.Type
	if p, ok := t.(*ast.ParenExpr); ok {
		t = p.X
	}
	if s, ok := t.(*ast.StarExpr); ok {
		m.RecvPointer = true
		t = s.X
	}
	if p, ok := t.(*ast.ParenExpr); ok {
		t = p.X
	}
	switch bt := t.(type) {
	case *ast.Ident:
		m.RecvTypeName = bt.Name
	case *ast.IndexExpr:
		id, ok := bt.X.(*ast.Ident)
		if !ok {
			return fmt.Errorf("method %s: unsupported receiver type", fd.Name.Name)
		}
		m.RecvTypeName = id.Name
		tp, ok := bt.Index.(*ast.Ident)
		if !ok {
			return fmt.Errorf("method %s: receiver type parameter must be an identifier", fd.Name.Name)
		}
		m.RecvTParams = []string{tp.Name}
	case *ast.IndexListExpr:
		id, ok := bt.X.(*ast.Ident)
		if !ok {
			return fmt.Errorf("method %s: unsupported receiver type", fd.Name.Name)
		}
		m.RecvTypeName = id.Name
		for _, idx := range bt.Indices {
			tp, ok := idx.(*ast.Ident)
			if !ok {
				return fmt.Errorf("method %s: receiver type parameter must be an identifier", fd.Name.Name)
			}
			m.RecvTParams = append(m.RecvTParams, tp.Name)
		}
	default:
		return fmt.Errorf("method %s: unsupported receiver type", fd.Name.Name)
	}
	return nil
}

// matchBracket returns the offset of the ']' matching the '[' at lbrack,
// scanning tokens (so brackets inside strings, comments, and nested
// constraint types are handled correctly).
func matchBracket(src []byte, lbrack int) (int, error) {
	sub := src[lbrack:]
	fset := token.NewFileSet()
	file := fset.AddFile("scan", -1, len(sub))
	var s scanner.Scanner
	s.Init(file, sub, nil, 0)
	depth := 0
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			return 0, fmt.Errorf("unbalanced '[' at offset %d", lbrack)
		}
		switch tok {
		case token.LBRACK:
			depth++
		case token.RBRACK:
			depth--
			if depth == 0 {
				return lbrack + file.Offset(pos), nil
			}
		}
	}
}

// rewritePositions applies fn to every token.Pos field reachable from n.
func rewritePositions(n ast.Node, fn func(token.Pos) token.Pos) {
	ast.Inspect(n, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		rewriteNodePositions(node, fn)
		return true
	})
}

func sortMethods(ms []*GenericMethod) {
	for i := 1; i < len(ms); i++ {
		for j := i; j > 0 && ms[j].LBrack < ms[j-1].LBrack; j-- {
			ms[j], ms[j-1] = ms[j-1], ms[j]
		}
	}
}
