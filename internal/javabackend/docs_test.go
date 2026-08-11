package javabackend

import (
	"go/ast"
	"strings"
	"testing"
)

func TestWriteJavaDocEscapesAndEmitsContractTags(t *testing.T) {
	e := &emitter{}
	w := newJavaWriter()
	e.writeJavaDoc(w, &ast.CommentGroup{List: []*ast.Comment{{Text: "// Parse <input> without closing */ docs."}}}, "fallback", []string{"input"}, true)
	got := string(w.bytes())
	for _, want := range []string{"Parse &lt;input&gt;", "*&#47; docs", "@param input", "@return"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Javadoc missing %q:\n%s", want, got)
		}
	}
}
