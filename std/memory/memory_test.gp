package memory

import (
	"testing"
	"testing/quick"

	"goforge.dev/goplus/std/result"
)

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
