package memory

import (
	"reflect"
	"testing"
	"testing/quick"

	"goforge.dev/goplus/std/result"
)

func TestGroupResetReclaimsAndReusesMembers(t *testing.T) {
	arena := requireArena(t); defer arena.Close(); var group *Group
	match arena.Group() { case result.Err(failure): t.Fatal(failure); case result.Ok(created): group = created }
	first := requireGroupAllocation(t, group, 32, 8); requireGroupAllocation(t, group, 16, 8)
	match group.Reset() { case result.Err(failure): t.Fatal(failure); case result.Ok(_): }
	if arena.Stats().Used != 0 { t.Fatalf("used = %d", arena.Stats().Used) }
	fresh := requireGroupAllocation(t, group, 32, 8); if first.offset != fresh.offset { t.Fatalf("offset = %d; want %d", fresh.offset, first.offset) }
	match arena.Bytes(first) { case result.Ok(_): t.Fatal("group reset revived old handle"); case result.Err(failure): match failure { case InvalidHandle: case _: t.Fatal(failure) } }
}

func TestReleasedGroupRejectsAllocation(t *testing.T) {
	arena := requireArena(t); defer arena.Close(); var group *Group
	match arena.Group() { case result.Err(failure): t.Fatal(failure); case result.Ok(created): group = created }
	match group.Release() { case result.Err(failure): t.Fatal(failure); case result.Ok(_): }
	match group.Allocate(1, 1) { case result.Err(failure): match failure { case GroupReleased: case _: t.Fatal(failure) }; case result.Ok(_): t.Fatal("released group allocated") }
}

func TestSoARoundTripProperty(t *testing.T) {
	law := func(values []int16) bool { rows := make([]Row3[int16, int, bool], len(values)); for index, value := range values { rows[index] = Row3[int16, int, bool]{First: value, Second: index, Third: index%2 == 0} }; columns := FromRows3(rows); return reflect.DeepEqual(columns.Rows(), rows) }
	if err := quick.Check(law, nil); err != nil { t.Fatal(err) }
}

func TestSoAResetRetainsCapacityAndErasesLength(t *testing.T) { columns := NewSoA2[int, string](8); columns.Append(1, "secret"); columns.Reset(); if columns.Len() != 0 || cap(columns.first) < 8 || cap(columns.second) < 8 { t.Fatalf("reset = %d/%d/%d", columns.Len(), cap(columns.first), cap(columns.second)) } }

func requireGroupAllocation(t *testing.T, group *Group, size int, alignment int) Handle { match group.Allocate(size, alignment) { case result.Ok(handle): return handle; case result.Err(failure): t.Fatal(failure); return Handle{} } }
