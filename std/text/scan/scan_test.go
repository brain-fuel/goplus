package scan

import (
	"strings"
	"testing"
	"unsafe"
)

// Conformance suite ported from org.jsoup.parser.CharacterReaderTest (jsoup-1.22.2),
// the source of this general scanner. HTML-token-specific consumers (attribute
// values, tag names) live with their caller, so those cases are not here.
//
// assertEquals(want, got) -> if got != want { t.Fatalf }. assertTrue/False and
// assertThrows(UncheckedIOException) -> assertPanics. Char/String overloads map
// to the char-named base method plus a *Seq variant (matches->Matches/MatchesSeq,
// consumeTo->ConsumeTo/ConsumeToSeq, nextIndexOf->NextIndexOf/NextIndexOfSeq).
//
// Skipped tests (require the streaming BufferedReader refill internals that the
// whole-buffer port deliberately collapses, or an external test resource):
//   - bufferUp                  : constructs CharacterReader(BufferedReader) — no streaming ctor
//   - containsIgnoreCaseBuffer  : asserts buffer-underrun behaviour ("we haven't
//                                 buffered up yet, we don't know") that cannot occur
//                                 when the whole input is buffered
//   - linenumbersAgreeWithEditor: reads /htmltests/large.html from disk (ParseTest)

const maxBufferLen = BufferSize

// assertPanics asserts that fn panics (jsoup's assertThrows(UncheckedIOException)).
func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic, got none")
		}
	}()
	fn()
}

// sameString reports pointer identity of two strings (jsoup's assertSame). The
// flyweight cache returns the exact stored string on a hit, so a cached hit and
// its origin share backing storage.
func sameString(a, b string) bool {
	return unsafe.StringData(a) == unsafe.StringData(b)
}

func TestCR_consume(t *testing.T) {
	r := New("one")
	if r.Pos() != 0 {
		t.Fatalf("pos = %d, want 0", r.Pos())
	}
	if r.Current() != 'o' {
		t.Fatalf("current = %q, want 'o'", r.Current())
	}
	if r.Consume() != 'o' {
		t.Fatalf("consume != 'o'")
	}
	if r.Pos() != 1 {
		t.Fatalf("pos = %d, want 1", r.Pos())
	}
	if r.Current() != 'n' {
		t.Fatalf("current != 'n'")
	}
	if r.Pos() != 1 {
		t.Fatalf("pos = %d, want 1", r.Pos())
	}
	if r.Consume() != 'n' {
		t.Fatalf("consume != 'n'")
	}
	if r.Consume() != 'e' {
		t.Fatalf("consume != 'e'")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	if r.Consume() != EOF {
		t.Fatalf("consume != EOF")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	if r.Consume() != EOF {
		t.Fatalf("consume != EOF")
	}
}

func TestCR_unconsume(t *testing.T) {
	r := New("one")
	if r.Consume() != 'o' {
		t.Fatalf("consume != 'o'")
	}
	if r.Current() != 'n' {
		t.Fatalf("current != 'n'")
	}
	r.Unconsume()
	if r.Current() != 'o' {
		t.Fatalf("current != 'o'")
	}

	if r.Consume() != 'o' {
		t.Fatalf("consume != 'o'")
	}
	if r.Consume() != 'n' {
		t.Fatalf("consume != 'n'")
	}
	if r.Consume() != 'e' {
		t.Fatalf("consume != 'e'")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	r.Unconsume()
	if r.IsEmpty() {
		t.Fatalf("empty")
	}
	if r.Current() != 'e' {
		t.Fatalf("current != 'e'")
	}
	if r.Consume() != 'e' {
		t.Fatalf("consume != 'e'")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}

	if r.Consume() != EOF {
		t.Fatalf("consume != EOF")
	}
	r.Unconsume() // read past, so have to eat again
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	r.Unconsume()
	if r.IsEmpty() {
		t.Fatalf("empty")
	}

	if r.Consume() != 'e' {
		t.Fatalf("consume != 'e'")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}

	if r.Consume() != EOF {
		t.Fatalf("consume != EOF")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}

	// unconsume all remaining characters
	for i := 0; i < 4; i++ {
		r.Unconsume()
	}
	assertPanics(t, r.Unconsume)
}

func TestCR_mark(t *testing.T) {
	r := New("one")
	r.Consume()
	r.Mark()
	if r.Pos() != 1 {
		t.Fatalf("pos = %d, want 1", r.Pos())
	}
	if r.Consume() != 'n' {
		t.Fatalf("consume != 'n'")
	}
	if r.Consume() != 'e' {
		t.Fatalf("consume != 'e'")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	r.RewindToMark()
	if r.Pos() != 1 {
		t.Fatalf("pos = %d, want 1", r.Pos())
	}
	if r.Consume() != 'n' {
		t.Fatalf("consume != 'n'")
	}
	if r.IsEmpty() {
		t.Fatalf("empty")
	}
	if r.Pos() != 2 {
		t.Fatalf("pos = %d, want 2", r.Pos())
	}
}

func TestCR_rewindToMark(t *testing.T) {
	r := New("nothing")
	// marking should be invalid
	assertPanics(t, r.RewindToMark)
}

func TestCR_consumeToEnd(t *testing.T) {
	in := "one two three"
	r := New(in)
	toEnd := r.ConsumeToEnd()
	if toEnd != in {
		t.Fatalf("toEnd = %q, want %q", toEnd, in)
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
}

func TestCR_nextIndexOfChar(t *testing.T) {
	in := "blah blah"
	r := New(in)

	if r.NextIndexOf('x') != -1 {
		t.Fatalf("nextIndexOf x != -1")
	}
	if r.NextIndexOf('h') != 3 {
		t.Fatalf("nextIndexOf h != 3")
	}
	pull := r.ConsumeTo('h')
	if pull != "bla" {
		t.Fatalf("pull = %q, want bla", pull)
	}
	r.Consume()
	if r.NextIndexOf('l') != 2 {
		t.Fatalf("nextIndexOf l != 2")
	}
	if r.ConsumeToEnd() != " blah" {
		t.Fatalf("consumeToEnd != ' blah'")
	}
	if r.NextIndexOf('x') != -1 {
		t.Fatalf("nextIndexOf x != -1")
	}
}

func TestCR_nextIndexOfString(t *testing.T) {
	in := "One Two something Two Three Four"
	r := New(in)

	if r.NextIndexOfSeq("Foo") != -1 {
		t.Fatalf("nextIndexOf Foo != -1")
	}
	if r.NextIndexOfSeq("Two") != 4 {
		t.Fatalf("nextIndexOf Two != 4")
	}
	if r.ConsumeToSeq("something") != "One Two " {
		t.Fatalf("consumeTo something")
	}
	if r.NextIndexOfSeq("Two") != 10 {
		t.Fatalf("nextIndexOf Two != 10")
	}
	if r.ConsumeToEnd() != "something Two Three Four" {
		t.Fatalf("consumeToEnd")
	}
	if r.NextIndexOfSeq("Two") != -1 {
		t.Fatalf("nextIndexOf Two != -1")
	}
}

func TestCR_nextIndexOfUnmatched(t *testing.T) {
	r := New("<[[one]]")
	if r.NextIndexOfSeq("]]>") != -1 {
		t.Fatalf("nextIndexOf ]]> != -1")
	}
}

func TestCR_consumeToChar(t *testing.T) {
	r := New("One Two Three")
	if r.ConsumeTo('T') != "One " {
		t.Fatalf("consumeTo T")
	}
	if r.ConsumeTo('T') != "" { // on Two
		t.Fatalf("consumeTo T empty")
	}
	if r.Consume() != 'T' {
		t.Fatalf("consume != T")
	}
	if r.ConsumeTo('T') != "wo " {
		t.Fatalf("consumeTo T = wo ")
	}
	if r.Consume() != 'T' {
		t.Fatalf("consume != T")
	}
	if r.ConsumeTo('T') != "hree" { // consume to end
		t.Fatalf("consumeTo T = hree")
	}
}

func TestCR_consumeToString(t *testing.T) {
	r := New("One Two Two Four")
	if r.ConsumeToSeq("Two") != "One " {
		t.Fatalf("consumeTo Two = One ")
	}
	if r.Consume() != 'T' {
		t.Fatalf("consume != T")
	}
	if r.ConsumeToSeq("Two") != "wo " {
		t.Fatalf("consumeTo Two = wo ")
	}
	if r.Consume() != 'T' {
		t.Fatalf("consume != T")
	}
	// To handle strings straddling across buffers, consumeTo() may return the
	// data in multiple pieces near EOF.
	var builder strings.Builder
	var part string
	for {
		part = r.ConsumeToSeq("Qux")
		builder.WriteString(part)
		if part == "" {
			break
		}
	}
	if builder.String() != "wo Four" {
		t.Fatalf("builder = %q, want 'wo Four'", builder.String())
	}
}

func TestCR_advance(t *testing.T) {
	r := New("One Two Three")
	if r.Consume() != 'O' {
		t.Fatalf("consume != O")
	}
	r.Advance()
	if r.Consume() != 'e' {
		t.Fatalf("consume != e")
	}
}

func TestCR_consumeToAny(t *testing.T) {
	r := New("One &bar; qux")
	if r.ConsumeToAny('&', ';') != "One " {
		t.Fatalf("consumeToAny = One ")
	}
	if !r.Matches('&') {
		t.Fatalf("matches &")
	}
	if !r.MatchesSeq("&bar;") {
		t.Fatalf("matches &bar;")
	}
	if r.Consume() != '&' {
		t.Fatalf("consume != &")
	}
	if r.ConsumeToAny('&', ';') != "bar" {
		t.Fatalf("consumeToAny = bar")
	}
	if r.Consume() != ';' {
		t.Fatalf("consume != ;")
	}
	if r.ConsumeToAny('&', ';') != " qux" {
		t.Fatalf("consumeToAny = ' qux'")
	}
}

func TestCR_consumeLetterSequence(t *testing.T) {
	r := New("One &bar; qux")
	if r.ConsumeLetterSequence() != "One" {
		t.Fatalf("consumeLetterSequence = One")
	}
	if r.ConsumeToSeq("bar;") != " &" {
		t.Fatalf("consumeTo bar; = ' &'")
	}
	if r.ConsumeLetterSequence() != "bar" {
		t.Fatalf("consumeLetterSequence = bar")
	}
	if r.ConsumeToEnd() != "; qux" {
		t.Fatalf("consumeToEnd = '; qux'")
	}
}

func TestCR_consumeLetterThenDigitSequence(t *testing.T) {
	r := New("One12 Two &bar; qux")
	if r.ConsumeLetterThenDigitSequence() != "One12" {
		t.Fatalf("consumeLetterThenDigitSequence = One12")
	}
	if r.Consume() != ' ' {
		t.Fatalf("consume != ' '")
	}
	if r.ConsumeLetterThenDigitSequence() != "Two" {
		t.Fatalf("consumeLetterThenDigitSequence = Two")
	}
	if r.ConsumeToEnd() != " &bar; qux" {
		t.Fatalf("consumeToEnd = ' &bar; qux'")
	}
}

func TestCR_matches(t *testing.T) {
	r := New("One Two Three")
	if !r.Matches('O') {
		t.Fatalf("matches O")
	}
	if !r.MatchesSeq("One Two Three") {
		t.Fatalf("matches One Two Three")
	}
	if !r.MatchesSeq("One") {
		t.Fatalf("matches One")
	}
	if r.MatchesSeq("one") {
		t.Fatalf("matches one")
	}
	if r.Consume() != 'O' {
		t.Fatalf("consume != O")
	}
	if r.MatchesSeq("One") {
		t.Fatalf("matches One")
	}
	if !r.MatchesSeq("ne Two Three") {
		t.Fatalf("matches ne Two Three")
	}
	if r.MatchesSeq("ne Two Three Four") {
		t.Fatalf("matches ne Two Three Four")
	}
	if r.ConsumeToEnd() != "ne Two Three" {
		t.Fatalf("consumeToEnd")
	}
	if r.MatchesSeq("ne") {
		t.Fatalf("matches ne")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
}

func TestCR_matchesIgnoreCase(t *testing.T) {
	r := New("One Two Three")
	if !r.MatchesIgnoreCase("O") {
		t.Fatalf("matchesIgnoreCase O")
	}
	if !r.MatchesIgnoreCase("o") {
		t.Fatalf("matchesIgnoreCase o")
	}
	if !r.Matches('O') {
		t.Fatalf("matches O")
	}
	if r.Matches('o') {
		t.Fatalf("matches o")
	}
	if !r.MatchesIgnoreCase("One Two Three") {
		t.Fatalf("matchesIgnoreCase One Two Three")
	}
	if !r.MatchesIgnoreCase("ONE two THREE") {
		t.Fatalf("matchesIgnoreCase ONE two THREE")
	}
	if !r.MatchesIgnoreCase("One") {
		t.Fatalf("matchesIgnoreCase One")
	}
	if !r.MatchesIgnoreCase("one") {
		t.Fatalf("matchesIgnoreCase one")
	}
	if r.Consume() != 'O' {
		t.Fatalf("consume != O")
	}
	if r.MatchesIgnoreCase("One") {
		t.Fatalf("matchesIgnoreCase One")
	}
	if !r.MatchesIgnoreCase("NE Two Three") {
		t.Fatalf("matchesIgnoreCase NE Two Three")
	}
	if r.MatchesIgnoreCase("ne Two Three Four") {
		t.Fatalf("matchesIgnoreCase ne Two Three Four")
	}
	if r.ConsumeToEnd() != "ne Two Three" {
		t.Fatalf("consumeToEnd")
	}
	if r.MatchesIgnoreCase("ne") {
		t.Fatalf("matchesIgnoreCase ne")
	}
}

func TestCR_containsIgnoreCase(t *testing.T) {
	r := New("One TWO three")
	if !r.ContainsIgnoreCase("two") {
		t.Fatalf("containsIgnoreCase two")
	}
	if !r.ContainsIgnoreCase("three") {
		t.Fatalf("containsIgnoreCase three")
	}
	// weird one: does not find one, because it scans for consistent case only
	if r.ContainsIgnoreCase("one") {
		t.Fatalf("containsIgnoreCase one")
	}
}

func TestCR_matchesAny(t *testing.T) {
	scan := []rune{' ', '\n', '\t'}
	r := New("One\nTwo\tThree")
	if r.MatchesAny(scan...) {
		t.Fatalf("matchesAny")
	}
	if r.ConsumeToAny(scan...) != "One" {
		t.Fatalf("consumeToAny = One")
	}
	if !r.MatchesAny(scan...) {
		t.Fatalf("matchesAny")
	}
	if r.Consume() != '\n' {
		t.Fatalf("consume != \\n")
	}
	if r.MatchesAny(scan...) {
		t.Fatalf("matchesAny")
	}
	// nothing to match
	r.ConsumeToEnd()
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	if r.MatchesAny(scan...) {
		t.Fatalf("matchesAny")
	}
}

func TestCR_matchesDigit(t *testing.T) {
	r := New("42")
	r.ConsumeToEnd()
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	// nothing to match
	if r.MatchesDigit() {
		t.Fatalf("matchesDigit")
	}
	r.Unconsume()
	if !r.MatchesDigit() {
		t.Fatalf("matchesDigit")
	}
}

func TestCR_cachesStrings(t *testing.T) {
	r := New("Check\tCheck\tCheck\tCHOKE\tA string that is longer than 16 chars")
	one := r.ConsumeTo('\t')
	r.Consume()
	two := r.ConsumeTo('\t')
	r.Consume()
	three := r.ConsumeTo('\t')
	r.Consume()
	four := r.ConsumeTo('\t')
	r.Consume()
	five := r.ConsumeTo('\t')

	if one != "Check" {
		t.Fatalf("one = %q", one)
	}
	if two != "Check" {
		t.Fatalf("two = %q", two)
	}
	if three != "Check" {
		t.Fatalf("three = %q", three)
	}
	if four != "CHOKE" {
		t.Fatalf("four = %q", four)
	}
	if !sameString(one, two) {
		t.Fatalf("one !same two")
	}
	if !sameString(two, three) {
		t.Fatalf("two !same three")
	}
	if sameString(three, four) {
		t.Fatalf("three same four")
	}
	if sameString(four, five) {
		t.Fatalf("four same five")
	}
	if five != "A string that is longer than 16 chars" {
		t.Fatalf("five = %q", five)
	}
}

func TestCR_rangeEquals(t *testing.T) {
	r := New("Check\tCheck\tCheck\tCHOKE")
	if !r.RangeEquals(0, 5, "Check") {
		t.Fatalf("rangeEquals 0")
	}
	if r.RangeEquals(0, 5, "CHOKE") {
		t.Fatalf("rangeEquals 0 CHOKE")
	}
	if r.RangeEquals(0, 5, "Chec") {
		t.Fatalf("rangeEquals 0 Chec")
	}

	if !r.RangeEquals(6, 5, "Check") {
		t.Fatalf("rangeEquals 6")
	}
	if r.RangeEquals(6, 5, "Chuck") {
		t.Fatalf("rangeEquals 6 Chuck")
	}

	if !r.RangeEquals(12, 5, "Check") {
		t.Fatalf("rangeEquals 12")
	}
	if r.RangeEquals(12, 5, "Cheeky") {
		t.Fatalf("rangeEquals 12 Cheeky")
	}

	if !r.RangeEquals(18, 5, "CHOKE") {
		t.Fatalf("rangeEquals 18")
	}
	if r.RangeEquals(18, 5, "CHIKE") {
		t.Fatalf("rangeEquals 18 CHIKE")
	}
}

func TestCR_empty(t *testing.T) {
	r := New("One")
	if !r.MatchConsume("One") {
		t.Fatalf("matchConsume One")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}

	r = New("Two")
	two := r.ConsumeToEnd()
	if two != "Two" {
		t.Fatalf("two = %q", two)
	}
}

func TestCR_consumeToNonexistentEndWhenAtAnd(t *testing.T) {
	r := New("<!")
	if !r.MatchConsume("<!") {
		t.Fatalf("matchConsume <!")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}

	after := r.ConsumeTo('>')
	if after != "" {
		t.Fatalf("after = %q", after)
	}

	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
}

func TestCR_notEmptyAtBufferSplitPoint(t *testing.T) {
	length := BufferSize * 12
	var builder strings.Builder
	for builder.Len() <= length {
		builder.WriteByte('!')
	}
	r := New(builder.String())

	// consume through
	for pos := 0; pos < length; pos++ {
		if r.Pos() != pos {
			t.Fatalf("pos = %d, want %d", r.Pos(), pos)
		}
		if r.IsEmpty() {
			t.Fatalf("empty at %d", pos)
		}
		if r.Consume() != '!' {
			t.Fatalf("consume != !")
		}
		if r.Pos() != pos+1 {
			t.Fatalf("pos = %d, want %d", r.Pos(), pos+1)
		}
		if r.IsEmpty() {
			t.Fatalf("empty at %d", pos)
		}
	}
	if r.Consume() != '!' {
		t.Fatalf("consume != !")
	}
	if !r.IsEmpty() {
		t.Fatalf("not empty")
	}
	if r.Consume() != EOF {
		t.Fatalf("consume != EOF")
	}
}

func TestCR_canEnableAndDisableLineNumberTracking(t *testing.T) {
	reader := New("Hello!")
	if reader.IsTrackNewlines() {
		t.Fatalf("track newlines on")
	}
	reader.TrackNewlines(true)
	if !reader.IsTrackNewlines() {
		t.Fatalf("track newlines off")
	}
	reader.TrackNewlines(false)
	if reader.IsTrackNewlines() {
		t.Fatalf("track newlines on")
	}
}

func TestCR_canTrackNewlines(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("<foo>\n<bar>\n<qux>\n")
	for builder.Len() < maxBufferLen {
		builder.WriteString("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")
	}
	builder.WriteString("[foo]\n[bar]")
	content := builder.String()

	noTrack := New(content)
	if noTrack.IsTrackNewlines() {
		t.Fatalf("noTrack tracking on")
	}
	track := New(content)
	track.TrackNewlines(true)
	if !track.IsTrackNewlines() {
		t.Fatalf("track tracking off")
	}

	// check that no tracking works as expected
	if noTrack.Pos() != 0 {
		t.Fatalf("noTrack pos")
	}
	if noTrack.LineNumber() != 1 {
		t.Fatalf("noTrack line")
	}
	if noTrack.ColumnNumber() != 1 {
		t.Fatalf("noTrack col")
	}
	noTrack.ConsumeToSeq("<qux>")
	if noTrack.Pos() != 12 {
		t.Fatalf("noTrack pos = %d, want 12", noTrack.Pos())
	}
	if noTrack.LineNumber() != 1 {
		t.Fatalf("noTrack line")
	}
	if noTrack.ColumnNumber() != 13 {
		t.Fatalf("noTrack col = %d, want 13", noTrack.ColumnNumber())
	}
	if noTrack.PosLineCol() != "1:13" {
		t.Fatalf("noTrack posLineCol = %q", noTrack.PosLineCol())
	}
	// get over the buffer
	for !noTrack.MatchesSeq("[foo]") {
		noTrack.ConsumeToSeq("[foo]")
	}
	if noTrack.Pos() != 2090 {
		t.Fatalf("noTrack pos = %d, want 2090", noTrack.Pos())
	}
	if noTrack.LineNumber() != 1 {
		t.Fatalf("noTrack line")
	}
	if noTrack.ColumnNumber() != noTrack.Pos()+1 {
		t.Fatalf("noTrack col")
	}
	if noTrack.PosLineCol() != "1:2091" {
		t.Fatalf("noTrack posLineCol = %q", noTrack.PosLineCol())
	}

	// and the line numbers: "<foo>\n<bar>\n<qux>\n"
	if track.Pos() != 0 {
		t.Fatalf("track pos")
	}
	if track.LineNumber() != 1 {
		t.Fatalf("track line")
	}
	if track.ColumnNumber() != 1 {
		t.Fatalf("track col")
	}

	track.ConsumeTo('\n')
	if track.LineNumber() != 1 {
		t.Fatalf("track line")
	}
	if track.ColumnNumber() != 6 {
		t.Fatalf("track col = %d, want 6", track.ColumnNumber())
	}
	track.Consume()
	if track.LineNumber() != 2 {
		t.Fatalf("track line = %d, want 2", track.LineNumber())
	}
	if track.ColumnNumber() != 1 {
		t.Fatalf("track col")
	}

	if track.ConsumeTo('\n') != "<bar>" {
		t.Fatalf("track consumeTo bar")
	}
	if track.LineNumber() != 2 {
		t.Fatalf("track line")
	}
	if track.ColumnNumber() != 6 {
		t.Fatalf("track col")
	}

	if track.ConsumeToSeq("<qux>") != "\n" {
		t.Fatalf("track consumeTo qux")
	}
	if track.Pos() != 12 {
		t.Fatalf("track pos")
	}
	if track.LineNumber() != 3 {
		t.Fatalf("track line = %d, want 3", track.LineNumber())
	}
	if track.ColumnNumber() != 1 {
		t.Fatalf("track col")
	}
	if track.PosLineCol() != "3:1" {
		t.Fatalf("track posLineCol")
	}
	if track.ConsumeTo('\n') != "<qux>" {
		t.Fatalf("track consumeTo qux2")
	}
	if track.PosLineCol() != "3:6" {
		t.Fatalf("track posLineCol = %q, want 3:6", track.PosLineCol())
	}
	// get over the buffer
	for !track.MatchesSeq("[foo]") {
		track.ConsumeToSeq("[foo]")
	}
	if track.Pos() != 2090 {
		t.Fatalf("track pos = %d, want 2090", track.Pos())
	}
	if track.LineNumber() != 4 {
		t.Fatalf("track line = %d, want 4", track.LineNumber())
	}
	if track.ColumnNumber() != 2073 {
		t.Fatalf("track col = %d, want 2073", track.ColumnNumber())
	}
	if track.PosLineCol() != "4:2073" {
		t.Fatalf("track posLineCol")
	}
	track.ConsumeTo('\n')
	if track.PosLineCol() != "4:2078" {
		t.Fatalf("track posLineCol = %q, want 4:2078", track.PosLineCol())
	}

	track.ConsumeToSeq("[bar]")
	if track.LineNumber() != 5 {
		t.Fatalf("track line = %d, want 5", track.LineNumber())
	}
	if track.PosLineCol() != "5:1" {
		t.Fatalf("track posLineCol")
	}
	track.ConsumeToEnd()
	if track.PosLineCol() != "5:6" {
		t.Fatalf("track posLineCol = %q, want 5:6", track.PosLineCol())
	}
}

func TestCR_countsColumnsOverBufferWhenNoNewlines(t *testing.T) {
	var builder strings.Builder
	for builder.Len() < maxBufferLen*4 {
		builder.WriteString("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")
	}
	content := builder.String()
	reader := New(content)
	reader.TrackNewlines(true)

	if reader.PosLineCol() != "1:1" {
		t.Fatalf("posLineCol = %q, want 1:1", reader.PosLineCol())
	}
	var seen strings.Builder
	for !reader.IsEmpty() {
		seen.WriteRune(reader.Consume())
	}
	if seen.String() != content {
		t.Fatalf("seen != content")
	}
	if reader.Pos() != len([]rune(content)) {
		t.Fatalf("pos = %d, want %d", reader.Pos(), len([]rune(content)))
	}
	if reader.ColumnNumber() != reader.Pos()+1 {
		t.Fatalf("col")
	}
	if reader.LineNumber() != 1 {
		t.Fatalf("line")
	}
}
