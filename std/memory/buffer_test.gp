package memory

import (
	"testing"
	"testing/quick"

	"goforge.dev/goplus/std/option"
)

func TestBufferRemovalPreservesOrderProperty(t *testing.T) { law := func(values []int, raw uint8) bool { buffer := NewBuffer[int](len(values)); for _, value := range values { buffer.Append(value) }; if len(values) == 0 { match buffer.Remove(0) { case option.None: return true; case option.Some(_): return false } }; index := int(raw)%len(values); match buffer.Remove(index) { case option.None: return false; case option.Some(removed): if removed != values[index] || buffer.Len() != len(values)-1 { return false }; for position := 0; position < buffer.Len(); position++ { expected := values[position]; if position >= index { expected = values[position+1] }; match buffer.At(position) { case option.None: return false; case option.Some(found): if found != expected { return false } } }; return true } }; if err := quick.Check(law, nil); err != nil { t.Fatal(err) } }

func TestBufferResetRetainsAndReleaseDropsCapacity(t *testing.T) { buffer := NewBuffer[*int](8); value := 1; buffer.Append(&value); buffer.Reset(); if buffer.Len() != 0 || buffer.Cap() < 8 { t.Fatalf("reset = %d/%d", buffer.Len(), buffer.Cap()) }; buffer.Release(); if buffer.Len() != 0 || buffer.Cap() != 0 { t.Fatalf("release = %d/%d", buffer.Len(), buffer.Cap()) } }

func TestBufferTruncateClearsTailAndRejectsGrowth(t *testing.T) { buffer := NewBuffer[int](4); buffer.Append(1); buffer.Append(2); if !buffer.Truncate(1) || buffer.Len() != 1 { t.Fatalf("truncate = %d", buffer.Len()) }; if buffer.Truncate(2) { t.Fatal("truncate grew buffer") }; match buffer.At(0) { case option.None: t.Fatal("prefix was removed"); case option.Some(value): if value != 1 { t.Fatalf("prefix = %d", value) } } }
