// Package goml implements the v0 goml front end: an ML-family surface
// (SML/OCaml/Idris2/Lean4 flavored) for the Go+ core. goml sources
// transpile to .gp text and generate through the ordinary goplus
// pipeline, emitting <file>_gml.go beside <file>.goml.
//
// The v0 subset is documented in spec/goml-design.md §7 (milestone M0
// plus the dependent-surface pieces the differential fixtures need).
package goml

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Kind classifies a lexical token.
type Kind int

// Token kinds. Keywords and multi-rune operators get their own kind;
// single-rune punctuation is spelled directly.
const (
	EOF Kind = iota
	IDENT
	INT
	FLOAT
	STRING
	COMMENT // a full "--" line comment (kept for doc attachment)

	// Keywords.
	KwModule
	KwImport
	KwOpen
	KwAs
	KwLet
	KwLetStar
	KwRec
	KwIn
	KwType
	KwWhere
	KwMatch
	KwWith
	KwIf
	KwThen
	KwElse
	KwFun
	KwClass
	KwInstance
	KwExtends
	KwLaw
	KwTotal
	KwDeriving
	KwNamespace
	KwEnd
	KwDo
	KwMut
	KwWhile
	KwFor
	KwReturn
	KwDefer
	KwGo
	KwSelect
	KwRecv
	KwSend
	KwDefault
	KwProp
	KwInterface
	KwImpossible

	// Operators and punctuation.
	Assign   // :=
	Eq       // =
	Colon    // :
	Question // ?
	Bar      // |
	Pipe     // |>
	Compose  // >>>
	Kleisli  // >=>
	Arrow    // ->
	FatArrow // =>
	LArrow   // <-
	LParen
	RParen
	LBrack
	RBrack
	LBrace
	RBrace
	Comma
	Semi
	Dot
	Plus
	Minus
	Star
	Slash
	Percent
	EqEq
	NotEq
	Lt
	LtEq
	Gt
	GtEq
	AndAnd
	OrOr
	Bang  // !
	Amp   // & (address-of)
	At    // @
	OpSym // a user-declarable operator symbol run, e.g. <+>
)

var keywords = map[string]Kind{
	"module": KwModule, "import": KwImport, "open": KwOpen, "as": KwAs,
	"let": KwLet, "rec": KwRec, "in": KwIn, "type": KwType,
	"where": KwWhere, "match": KwMatch, "with": KwWith, "if": KwIf,
	"then": KwThen, "else": KwElse, "fun": KwFun, "class": KwClass,
	"instance": KwInstance, "extends": KwExtends, "law": KwLaw,
	"total": KwTotal, "deriving": KwDeriving, "namespace": KwNamespace,
	"end": KwEnd, "do": KwDo, "mut": KwMut, "while": KwWhile,
	"for": KwFor, "return": KwReturn, "defer": KwDefer, "go": KwGo,
	"select": KwSelect, "recv": KwRecv, "send": KwSend,
	"default": KwDefault, "prop": KwProp, "interface": KwInterface,
	"impossible": KwImpossible,
}

// opSymAlphabet is the character set operator-symbol runs draw from.
// `:`, `?`, `@`, and `.` are deliberately excluded: they glue to holes,
// binders, witnesses, and selectors, and are never part of a user op.
var opSymAlphabet = map[rune]bool{
	'+': true, '-': true, '*': true, '/': true, '<': true, '>': true,
	'=': true, '|': true, '&': true, '^': true, '%': true, '!': true,
	'~': true, '$': true,
}

// fixedSymOps maps exact symbol runs to their fixed token kinds.
var fixedSymOps = map[string]Kind{
	">>>": Compose, ">=>": Kleisli, "==": EqEq, "!=": NotEq,
	"<=": LtEq, ">=": GtEq, "&&": AndAnd, "||": OrOr, "|>": Pipe,
	"->": Arrow, "=>": FatArrow, "<-": LArrow, "=": Eq, "|": Bar,
	"!": Bang, "&": Amp, "+": Plus, "-": Minus, "*": Star, "/": Slash,
	"%": Percent, "<": Lt, ">": Gt,
}

// Pos is a 1-based source position in the .goml file.
type Pos struct {
	Line, Col int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Token is one lexical token with its source position.
type Token struct {
	Kind Kind
	Text string
	Pos  Pos
	Adj  bool // glued to the previous token (no whitespace between)
}

// Error is a positioned goml front-end error.
type Error struct {
	Path string
	Pos  Pos
	Msg  string
}

func (e *Error) Error() string {
	if e.Pos.Line == 0 {
		return fmt.Sprintf("%s: %s", e.Path, e.Msg)
	}
	return fmt.Sprintf("%s:%s: %s", e.Path, e.Pos, e.Msg)
}

type lexer struct {
	path string
	src  string
	off  int
	line int
	col  int
}

func newLexer(path string, src []byte) *lexer {
	return &lexer{path: path, src: string(src), line: 1, col: 1}
}

func (l *lexer) errf(pos Pos, format string, args ...any) *Error {
	return &Error{Path: l.path, Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func (l *lexer) peekRune() (rune, int) {
	if l.off >= len(l.src) {
		return 0, 0
	}
	return utf8.DecodeRuneInString(l.src[l.off:])
}

func (l *lexer) advance(n int) {
	for i := 0; i < n; {
		r, w := utf8.DecodeRuneInString(l.src[l.off:])
		l.off += w
		i += w
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
}

func (l *lexer) pos() Pos { return Pos{Line: l.line, Col: l.col} }

// tokens lexes the whole file. COMMENT tokens carry "--" line comments
// (block comments are discarded); everything else is significant.
func (l *lexer) tokens() ([]Token, *Error) {
	var out []Token
	for {
		// Skip whitespace, remembering whether any separated this token
		// from the previous one (adjacency distinguishes xs[i] from a
		// list-literal argument f [1, 2]).
		skipped := l.off == 0
		for {
			r, w := l.peekRune()
			if w == 0 || !unicode.IsSpace(r) {
				break
			}
			skipped = true
			l.advance(w)
		}
		adj := !skipped
		start := l.pos()
		r, w := l.peekRune()
		if w == 0 {
			out = append(out, Token{Kind: EOF, Pos: start, Adj: adj})
			return out, nil
		}
		rest := l.src[l.off:]
		switch {
		case strings.HasPrefix(rest, "--"):
			end := strings.IndexByte(rest, '\n')
			if end < 0 {
				end = len(rest)
			}
			text := strings.TrimRight(rest[2:end], " \t")
			l.advance(end)
			out = append(out, Token{Kind: COMMENT, Text: text, Pos: start, Adj: adj})
			continue
		case strings.HasPrefix(rest, "(*"):
			depth := 1
			i := 2
			for depth > 0 {
				if i >= len(rest) {
					return nil, l.errf(start, "unterminated block comment")
				}
				switch {
				case strings.HasPrefix(rest[i:], "(*"):
					depth++
					i += 2
				case strings.HasPrefix(rest[i:], "*)"):
					depth--
					i += 2
				default:
					_, rw := utf8.DecodeRuneInString(rest[i:])
					i += rw
				}
			}
			l.advance(i)
			continue
		case r == '"':
			i := 1
			for {
				if i >= len(rest) {
					return nil, l.errf(start, "unterminated string literal")
				}
				if rest[i] == '\\' {
					i += 2
					continue
				}
				if rest[i] == '"' {
					i++
					break
				}
				_, rw := utf8.DecodeRuneInString(rest[i:])
				i += rw
			}
			l.advance(i)
			out = append(out, Token{Kind: STRING, Text: rest[:i], Pos: start, Adj: adj})
			continue
		case unicode.IsDigit(r):
			i := 0
			kind := INT
			for i < len(rest) && (isDigit(rest[i]) || rest[i] == '_') {
				i++
			}
			if i < len(rest) && rest[i] == '.' && i+1 < len(rest) && isDigit(rest[i+1]) {
				kind = FLOAT
				i++
				for i < len(rest) && isDigit(rest[i]) {
					i++
				}
			}
			l.advance(i)
			out = append(out, Token{Kind: kind, Text: rest[:i], Pos: start, Adj: adj})
			continue
		case unicode.IsLetter(r) || r == '_':
			i := 0
			for i < len(rest) {
				rr, rw := utf8.DecodeRuneInString(rest[i:])
				if !unicode.IsLetter(rr) && !unicode.IsDigit(rr) && rr != '_' {
					break
				}
				i += rw
			}
			word := rest[:i]
			l.advance(i)
			if word == "let" && strings.HasPrefix(l.src[l.off:], "*") {
				l.advance(1)
				out = append(out, Token{Kind: KwLetStar, Text: "let*", Pos: start, Adj: adj})
				continue
			}
			if k, ok := keywords[word]; ok {
				out = append(out, Token{Kind: k, Text: word, Pos: start, Adj: adj})
			} else {
				out = append(out, Token{Kind: IDENT, Text: word, Pos: start, Adj: adj})
			}
			continue
		}
		// A maximal run over the operator-symbol alphabet: an exact fixed
		// operator keeps its kind; any other run is a user-declarable
		// OpSym (validated against the file's fixity table at parse).
		if opSymAlphabet[r] {
			i := 0
			for i < len(rest) && opSymAlphabet[rune(rest[i])] {
				i++
			}
			run := rest[:i]
			l.advance(i)
			if k, ok := fixedSymOps[run]; ok {
				out = append(out, Token{Kind: k, Text: run, Pos: start, Adj: adj})
			} else {
				out = append(out, Token{Kind: OpSym, Text: run, Pos: start, Adj: adj})
			}
			continue
		}
		// Operators, longest first.
		ops := []struct {
			text string
			kind Kind
		}{
			{">>>", Compose}, {">=>", Kleisli},
			{":=", Assign}, {"==", EqEq}, {"!=", NotEq}, {"<=", LtEq},
			{">=", GtEq}, {"&&", AndAnd}, {"||", OrOr}, {"|>", Pipe},
			{"->", Arrow}, {"=>", FatArrow}, {"<-", LArrow},
			{"=", Eq}, {":", Colon}, {"?", Question}, {"|", Bar}, {"!", Bang}, {"&", Amp},
			{"(", LParen}, {")", RParen}, {"[", LBrack}, {"]", RBrack},
			{"{", LBrace}, {"}", RBrace}, {",", Comma}, {";", Semi},
			{".", Dot}, {"+", Plus}, {"-", Minus}, {"*", Star},
			{"/", Slash}, {"%", Percent}, {"<", Lt}, {">", Gt}, {"@", At},
			{"`", EOF}, // backtick shows a better error below
		}
		matched := false
		for _, op := range ops {
			if strings.HasPrefix(rest, op.text) && op.kind != EOF {
				l.advance(len(op.text))
				out = append(out, Token{Kind: op.kind, Text: op.text, Pos: start, Adj: adj})
				matched = true
				break
			}
		}
		if !matched {
			return nil, l.errf(start, "unexpected character %q", r)
		}
	}
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
