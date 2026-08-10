// Package memory provides explicit region lifetime, ownership, clearing, and
// reuse. Its primary Go+ API is algebraic: expected failures travel on Result,
// optional free-span locations travel on Option, and policy/state alternatives
// are exhaustive enums.
package memory

import (
	"fmt"
	"sync"
	"sync/atomic"

	"goforge.dev/goplus/std/option"
	"goforge.dev/goplus/std/result"
)

type ZeroPolicy enum {
	// ZeroOnRelease is the secure default and the zero value after erasure.
	ZeroOnRelease()
	// PreserveOnRelease is an explicit performance capability. Its caller must
	// prove released bytes are neither secret nor observable before overwrite.
	PreserveOnRelease()
}

type StoragePolicy enum {
	PlatformStorage()
	ManagedStorage()
}

type Failure enum {
	ArenaClosed()
	CapacityExhausted()
	InvalidHandle()
	InvalidCheckpoint()
	InvalidConfiguration(Message string)
	BoundsViolation(Operation string)
	PlatformFailure(Cause error)
	GroupReleased()
}

func (failure Failure) Error() string {
	return failureMessage(failure)
}

func failureMessage(failure Failure) string {
	match failure {
	case ArenaClosed():
		return "memory: arena is closed"
	case CapacityExhausted():
		return "memory: arena capacity exhausted"
	case InvalidHandle():
		return "memory: invalid or expired handle"
	case InvalidCheckpoint():
		return "memory: checkpoint does not belong to this arena generation"
	case InvalidConfiguration(message):
		return "memory: " + message
	case BoundsViolation(operation):
		return "memory: " + operation + " exceeds allocation bounds"
	case PlatformFailure(cause):
		return "memory: platform storage: " + cause.Error()
	case GroupReleased():
		return "memory: allocation group is released"
	}
}

func (failure Failure) Unwrap() error {
	return failureCause(failure)
}

func failureCause(failure Failure) error {
	match failure {
	case PlatformFailure(cause):
		return cause
	case _:
		return nil
	}
}

type Mutation enum {
	Applied()
}

type Config struct {
	Capacity int
	Zero     ZeroPolicy
	Storage  StoragePolicy
}

type Handle struct {
	id         uint64
	offset     int
	length     int
	generation uint64
}

func (handle Handle) Len() int { return handle.length }
func (handle Handle) IsZero() bool { return handle.id == 0 }

type Checkpoint struct {
	arena      uint64
	generation uint64
	order      int
}

type Stats struct {
	Capacity    int
	Used        int
	Peak        int
	Allocations int
	Generation  uint64
	Closed      bool
}

type allocation struct {
	offset int
	length int
	alive  bool
}

type span struct {
	offset int
	length int
}

type Arena struct {
	mu          sync.Mutex
	id          uint64
	storage     []byte
	cursor      int
	free        []span
	allocations map[uint64]allocation
	order       []uint64
	nextID      uint64
	generation  uint64
	used        int
	live        int
	peak        int
	zero        ZeroPolicy
	managed     bool
	closed      bool
}

var nextArenaID atomic.Uint64

func New(config Config) result.Result[*Arena, Failure] {
	if config.Capacity <= 0 {
		return result.Err[*Arena, Failure](InvalidConfiguration("capacity must be positive"))
	}
	if config.Zero == nil { config.Zero = ZeroOnRelease() }
	if config.Storage == nil { config.Storage = PlatformStorage() }
	if config.Storage != nil {
		match config.Storage {
		case ManagedStorage:
			return newArena(config, make([]byte, config.Capacity), true)
		case PlatformStorage:
		}
	}
	match result.Of(platformAllocate(config.Capacity)) {
	case result.Ok(storage):
		return newArena(config, storage, false)
	case result.Err(cause):
		return result.Err[*Arena, Failure](PlatformFailure(cause))
	}
}

func newArena(config Config, storage []byte, managed bool) result.Result[*Arena, Failure] {
	return result.Ok[*Arena, Failure](&Arena{
		id: nextArenaID.Add(1), storage: storage, allocations: map[uint64]allocation{},
		nextID: 1, generation: 1, zero: config.Zero, managed: managed,
	})
}

func (arena *Arena) Allocate(size int, alignment int) result.Result[Handle, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	if arena.closed {
		return result.Err[Handle, Failure](ArenaClosed())
	}
	if size < 0 {
		return result.Err[Handle, Failure](InvalidConfiguration("allocation size cannot be negative"))
	}
	if alignment <= 0 || alignment&(alignment-1) != 0 {
		return result.Err[Handle, Failure](InvalidConfiguration("alignment must be a positive power of two"))
	}
	if size == 0 {
		size = 1
	}
	offset := 0
	match arena.takeFree(size, alignment) {
	case option.Some(found):
		offset = found
	case option.None:
		match alignWithin(arena.cursor, alignment, len(arena.storage)) {
		case option.None:
			return result.Err[Handle, Failure](CapacityExhausted())
		case option.Some(aligned):
			offset = aligned
		}
		if size > len(arena.storage)-offset {
			return result.Err[Handle, Failure](CapacityExhausted())
		}
		arena.cursor = offset + size
	}
	id := arena.nextID
	arena.nextID++
	arena.allocations[id] = allocation{offset: offset, length: size, alive: true}
	arena.order = append(arena.order, id)
	arena.used += size
	arena.live++
	if arena.used > arena.peak {
		arena.peak = arena.used
	}
	return result.Ok[Handle, Failure](Handle{
		id:         id,
		offset:     offset,
		length:     size,
		generation: arena.generation,
	})
}

func (arena *Arena) Write(handle Handle, at int, source []byte) result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	match arena.resolve(handle) {
	case result.Err(failure):
		return result.Err[Mutation, Failure](failure)
	case result.Ok(current):
		if at < 0 || at > current.length || len(source) > current.length-at {
			return result.Err[Mutation, Failure](BoundsViolation("write"))
		}
		copy(arena.storage[current.offset+at:current.offset+at+len(source)], source)
		return result.Ok[Mutation, Failure](Applied())
	}
}

func (arena *Arena) Read(handle Handle, at int, destination []byte) result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	match arena.resolve(handle) {
	case result.Err(failure):
		return result.Err[Mutation, Failure](failure)
	case result.Ok(current):
		if at < 0 || at > current.length || len(destination) > current.length-at {
			return result.Err[Mutation, Failure](BoundsViolation("read"))
		}
		copy(destination, arena.storage[current.offset+at:current.offset+at+len(destination)])
		return result.Ok[Mutation, Failure](Applied())
	}
}

func (arena *Arena) Bytes(handle Handle) result.Result[[]byte, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	match arena.resolve(handle) {
	case result.Err(failure):
		return result.Err[[]byte, Failure](failure)
	case result.Ok(current):
		out := make([]byte, current.length)
		copy(out, arena.storage[current.offset:current.offset+current.length])
		return result.Ok[[]byte, Failure](out)
	}
}

func (arena *Arena) Zero(handle Handle) result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	match arena.resolve(handle) {
	case result.Err(failure):
		return result.Err[Mutation, Failure](failure)
	case result.Ok(current):
		clear(arena.storage[current.offset:current.offset+current.length])
		return result.Ok[Mutation, Failure](Applied())
	}
}

func (arena *Arena) Delete(handle Handle) result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	match arena.resolve(handle) {
	case result.Err(failure):
		return result.Err[Mutation, Failure](failure)
	case result.Ok(current):
		arena.release(handle.id, current)
		arena.coalesce()
		return result.Ok[Mutation, Failure](Applied())
	}
}

func (arena *Arena) Checkpoint() result.Result[Checkpoint, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	if arena.closed {
		return result.Err[Checkpoint, Failure](ArenaClosed())
	}
	return result.Ok[Checkpoint, Failure](Checkpoint{
		arena:      arena.id,
		generation: arena.generation,
		order:      len(arena.order),
	})
}

func (arena *Arena) Rollback(checkpoint Checkpoint) result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	if arena.closed {
		return result.Err[Mutation, Failure](ArenaClosed())
	}
	if checkpoint.arena != arena.id ||
		checkpoint.generation != arena.generation ||
		checkpoint.order < 0 ||
		checkpoint.order > len(arena.order) {
		return result.Err[Mutation, Failure](InvalidCheckpoint())
	}
	for _, id := range arena.order[checkpoint.order:] {
		current, present := arena.allocations[id]
		currentOption := option.Of(current, present)
		match currentOption {
		case option.Some(current):
			if current.alive {
				arena.release(id, current)
			}
		case option.None:
		}
	}
	arena.order = arena.order[:checkpoint.order]
	arena.coalesce()
	return result.Ok[Mutation, Failure](Applied())
}

func (arena *Arena) Reset() result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	if arena.closed {
		return result.Err[Mutation, Failure](ArenaClosed())
	}
	if zeroesReleased(arena.zero) {
		clear(arena.storage)
	}
	arena.cursor = 0
	arena.free = nil
	arena.allocations = map[uint64]allocation{}
	arena.order = nil
	arena.used = 0
	arena.live = 0
	arena.generation++
	return result.Ok[Mutation, Failure](Applied())
}

func (arena *Arena) Stats() Stats {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	return Stats{
		Capacity:    len(arena.storage),
		Used:        arena.used,
		Peak:        arena.peak,
		Allocations: arena.live,
		Generation:  arena.generation,
		Closed:      arena.closed,
	}
}

func (arena *Arena) Close() result.Result[Mutation, Failure] {
	arena.mu.Lock()
	defer arena.mu.Unlock()
	if arena.closed {
		return result.Ok[Mutation, Failure](Applied())
	}
	if zeroesReleased(arena.zero) {
		clear(arena.storage)
	}
	var release result.Result[struct{}, error] = result.Ok[struct{}, error](struct{}{})
	if !arena.managed { release = result.Of(struct{}{}, platformRelease(arena.storage)) }
	arena.storage = nil
	arena.free = nil
	arena.allocations = nil
	arena.order = nil
	arena.used = 0
	arena.live = 0
	arena.closed = true
	arena.generation++
	match release {
	case result.Ok(_):
		return result.Ok[Mutation, Failure](Applied())
	case result.Err(cause):
		return result.Err[Mutation, Failure](PlatformFailure(cause))
	}
}

func (arena *Arena) resolve(handle Handle) result.Result[allocation, Failure] {
	if arena.closed {
		return result.Err[allocation, Failure](ArenaClosed())
	}
	if handle.id == 0 || handle.generation != arena.generation {
		return result.Err[allocation, Failure](InvalidHandle())
	}
	current, present := arena.allocations[handle.id]
	currentOption := option.Of(current, present)
	match currentOption {
	case option.None:
		return result.Err[allocation, Failure](InvalidHandle())
	case option.Some(current):
		if !current.alive ||
			current.offset != handle.offset ||
			current.length != handle.length {
			return result.Err[allocation, Failure](InvalidHandle())
		}
		return result.Ok[allocation, Failure](current)
	}
}

func (arena *Arena) release(id uint64, current allocation) {
	if zeroesReleased(arena.zero) {
		clear(arena.storage[current.offset:current.offset+current.length])
	}
	current.alive = false
	arena.allocations[id] = current
	arena.free = append(arena.free, span{offset: current.offset, length: current.length})
	arena.used -= current.length
	arena.live--
}

func (arena *Arena) takeFree(size int, alignment int) option.Option[int] {
	for index, candidate := range arena.free {
		candidateEnd := candidate.offset + candidate.length
		offset := 0
		match alignWithin(candidate.offset, alignment, candidateEnd) {
		case option.None:
			continue
		case option.Some(aligned):
			offset = aligned
		}
		if size > candidateEnd-offset {
			continue
		}
		end := offset + size
		arena.free = append(arena.free[:index], arena.free[index+1:]...)
		if offset > candidate.offset {
			arena.free = append(arena.free, span{
				offset: candidate.offset,
				length: offset - candidate.offset,
			})
		}
		if end < candidateEnd {
			arena.free = append(arena.free, span{offset: end, length: candidateEnd - end})
		}
		return option.Some(offset)
	}
	return option.None[int]
}

func (arena *Arena) coalesce() {
	for leftIndex := 0; leftIndex < len(arena.free); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(arena.free); {
			left, right := arena.free[leftIndex], arena.free[rightIndex]
			if right.offset < left.offset {
				left, right = right, left
			}
			if left.offset+left.length == right.offset {
				arena.free[leftIndex] = span{
					offset: left.offset,
					length: left.length + right.length,
				}
				arena.free = append(arena.free[:rightIndex], arena.free[rightIndex+1:]...)
				continue
			}
			rightIndex++
		}
	}
}

func zeroesReleased(policy ZeroPolicy) bool {
	match policy {
	case ZeroOnRelease():
		return true
	case PreserveOnRelease():
		return false
	}
}

func alignWithin(value int, alignment int, limit int) option.Option[int] {
	if value < 0 || value > limit || alignment <= 0 || alignment&(alignment-1) != 0 {
		return option.None[int]()
	}
	remainder := value & (alignment - 1)
	if remainder == 0 { return option.Some(value) }
	padding := alignment - remainder
	if padding > limit-value { return option.None[int]() }
	return option.Some(value + padding)
}
