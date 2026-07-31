package memory

import "goforge.dev/goplus/std/option"

type Row2[A, B any] struct { First A; Second B }
type Row3[A, B, C any] struct { First A; Second B; Third C }

type SoA2[A, B any] struct { first []A; second []B }
type SoA3[A, B, C any] struct { first []A; second []B; third []C }

func NewSoA2[A, B any](capacity int) SoA2[A, B] { if capacity < 0 { capacity = 0 }; return SoA2[A, B]{first: make([]A, 0, capacity), second: make([]B, 0, capacity)} }
func NewSoA3[A, B, C any](capacity int) SoA3[A, B, C] { if capacity < 0 { capacity = 0 }; return SoA3[A, B, C]{first: make([]A, 0, capacity), second: make([]B, 0, capacity), third: make([]C, 0, capacity)} }

func FromRows2[A, B any](rows []Row2[A, B]) SoA2[A, B] { columns := NewSoA2[A, B](len(rows)); for _, row := range rows { columns.Append(row.First, row.Second) }; return columns }
func FromRows3[A, B, C any](rows []Row3[A, B, C]) SoA3[A, B, C] { columns := NewSoA3[A, B, C](len(rows)); for _, row := range rows { columns.Append(row.First, row.Second, row.Third) }; return columns }

func (columns *SoA2[A, B]) Append(first A, second B) { columns.first = append(columns.first, first); columns.second = append(columns.second, second) }
func (columns *SoA3[A, B, C]) Append(first A, second B, third C) { columns.first = append(columns.first, first); columns.second = append(columns.second, second); columns.third = append(columns.third, third) }
func (columns SoA2[A, B]) Len() int { return len(columns.first) }
func (columns SoA3[A, B, C]) Len() int { return len(columns.first) }

func (columns SoA2[A, B]) At(index int) option.Option[Row2[A, B]] { if index < 0 || index >= columns.Len() { return option.None[Row2[A, B]]() }; return option.Some(Row2[A, B]{First: columns.first[index], Second: columns.second[index]}) }
func (columns SoA3[A, B, C]) At(index int) option.Option[Row3[A, B, C]] { if index < 0 || index >= columns.Len() { return option.None[Row3[A, B, C]]() }; return option.Some(Row3[A, B, C]{First: columns.first[index], Second: columns.second[index], Third: columns.third[index]}) }

func (columns SoA2[A, B]) Rows() []Row2[A, B] { rows := make([]Row2[A, B], columns.Len()); for index := range rows { rows[index] = Row2[A, B]{First: columns.first[index], Second: columns.second[index]} }; return rows }
func (columns SoA3[A, B, C]) Rows() []Row3[A, B, C] { rows := make([]Row3[A, B, C], columns.Len()); for index := range rows { rows[index] = Row3[A, B, C]{First: columns.first[index], Second: columns.second[index], Third: columns.third[index]} }; return rows }

func (columns *SoA2[A, B]) Reset() { clear(columns.first); clear(columns.second); columns.first = columns.first[:0]; columns.second = columns.second[:0] }
func (columns *SoA3[A, B, C]) Reset() { clear(columns.first); clear(columns.second); clear(columns.third); columns.first = columns.first[:0]; columns.second = columns.second[:0]; columns.third = columns.third[:0] }
func (columns *SoA2[A, B]) Release() { clear(columns.first); clear(columns.second); columns.first = nil; columns.second = nil }
func (columns *SoA3[A, B, C]) Release() { clear(columns.first); clear(columns.second); clear(columns.third); columns.first = nil; columns.second = nil; columns.third = nil }
