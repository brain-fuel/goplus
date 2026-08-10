package memory

import (
	"math"
	"testing"
	"testing/quick"

	"goforge.dev/goplus/std/result"
)

func TestOmittedPoliciesUseSecurePlatformDefaults(t *testing.T) {
	var arena *Arena
	match New(Config{Capacity: 32}) { case result.Err(failure): t.Fatal(failure); case result.Ok(created): arena = created }
	handle := requireAllocation(t, arena, 8, 8)
	match arena.Delete(handle) { case result.Err(failure): t.Fatal(failure); case result.Ok(_): }
	match arena.Close() { case result.Err(failure): t.Fatal(failure); case result.Ok(_): }
}

func TestAllocationAlignmentNeverOverflowsProperty(t *testing.T) {
	law := func(raw uint8) bool {
		arena := requireArena(t); defer arena.Close()
		requireAllocation(t, arena, int(raw%16)+1, 1)
		match arena.Allocate(1, math.MaxInt/2+1) {
		case result.Err(failure): match failure { case CapacityExhausted(): return true; case _: return false }
		case result.Ok(_): return false
		}
	}
	if err := quick.Check(law, nil); err != nil { t.Fatal(err) }
}

func TestGenerationInvalidatesHandles(t *testing.T) {
	arena := requireArena(t)
	defer arena.Close()
	var handle Handle
	match arena.Allocate(16, 8) {
	case result.Ok(allocated):
		handle = allocated
	case result.Err(failure):
		t.Fatal(failure)
	}
	match arena.Reset() {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	match arena.Bytes(handle) {
	case result.Ok(_):
		t.Fatal("expired handle was accepted")
	case result.Err(failure):
		match failure {
		case InvalidHandle():
		case _:
			t.Fatalf("unexpected failure: %v", failure)
		}
	}
}

func TestRollbackPreservesEarlierAllocations(t *testing.T) {
	arena := requireArena(t)
	defer arena.Close()
	first := requireAllocation(t, arena, 8, 8)
	var point Checkpoint
	match arena.Checkpoint() {
	case result.Ok(checkpoint):
		point = checkpoint
	case result.Err(failure):
		t.Fatal(failure)
	}
	later := requireAllocation(t, arena, 8, 8)
	match arena.Rollback(point) {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	match arena.Bytes(first) {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatalf("earlier allocation expired: %v", failure)
	}
	match arena.Bytes(later) {
	case result.Ok(_):
		t.Fatal("rolled-back handle survived")
	case result.Err(failure):
		match failure {
		case InvalidHandle():
		case _:
			t.Fatalf("unexpected failure: %v", failure)
		}
	}
}

func TestDeleteReuseNeverRevivesHandle(t *testing.T) {
	arena := requireArena(t)
	defer arena.Close()
	old := requireAllocation(t, arena, 64, 16)
	match arena.Delete(old) {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	fresh := requireAllocation(t, arena, 64, 16)
	if old.offset != fresh.offset {
		t.Fatalf("allocation was not reused: old=%d fresh=%d", old.offset, fresh.offset)
	}
	match arena.Bytes(old) {
	case result.Ok(_):
		t.Fatal("deleted handle revived")
	case result.Err(failure):
		match failure {
		case InvalidHandle():
		case _:
			t.Fatalf("unexpected failure: %v", failure)
		}
	}
}

func TestStatsTrackLiveAllocationsAcrossLifecycle(t *testing.T) {
	arena := requireArena(t)
	assertStats := func(used int, live int) {
		stats := arena.Stats()
		if stats.Used != used || stats.Allocations != live {
			t.Fatalf("stats mismatch: used=%d allocations=%d; want used=%d allocations=%d", stats.Used, stats.Allocations, used, live)
		}
	}
	assertStats(0, 0)
	first := requireAllocation(t, arena, 8, 8)
	assertStats(8, 1)
	var point Checkpoint
	match arena.Checkpoint() {
	case result.Ok(checkpoint):
		point = checkpoint
	case result.Err(failure):
		t.Fatal(failure)
	}
	second := requireAllocation(t, arena, 13, 1)
	third := requireAllocation(t, arena, 21, 1)
	assertStats(42, 3)
	match arena.Delete(second) {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	assertStats(29, 2)
	match arena.Rollback(point) {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	assertStats(8, 1)
	match arena.Delete(first) {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	assertStats(0, 0)
	match arena.Reset() {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	assertStats(0, 0)
	match arena.Close() {
	case result.Ok(_):
	case result.Err(failure):
		t.Fatal(failure)
	}
	assertStats(0, 0)
	_ = third
}

func TestAlignedAllocationsProperty(t *testing.T) {
	property := func(rawSize uint16, exponent uint8) bool {
		size := int(rawSize%128) + 1
		alignment := 1 << (exponent % 6)
		match New(Config{Capacity: 4096, Zero: ZeroOnRelease()}) {
		case result.Err(_):
			return false
		case result.Ok(arena):
			defer arena.Close()
			match arena.Allocate(size, alignment) {
			case result.Err(_):
				return false
			case result.Ok(handle):
				return handle.offset%alignment == 0
			}
		}
	}
	match result.Of(true, quick.Check(property, nil)) {
	case result.Err(cause): t.Fatal(cause)
	case result.Ok(_):
	}
}

func TestManagedStoragePreservesArenaSemantics(t *testing.T) {
	match New(Config{Capacity:64, Zero:ZeroOnRelease(), Storage:ManagedStorage()}) {
	case result.Err(failure): t.Fatal(failure)
	case result.Ok(arena):
		match arena.Allocate(16, 8) { case result.Err(failure): t.Fatal(failure); case result.Ok(handle): if handle.Len()!=16 { t.Fatalf("length = %d",handle.Len()) } }
		match arena.Close() { case result.Err(failure): t.Fatal(failure); case result.Ok(_): }
		if !arena.Stats().Closed { t.Fatal("managed arena remained open") }
	}
}

func requireArena(t *testing.T) *Arena {
	match New(Config{Capacity: 4096, Zero: ZeroOnRelease()}) {
	case result.Ok(arena):
		return arena
	case result.Err(failure):
		t.Fatal(failure)
		return nil
	}
}

func requireAllocation(t *testing.T, arena *Arena, size int, alignment int) Handle {
	match arena.Allocate(size, alignment) {
	case result.Ok(handle):
		return handle
	case result.Err(failure):
		t.Fatal(failure)
		return Handle{}
	}
}
