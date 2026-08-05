package javabackend

import (
	"strconv"
	"strings"
	"unicode"
)

func javaString(value string) string { return strconv.Quote(value) }

type packageMapper struct {
	modulePath string
	prefix     string
}

func (m packageMapper) javaPackage(goPath string) string {
	if goPath == m.modulePath {
		return m.prefix
	}
	if rest, ok := strings.CutPrefix(goPath, m.modulePath+"/"); ok {
		parts := strings.Split(rest, "/")
		for i := range parts {
			parts[i] = javaIdent(parts[i], false)
		}
		return m.prefix + "." + strings.Join(parts, ".")
	}
	return JavaPackage(goPath)
}

// JavaPackage reverses an import path's DNS name and preserves path segments:
// goforge.dev/x/y becomes dev.goforge.x.y.
func JavaPackage(goPath string) string {
	parts := strings.Split(goPath, "/")
	host := strings.Split(parts[0], ".")
	for i, j := 0, len(host)-1; i < j; i, j = i+1, j-1 {
		host[i], host[j] = host[j], host[i]
	}
	all := append(host, parts[1:]...)
	for i := range all {
		all[i] = javaIdent(all[i], false)
	}
	return strings.Join(all, ".")
}

func javaIdent(name string, exported bool) string {
	var b strings.Builder
	for i, r := range name {
		valid := unicode.IsLetter(r) || r == '_' || (i > 0 && unicode.IsDigit(r))
		if !valid {
			b.WriteByte('_')
			continue
		}
		if i == 0 && exported {
			r = unicode.ToUpper(r)
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		b.WriteByte('_')
	}
	out := b.String()
	if javaReserved[out] {
		return "_" + out
	}
	return out
}

var javaReserved = map[string]bool{
	"_": true, "abstract": true, "assert": true, "boolean": true, "break": true,
	"byte": true, "case": true, "catch": true, "char": true,
	"class": true, "const": true, "continue": true, "default": true,
	"do": true, "double": true, "else": true, "enum": true,
	"exports": true, "extends": true, "false": true, "final": true,
	"finally": true, "float": true, "for": true, "goto": true,
	"if": true, "implements": true, "import": true, "instanceof": true,
	"int": true, "interface": true, "long": true, "module": true,
	"native": true, "new": true, "non-sealed": true, "null": true,
	"open": true, "opens": true, "package": true, "permits": true,
	"private": true, "protected": true, "provides": true, "public": true,
	"record": true, "requires": true, "return": true, "sealed": true,
	"short": true, "static": true, "strictfp": true, "super": true,
	"switch": true, "synchronized": true, "this": true, "throw": true,
	"throws": true, "to": true, "transient": true, "transitive": true,
	"true": true, "try": true, "uses": true, "var": true, "void": true,
	"volatile": true, "while": true, "with": true, "yield": true,
}
