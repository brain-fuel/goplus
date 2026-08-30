package goml

import (
	"strings"
	"testing"
)

func convertOK(t *testing.T, src string) string {
	t.Helper()
	out, err := Convert("test.goml", []byte(src))
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return string(out)
}

func TestConvertOptionModule(t *testing.T) {
	src := `module option

-- Option carries one value or nothing.
type Option (a : Type) :=
  | Some (value : a)
  | None

let Map (f : a -> b) (o : Option a) : Option b :=
  match o with
  | Some v => Some (f v)
  | None => None

let UnwrapOr (o : Option a) (d : a) : a :=
  match o with
  | Some v => v
  | None => d
`
	got := convertOK(t, src)
	want := `package option

// Option carries one value or nothing.
type Option[a any] enum {
	Some(value a)
	None
}

func Map[a any, b any](f func(a) b, o Option[a]) Option[b] {
	match o {
	case Some(v):
		return Some(f(v))
	case None:
		return None
	}
}

func UnwrapOr[a any](o Option[a], d a) a {
	match o {
	case Some(v):
		return v
	case None:
		return d
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertVecModule(t *testing.T) {
	src := `module vec

type Vec (a : Type) : Nat -> Type where
  | Nil : Vec a 0
  | Cons (head : a) (tail : Vec a n) : Vec a (n + 1)

let First : Vec a (n + 1) -> a
  | Cons h _ => h

let Rest : Vec a (n + 1) -> Vec a n
  | Cons _ t => t

total let Concat (xs : Vec a n) (ys : Vec a m) : Vec a (n + m) :=
  match xs with
  | Nil => ys
  | Cons h t => Cons h (Concat t ys)

let Cast {0 n m : Nat} (0 p : n = m) (v : Vec a n) : Vec a m := v
`
	got := convertOK(t, src)
	want := `package vec

type Vec[a any, n nat] enum {
	Nil() Vec[a, 0]
	Cons(head a, tail Vec[a, n]) Vec[a, n+1]
}

func First[a any](0 n nat, v Vec[a, n+1]) a {
	match v {
	case Cons(h, _):
		return h
	}
}

func Rest[a any](0 n nat, v Vec[a, n+1]) Vec[a, n] {
	match v {
	case Cons(_, t):
		return t
	}
}

total func Concat[a any](0 n nat, 0 m nat, xs Vec[a, n], ys Vec[a, m]) Vec[a, n+m] {
	match xs {
	case Nil():
		return ys
	case Cons(h, t):
		return Cons(h, Concat(t, ys))
	}
}

func Cast[a any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[a, n]) Vec[a, m] {
	return v
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertPropDecl(t *testing.T) {
	src := `module vec

type Vec (a : Type) : Nat -> Type where
  | Nil : Vec a 0
  | Cons (head : a) (tail : Vec a n) : Vec a (n + 1)

-- InRange names the bound facts, never a value.
type InRange (i : Nat) (n : Nat) := prop { And (Le 0 i) (Lt i n) }

type Same (n : Nat) (m : Nat) := prop { n = m }

let At {0 i n : Nat} (0 p : InRange i n) (v : Vec a n) : a :=
  match v with
  | Cons h _ => h

let Pick (v : Vec Int 3) : Int :=
  At 1 3 decide v
`
	got := convertOK(t, src)
	want := `package vec

type Vec[a any, n nat] enum {
	Nil() Vec[a, 0]
	Cons(head a, tail Vec[a, n]) Vec[a, n+1]
}

// InRange names the bound facts, never a value.
type InRange[i nat, n nat] prop { And[Le[0, i], Lt[i, n]] }

type Same[n nat, m nat] prop { Eq[n, m] }

func At[a any](0 i nat, 0 n nat, 0 p InRange[i, n], v Vec[a, n]) a {
	match v {
	case Cons(h, _):
		return h
	}
}

func Pick(v Vec[int, 3]) int {
	return At(1, 3, decide, v)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertInterfaceDecl(t *testing.T) {
	src := `module clock

import "time"
import "io"

-- Clock abstracts the wall clock.
type Clock := interface {
  Now : Unit -> time.Time;
  After : time.Duration -> Chan time.Time;
  io.Closer
}

type Sink (a : Type) := interface {
  Put : a -> Bool;
  Flush : Unit -> Unit
}

let Tick (c : Clock) : time.Time := c.Now ()
`
	got := convertOK(t, src)
	want := `package clock

import (
	"time"
	"io"
)

// Clock abstracts the wall clock.
type Clock interface {
	Now() time.Time
	After(time.Duration) chan time.Time
	io.Closer
}

type Sink[a any] interface {
	Put(a) bool
	Flush()
}

func Tick(c Clock) time.Time {
	return c.Now()
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertMakeAndConversions(t *testing.T) {
	src := `module conv

import "strings"

let Widen (n : Int) : Int64 := Int64 n

let Bytes (s : String) : Slice Byte := Slice Byte s

let Upper (bs : Slice Byte) : String := strings.ToUpper (String bs)

let Buffer (n : Int) : Chan Int := make (Chan Int) n

let Table () : Map String Int := make (Map String Int)

let Grow (n : Int) : Slice Int := make (Slice Int) n (n * 2)
`
	got := convertOK(t, src)
	want := `package conv

import (
	"strings"
)

func Widen(n int) int64 {
	return int64(n)
}

func Bytes(s string) []byte {
	return []byte(s)
}

func Upper(bs []byte) string {
	return strings.ToUpper(string(bs))
}

func Buffer(n int) chan int {
	return make(chan int, n)
}

func Table() map[string]int {
	return make(map[string]int)
}

func Grow(n int) []int {
	return make([]int, n, n * 2)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertExpectedTypes(t *testing.T) {
	src := `module lists

type Option (a : Type) :=
  | Some (value : a)
  | None

type Pair := { Tags : Slice String; Count : Int }

let Sum (xs : Slice Int) : Int := do {
  let mut t := 0;
  for _, x in xs do { t := t + x };
  t
}

let Total () : Int := Sum [2, 3, 4]

let Nested : Slice (Slice Int) := [[1], [2, 3]]

let Apply (f : Int -> String) (n : Int) : String := f n

let Render (n : Int) : String := Apply (fun x => "num") n

let Empty () : Slice Int := []

let Tagged () : Pair := Pair { Tags = ["a", "b"], Count = 2 }

let Pick (xs : Slice Int) (i : Int) : Int := xs[i]
`
	got := convertOK(t, src)
	want := `package lists

type Option[a any] enum {
	Some(value a)
	None
}

type Pair struct {
	Tags []string
	Count int
}

func Sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t = t + x
	}
	return t
}

func Total() int {
	return Sum([]int{2, 3, 4})
}

var Nested [][]int = [][]int{[]int{1}, []int{2, 3}}

func Apply(f func(int) string, n int) string {
	return f(n)
}

func Render(n int) string {
	return Apply(func(x int) string { return "num" }, n)
}

func Empty() []int {
	return []int{}
}

func Tagged() Pair {
	return Pair{Tags: []string{"a", "b"}, Count: 2}
}

func Pick(xs []int, i int) int {
	return xs[i]
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertLiteralPatterns(t *testing.T) {
	src := `module names

let Name (n : Nat) (0 p : Lt n 3) : String :=
  match n with
  | 0 => "zero"
  | 1 => "one"
  | 2 => "two"

let Describe (n : Nat) : String :=
  match n with
  | 0 => "none"
  | 1 => "single"
  | k => "many"
`
	got := convertOK(t, src)
	want := `package names

func Name(n nat, 0 p Lt[n, 3]) string {
	match n {
	case 0:
		return "zero"
	case 1:
		return "one"
	case 2:
		return "two"
	}
}

func Describe(n nat) string {
	match n {
	case 0:
		return "none"
	case 1:
		return "single"
	case _:
		return "many"
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertGuards(t *testing.T) {
	src := `module shape

type Shape :=
  | Circle (r : Int)
  | Rect (w : Int) (h : Int)

let Classify (s : Shape) : String :=
  match s with
  | Circle r if r > 10 => "big circle"
  | Circle _ => "circle"
  | Rect w h if w == h => "square"
  | Rect _ _ => "rect"

let Grade : Shape -> Int
  | Circle r if r > 10 => 2
  | Circle _ => 1
  | Rect _ _ => 0
`
	got := convertOK(t, src)
	want := `package shape

type Shape enum {
	Circle(r int)
	Rect(w int, h int)
}

func Classify(s Shape) string {
	match s {
	case Circle(r) if r > 10:
		return "big circle"
	case Circle(_):
		return "circle"
	case Rect(w, h) if w == h:
		return "square"
	case Rect(_, _):
		return "rect"
	}
}

func Grade(v Shape) int {
	match v {
	case Circle(r) if r > 10:
		return 2
	case Circle(_):
		return 1
	case Rect(_, _):
		return 0
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertImpossibleArm(t *testing.T) {
	src := `module vec

type Vec (a : Type) : Nat -> Type where
  | Nil : Vec a 0
  | Cons (head : a) (tail : Vec a n) : Vec a (n + 1)

let First : Vec a (n + 1) -> a
  | Cons h _ => h
  | Nil => impossible

let Rest (v : Vec a (n + 1)) : Vec a n :=
  match v with
  | Cons _ t => t
  | Nil => impossible
`
	got := convertOK(t, src)
	want := `package vec

type Vec[a any, n nat] enum {
	Nil() Vec[a, 0]
	Cons(head a, tail Vec[a, n]) Vec[a, n+1]
}

func First[a any](0 n nat, v Vec[a, n+1]) a {
	match v {
	case Cons(h, _):
		return h
	case Nil():
		impossible
	}
}

func Rest[a any](0 n nat, v Vec[a, n+1]) Vec[a, n] {
	match v {
	case Cons(_, t):
		return t
	case Nil():
		impossible
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertFixity(t *testing.T) {
	src := `module ops

infixl 5 <+> := Combine
infixr 6 <^> := Power

let Combine (a b : Int) : Int := a + b

let Power (a b : Int) : Int := a * b

let Demo (x y z : Int) : Int := x <+> y <+> z

let Rassoc (x y z : Int) : Int := x <^> y <^> z

let Mixed (x y z : Int) : Int := x <+> y * z
`
	got := convertOK(t, src)
	want := `package ops

func Combine(a, b int) int {
	return a + b
}

func Power(a, b int) int {
	return a * b
}

func Demo(x, y, z int) int {
	return Combine(Combine(x, y), z)
}

func Rassoc(x, y, z int) int {
	return Power(x, Power(y, z))
}

func Mixed(x, y, z int) int {
	return Combine(x, y * z)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertOpenExposing(t *testing.T) {
	src := `module app

import "goforge.dev/goplus/std/result"
open result exposing (Result, Ok, Err)

let Half (n : Int) : Result Int Error :=
  if n == 0 then Err nil else Ok (n / 2)

let Classify (r : Result Int Error) : Int :=
  match r with
  | Ok v => v
  | Err _ => 0
`
	got := convertOK(t, src)
	want := `package app

import (
	"goforge.dev/goplus/std/result"
)

func Half(n int) result.Result[int, error] {
	if n == 0 {
		return result.Err(nil)
	}
	return result.Ok(n / 2)
}

func Classify(r result.Result[int, error]) int {
	match r {
	case result.Ok(v):
		return v
	case result.Err(_):
		return 0
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertWhereHelpers(t *testing.T) {
	src := `module fact

let Factorial (n : UInt64) : UInt64 :=
  loop n 1
  where loop (k : UInt64) (acc : UInt64) : UInt64 :=
    if k == 0 then acc else loop (k - 1) (acc * k)

let Fib (n : Int) : Int :=
  even n
  where even (k : Int) : Int := if k == 0 then 1 else odd (k - 1);
        odd (k : Int) : Int := if k == 0 then 0 else even (k - 1)

let rec Ping (n : Int) : Int := if n == 0 then 0 else Pong (n - 1)

let rec Pong (n : Int) : Int := if n == 0 then 1 else Ping (n - 1)
`
	got := convertOK(t, src)
	want := `package fact

func Factorial(n uint64) uint64 {
	return loop(n, 1)
}

func loop(k uint64, acc uint64) uint64 {
	if k == 0 {
		return acc
	}
	return loop(k - 1, acc * k)
}

func Fib(n int) int {
	return even(n)
}

func even(k int) int {
	if k == 0 {
		return 1
	}
	return odd(k - 1)
}

func odd(k int) int {
	if k == 0 {
		return 0
	}
	return even(k - 1)
}

func Ping(n int) int {
	if n == 0 {
		return 0
	}
	return Pong(n - 1)
}

func Pong(n int) int {
	if n == 0 {
		return 1
	}
	return Ping(n - 1)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertRecordUpdate(t *testing.T) {
	src := `module cfg

type Settings := { Port : Int; Host : String }

let WithPort (s : Settings) (p : Int) : Settings :=
  { s with Port = p }

let Local (s : Settings) : Settings := do {
  let base := { s with Host = "localhost", Port = 8080 };
  base
}
`
	got := convertOK(t, src)
	want := `package cfg

type Settings struct {
	Port int
	Host string
}

func WithPort(s Settings, p int) Settings {
	u1 := s
	u1.Port = p
	return u1
}

func Local(s Settings) Settings {
	u1 := s
	u1.Host = "localhost"
	u1.Port = 8080
	base := u1
	return base
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertCtorShadowsBuiltin(t *testing.T) {
	src := `module msg

type Msg :=
  | String (value : Int)
  | Empty

let Wrap (n : Int) : Msg := String n
`
	got := convertOK(t, src)
	want := `package msg

type Msg enum {
	String(value int)
	Empty
}

func Wrap(n int) Msg {
	return String(n)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertTailAndIf(t *testing.T) {
	src := `module sums

@[tail]
let rec SumTo (n : UInt64) (acc : UInt64) : UInt64 :=
  if n == 0 then acc else SumTo (n - 1) (acc + n)
`
	got := convertOK(t, src)
	want := `package sums

tail func SumTo(n uint64, acc uint64) uint64 {
	if n == 0 {
		return acc
	}
	recur(n - 1, acc + n)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertClassAndInstance(t *testing.T) {
	src := `module algebra

import "reflect"

class Magma (t : Type) where
  Combine : t -> t -> t

class UnitalMagma (t : Type) extends Magma t where
  Empty : t
  law LeftId (a : t) := reflect.DeepEqual (Combine Empty a) a

instance IntAdd : UnitalMagma Int where
  Combine (a b : Int) : Int := a + b
  Empty : Int := 0
`
	got := convertOK(t, src)
	want := `package algebra

import (
	"reflect"
)

type Magma[t any] class {
	Combine(a t, b t) t
}

type UnitalMagma[t any] class {
	Magma[t]
	Empty() t
	law LeftId(a t) { return reflect.DeepEqual(Combine(Empty(), a), a) }
}

instance IntAdd UnitalMagma[int] {
	Combine(a, b int) int {
		return a + b
	}
	Empty() int {
		return 0
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertRefinementAndRecord(t *testing.T) {
	src := `module conf

type Port := { value : Int | 0 < value && value < 65536 }

type Config := { Port : Int @[json "port", yaml "port"]; Name : String @[json "name,omitempty"] }
`
	got := convertOK(t, src)
	want := `package conf

type Port refine(value int) { 0 < value && value < 65536 }

type Config struct {
	Port int ` + "`" + `json:"port" yaml:"port"` + "`" + `
	Name string ` + "`" + `json:"name,omitempty"` + "`" + `
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertErrors(t *testing.T) {
	cases := []struct {
		name, src, wantErr string
	}{
		{"openBare", "module m\nopen foo\n", "expected `exposing`"},
		{"opUndeclared", "module m\nlet F (a b : Int) : Int := a <+> b", "is not declared"},
		{"fixityPrec", "module m\ninfixl 9 <+> := F\n", "1 (loosest) to 6"},
		{"fixityDup", "module m\ninfixl 5 <+> := F\ninfixl 5 <+> := G\n", "already declared"},
		{"fixityInfix", "module m\ninfix 5 <+> := F\n", "pick infixl or infixr"},
		{"impossibleValue", "module m\nlet X : Int := impossible", "whole match arm"},
		{"openLower", "module m\nopen foo exposing (bar)\n", "exposed names are Capitalized"},
		{"openClash", "module m\nopen foo exposing (Ok)\nopen bar exposing (Ok)\n", "exposed by both"},
		{"floatPat", "module m\nlet F (x : Float64) : Int := match x with | 1.5 => 1", "expected a pattern"},
		{"guardAlts", "module m\ntype T := | A (n : Int) | B\nlet F (t : T) : Int := match t with | A _ | B if true => 1", "cannot take a guard"},
		{"noModule", "let X := 1\n", "expected `module`"},
		{"valueMatch", "module m\ntype T := | A | B\nlet S : T := A\nlet V : Int := match S with | A => 1 | B => 2", "cannot hoist at package level"},
		{"valueIf", "module m\nlet V : Int := if C then 1 else 2", "cannot hoist at package level"},
		{"valueGeneric", "module m\ntype Option (a : Type) := | Some (value : a) | None\nlet Nothing : Option a := None", "cannot be generic"},
		{"valueRec", "module m\nlet rec X : Int := X", "`let rec` needs binders"},
		{"valueTotal", "module m\ntotal let X : Int := 1", "`total` describes a function"},
		{"valueAttr", "module m\n@[laws \"out=lawtest\"]\nlet X : Int := 1", "binds a value"},
		{"valueLambda", "module m\nlet F := fun (x : Int) => x", "needs a result type"},
		{"whileMatch", "module m\ntype T := | A | B\nlet F (t : T) : Int := do { while (match t with | A => true | B => false) do { }; 1 }", "while condition cannot contain a match"},
		{"holeDuplicate", "module m\nlet F (a b : Int) : Int := ?gap + ?gap", "hole ?gap already appears"},
		{"holeSpaced", "module m\nlet F (n : Int) : Int := ? gap", "a typed hole is spelled ?name"},
		{"propDeriving", "module m\ntype P (n : Nat) := prop { Lt 0 n } deriving Eq", "cannot derive"},
		{"propImplicitBinder", "module m\ntype P {n : Nat} := prop { Lt 0 n }", "prop parameters are explicit"},
		{"propKind", "module m\ntype P : Nat -> Type := prop { Lt 0 n }", "takes binders, not a kind"},
		{"ifaceValueMember", "module m\nimport \"time\"\ntype I := interface { Now : time.Time }", "needs a function type"},
		{"ifaceUnitMixed", "module m\ntype I := interface { Put : Unit -> Int -> Bool }", "must stand alone"},
		{"ifaceDeriving", "module m\ntype I := interface { Len : Unit -> Int } deriving Eq", "cannot derive"},
		{"convArity", "module m\nlet F (a b : Int) : Int64 := Int64 a b", "exactly one value"},
		{"updateNoFields", "module m\ntype S := { A : Int }\nlet F (s : S) : S := { s with }", "at least one field"},
		{"listUnknownWant", "module m\nlet F (x : Int) : Int := Len [1, 2]", "position whose type is known"},
		{"listNonSlice", "module m\nlet F () : Int := [1]", "expects int, not a list literal"},
		{"lambdaUnknownWant", "module m\nlet F (g : Int) : Int := Use (fun x => x)", "binder x needs a type"},
		{"whereCapture", "module m\nlet F (n : Int) : Int := g 1\n  where g (k : Int) : Int := k + n", "cannot capture n"},
		{"whereDuplicate", "module m\nlet F (n : Int) : Int := g n\n  where g (k : Int) : Int := k\nlet H (n : Int) : Int := g n\n  where g (k : Int) : Int := k + 1", "file-unique"},
		{"whereUpper", "module m\nlet F (n : Int) : Int := G n\n  where G (k : Int) : Int := k", "must be lowercase"},
		{"updateValue", "module m\ntype S := { A : Int }\nlet Base : S := S { A = 1 }\nlet X : S := { Base with A = 2 }", "cannot hoist at package level"},
		{"sliceExprBare", "module m\nlet F (s : String) : Slice Byte := Slice Byte", "Slice converts"},
		{"ptrExpr", "module m\nlet F (n : Int) : Ptr Int := Ptr n", "type former"},
		{"makeNoType", "module m\nlet F () : Int := make 4", "make takes a type first"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert("test.goml", []byte(tc.src))
			if err == nil {
				t.Fatalf("expected error containing %q, got success", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestConvertLetStarAndDo(t *testing.T) {
	src := `module app

import "os"
import "strconv"

type Result (t : Type) (e : Type) :=
  | Ok (value : t)
  | Err (err : e)

let ReadPort (path : String) : Result Int Error := do {
  let raw := os.ReadFile path ?;
  let n := strconv.Atoi (trim raw) ?;
  Ok n
}

let Build (path : String) : Result Int Error :=
  let* a = ReadPort path in
  let* b = ReadPort path in
  Ok (a + b)
`
	got := convertOK(t, src)
	want := `package app

import (
	"os"
	"strconv"
)

type Result[t any, e any] enum {
	Ok(value t)
	Err(err e)
}

func ReadPort(path string) Result[int, error] {
	raw := os.ReadFile(path)?
	n := strconv.Atoi(trim(raw))?
	return Ok(n)
}

func Build(path string) Result[int, error] {
	a := ReadPort(path)?
	b := ReadPort(path)?
	return Ok(a + b)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertDoSelectWhileFor(t *testing.T) {
	src := `module worker

let Run (inbox : Chan Int) (done : Chan Bool) (acks : Chan Int) (s0 : Int) : Int := do {
  let mut s := s0;
  defer close acks;
  go warm s0;
  while true do {
    select with
    | m <- recv inbox => s := s + m
    | _ <- recv done => return s
    | _ <- send acks s => ()
    | default => idle ()
  };
  s
}

let Sum (xs : Slice Int) : Int := do {
  let mut acc := 0;
  for _, x in xs do {
    acc := acc + x
  };
  acc
}
`
	got := convertOK(t, src)
	want := `package worker

func Run(inbox chan int, done chan bool, acks chan int, s0 int) int {
	s := s0
	defer close(acks)
	go warm(s0)
	for {
		select {
		case m := <-inbox:
			s = s + m
		case <-done:
			return s
		case acks <- s:
		default:
			idle()
		}
	}
	return s
}

func Sum(xs []int) int {
	acc := 0
	for _, x := range xs {
		acc = acc + x
	}
	return acc
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertMultiColumnClausal(t *testing.T) {
	src := `module zips

type List (a : Type) :=
  | Nil2
  | Cons2 (head : a) (tail : List a)

let Weight : Int -> List Int -> Int
  | n, Nil2 => n
  | n, Cons2 h t => Weight (n + h) t
`
	got := convertOK(t, src)
	want := `package zips

type List[a any] enum {
	Nil2
	Cons2(head a, tail List[a])
}

func Weight(n int, v1 List[int]) int {
	match v1 {
	case Nil2:
		return n
	case Cons2(h, t):
		return Weight(n + h, t)
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertBareInstanceMembers(t *testing.T) {
	src := `module alg

class Semigroup2 (t : Type) where
  Combine : t -> t -> t

class Monoid2 (t : Type) extends Semigroup2 t where
  Empty : t

instance IntAdd : Monoid2 Int where
  Combine a b := a + b
  Empty := 0
`
	got := convertOK(t, src)
	want := `package alg

type Semigroup2[t any] class {
	Combine(a t, b t) t
}

type Monoid2[t any] class {
	Semigroup2[t]
	Empty() t
}

instance IntAdd Monoid2[int] {
	Combine(a, b int) int {
		return a + b
	}
	Empty() int {
		return 0
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertModuleAttrsAndMult(t *testing.T) {
	src := `@[laws "out=lawtest"]
module dup

let Dup {m : Mult} (m x : Slice Int) : Int := use x
`
	got := convertOK(t, src)
	want := `//goplus:laws out=lawtest
package dup

func Dup[m mult](m x ([]int)) int {
	return use(x)
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertNestedMatchHoists(t *testing.T) {
	src := `module shapes

type Shape :=
  | Circle (r : Int)
  | Rect (w : Int) (h : Int)

let Double (s : Shape) : Int :=
  let a := match s with | Circle r => r * r | Rect w h => w * h;
  a + a
`
	got := convertOK(t, src)
	want := `package shapes

type Shape enum {
	Circle(r int)
	Rect(w int, h int)
}

func Double(s Shape) int {
	a := match s {
	case Circle(r):
		r * r
	case Rect(w, h):
		w * h
	}
	a + a
	return a + a
}
`
	_ = want
	if !strings.Contains(got, "a := match s {") {
		t.Fatalf("let-bound match expression not lowered:\n%s", got)
	}
	if !strings.Contains(got, "return a + a") {
		t.Fatalf("tail expression not returned:\n%s", got)
	}
}

func TestConvertTypeIndexedGADT(t *testing.T) {
	src := `module expr

type Expr : Type -> Type where
  | Lit (v : Int) : Expr Int
  | Truth (b : Bool) : Expr Bool
  | If (c : Expr Bool) (t : Expr a) (e : Expr a) : Expr a

let Eval (e : Expr a) : a :=
  match e with
  | Lit v => v
  | Truth b => b
  | If c t e => if Eval c then Eval t else Eval e
`
	got := convertOK(t, src)
	want := `package expr

type Expr[a any] enum {
	Lit(v int) Expr[int]
	Truth(b bool) Expr[bool]
	If(c Expr[bool], t Expr[a], e Expr[a]) Expr[a]
}

func Eval[a any](e Expr[a]) a {
	match e {
	case Lit(v):
		return v
	case Truth(b):
		return b
	case If(c, t, e):
		if Eval(c) {
			return Eval(t)
		}
		return Eval(e)
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertRecordLiteralAndNot(t *testing.T) {
	src := `module conf

type Settings := { Port : Int; Host : String }

let Make (p : Int) (h : String) : Settings := Settings { Port = p, Host = h }

let Empty () : Settings := Settings { }

let Toggle (b : Bool) : Bool := !b
`
	got := convertOK(t, src)
	want := `package conf

type Settings struct {
	Port int
	Host string
}

func Make(p int, h string) Settings {
	return Settings{Port: p, Host: h}
}

func Empty() Settings {
	return Settings{}
}

func Toggle(b bool) bool {
	return !b
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertNullaryValues(t *testing.T) {
	src := `module conf

type Settings := { Port : Int; Host : String }

let Answer := 42

let Greeting : String := "hi"

let Defaults : Settings := Settings { Port = 80, Host = "localhost" }

let Doubled : Int := Answer * 2

let Inc : Int -> Int := fun (x : Int) => x + 1
`
	got := convertOK(t, src)
	want := `package conf

type Settings struct {
	Port int
	Host string
}

var Answer = 42

var Greeting string = "hi"

var Defaults Settings = Settings{Port: 80, Host: "localhost"}

var Doubled int = Answer * 2

var Inc func(int) int = func(x int) int { return x + 1 }
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertUnitBinderProcedures(t *testing.T) {
	src := `module app

let Boot () : Unit := start ()

let Compute () : Int := 1 + 1

let main () := do {
  println "hi"
}
`
	got := convertOK(t, src)
	want := `package app

func Boot() {
	start()
}

func Compute() int {
	return 1 + 1
}

func main() {
	println("hi")
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// A function with no result does not return from its then-arm, so its
// else must be a real else. Falling through would run both arms.
func TestConvertVoidIfKeepsElse(t *testing.T) {
	src := `module m

let Report (bad : Bool) : Unit :=
  if bad then do {
    warn ()
  } else do {
    proceed ()
  }
`
	got := convertOK(t, src)
	want := `package m

func Report(bad bool) {
	if bad {
		warn()
	} else {
		proceed()
	}
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// A binder used only inside a record literal or a do block is still
// used; blanking it would generate code that does not compile.
func TestConvertBinderUsedInsideNestedForms(t *testing.T) {
	src := `module m

type Box := { Held : Int }

type Maybe :=
  | Full (value : Int)
  | Empty

let Wrap (m : Maybe) : Box :=
  match m with
  | Full v => Box { Held = v }
  | Empty => Box { Held = 0 }

let Shout (m : Maybe) : Unit :=
  match m with
  | Full v => do { println v }
  | Empty => do { println "empty" }
`
	got := convertOK(t, src)
	if !strings.Contains(got, "case Full(v):\n\t\treturn Box{Held: v}") {
		t.Fatalf("binder used in a record literal was blanked:\n%s", got)
	}
	if !strings.Contains(got, "case Full(v):\n\t\tprintln(v)") {
		t.Fatalf("binder used in a do block was blanked:\n%s", got)
	}
}

func TestConvertGoInteropForms(t *testing.T) {
	src := `module m

import "encoding/json"
import "strings"

type Cfg := { Name : String }

let Decode (data : Slice Byte) : Cfg := do {
  let mut c := Cfg { Name = "" };
  let _ := json.Unmarshal data &c;
  c
}

let Tail (s : String) : String := do {
  let parts := strings.SplitN s "." 2;
  parts[1]
}

let Relay (src : Chan Int) (dst : Chan Int) : Unit := do {
  let v := <- src;
  dst <- v * 2
}

let Deref (p : Ptr Int) : Int := *p
`
	got := convertOK(t, src)
	for _, want := range []string{
		"json.Unmarshal(data, &c)",
		"return parts[1]",
		"v := <-src",
		"dst <- v * 2",
		"return *p",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestConvertQualifiedConstructorPatterns(t *testing.T) {
	src := `module m

import "goforge.dev/goplus/std/result" as result

let Describe (r : result.Result Int Error) : String :=
  match r with
  | result.Ok v => "got it"
  | result.Err e => "failed"
`
	got := convertOK(t, src)
	if !strings.Contains(got, "case result.Ok(_):") || !strings.Contains(got, "case result.Err(_):") {
		t.Fatalf("imported constructors did not match:\n%s", got)
	}
}

// An if used as a statement inside a loop must lower as a statement,
// and an empty else is how goml spells "no else".
func TestConvertIfInsideLoop(t *testing.T) {
	src := `module m

let Any (xs : Slice Int) : Bool := do {
  let mut found := false;
  for _, x in xs do {
    if x > 10 then do {
      found := true
    } else do { }
  };
  found
}
`
	got := convertOK(t, src)
	want := `package m

func Any(xs []int) bool {
	found := false
	for _, x := range xs {
		if x > 10 {
			found = true
		}
	}
	return found
}
`
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// A hole passes through to the .gp text verbatim: the core reports its
// goal, so goml neither infers nor rewrites anything.
func TestConvertHoles(t *testing.T) {
	src := `module m

let Pick (xs : Slice String) : String := ?choice

let Add (a : Int) : Int := a + ?rest

let Apply (f : Int -> Int) (n : Int) : Int := f ?arg
`
	got := convertOK(t, src)
	for _, want := range []string{
		"return ?choice",
		"return a + ?rest",
		"return f(?arg)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lowered .gp missing %q:\n%s", want, got)
		}
	}
}

// A hole's position is recorded by name, so its diagnostic can be placed
// exactly rather than through the line map's decl-grained lookup.
func TestConvertWithInfoHoles(t *testing.T) {
	src := `module m

let Pick (xs : Slice String) : String := ?choice

let Run (n : Int) : Int := do {
  let m := ?step;
  m + n
}
`
	_, info, err := ConvertWithInfo("test.goml", []byte(src))
	if err != nil {
		t.Fatalf("ConvertWithInfo: %v", err)
	}
	want := map[string]Pos{
		"choice": {Line: 3, Col: 42},
		"step":   {Line: 6, Col: 12},
	}
	if len(info.Holes) != len(want) {
		t.Fatalf("got %d holes, want %d: %v", len(info.Holes), len(want), info.Holes)
	}
	for name, pos := range want {
		if got := info.Holes[name]; got != pos {
			t.Errorf("hole ?%s at %v, want %v", name, got, pos)
		}
	}
}

// Postfix `?` and a typed hole never compete: the try's `?` is attached
// to the expression before it, a hole's to the name after it.
func TestConvertHoleAndTryCoexist(t *testing.T) {
	src := `module m

import "strconv"

let Parse (s : String) : Result Int Error := do {
  let n := strconv.Atoi s ?;
  Ok (n + ?bump)
}
`
	got := convertOK(t, src)
	if !strings.Contains(got, "strconv.Atoi(s)?") {
		t.Errorf("postfix try was not preserved:\n%s", got)
	}
	if !strings.Contains(got, "?bump") {
		t.Errorf("hole was not preserved:\n%s", got)
	}
}

// An or-pattern arm renders once per alternative, so the same hole is
// printed twice; that must not read as two holes of the same name.
func TestConvertHoleInOrPatternArm(t *testing.T) {
	src := `module m

type T :=
  | A (x : Int)
  | B (x : Int)

let F (t : T) : Int :=
  match t with
  | A x | B x => ?todo
`
	out, err := Convert("test.goml", []byte(src))
	if err != nil {
		t.Fatalf("or-pattern arm with a hole was rejected: %v", err)
	}
	if n := strings.Count(string(out), "?todo"); n != 2 {
		t.Fatalf("expected the arm body duplicated per alternative, got %d:\n%s", n, out)
	}
}
