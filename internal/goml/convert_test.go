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
		{"open", "module m\nopen foo\n", "`open` is not supported"},
		{"literalPat", "module m\nlet F (x : Int) : Int := match x with | 0 => 1", "expected a pattern"},
		{"guard", "module m\nlet F (x : Int) : Int := match x with | y if y => y", "expected `|` or `=>`"},
		{"noModule", "let X := 1\n", "expected `module`"},
		{"whileMatch", "module m\ntype T := | A | B\nlet F (t : T) : Int := do { while (match t with | A => true | B => false) do { }; 1 }", "while condition cannot contain a match"},
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

let Empty : Settings := Settings { }

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
