package memory

import (
	"sync"

	"goforge.dev/goplus/std/result"
)

type Group struct {
	mu sync.Mutex
	arena *Arena
	handles []Handle
	released bool
}

func (arena *Arena) Group() result.Result[*Group, Failure] {
	if arena == nil { return result.Err[*Group, Failure](InvalidConfiguration("arena is nil")) }
	stats := arena.Stats(); if stats.Closed { return result.Err[*Group, Failure](ArenaClosed()) }
	return result.Ok[*Group, Failure](&Group{arena: arena})
}

func (group *Group) Allocate(size int, alignment int) result.Result[Handle, Failure] {
	group.mu.Lock(); defer group.mu.Unlock()
	if group.released { return result.Err[Handle, Failure](GroupReleased()) }
	match group.arena.Allocate(size, alignment) { case result.Err(failure): return result.Err[Handle, Failure](failure); case result.Ok(handle): group.handles = append(group.handles, handle); return result.Ok[Handle, Failure](handle) }
}

func (group *Group) Reset() result.Result[Mutation, Failure] {
	group.mu.Lock(); defer group.mu.Unlock()
	if group.released { return result.Err[Mutation, Failure](GroupReleased()) }
	match group.releaseHandles() { case result.Err(failure): return result.Err[Mutation, Failure](failure); case result.Ok(_): group.handles = nil; return result.Ok[Mutation, Failure](Applied()) }
}

func (group *Group) Release() result.Result[Mutation, Failure] {
	group.mu.Lock(); defer group.mu.Unlock()
	if group.released { return result.Ok[Mutation, Failure](Applied()) }
	match group.releaseHandles() { case result.Err(failure): return result.Err[Mutation, Failure](failure); case result.Ok(_): group.handles = nil; group.released = true; return result.Ok[Mutation, Failure](Applied()) }
}

func (group *Group) Stats() (Allocations int, Released bool) { group.mu.Lock(); defer group.mu.Unlock(); return len(group.handles), group.released }

func (group *Group) releaseHandles() result.Result[Mutation, Failure] {
	for index := len(group.handles)-1; index >= 0; index-- { match group.arena.Delete(group.handles[index]) { case result.Err(failure): return result.Err[Mutation, Failure](failure); case result.Ok(_): } }
	return result.Ok[Mutation, Failure](Applied())
}
