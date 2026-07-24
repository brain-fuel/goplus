package smt

import (
	"math/big"
	"math/bits"
)

func floatingPointToBitVector(
	mode uint8,
	width int,
	value FloatingPointValue,
	signed bool,
) (BitVectorValue, bool) {
	if mode < 1 || mode > 5 {
		panic("smt: invalid floating-point rounding mode")
	}
	if width <= 0 {
		panic("smt: bit-vector width must be positive")
	}
	if FloatingPointIsNaN(value) || FloatingPointIsInfinite(value) {
		return NewBitVectorUint64(width, 0), false
	}
	exponentBits := FloatingPointExponentBits(value)
	significandBits := FloatingPointSignificandBits(value)
	if width <= 64 && exponentBits+significandBits <= 64 {
		if raw, ok := FloatingPointBits(value).Uint64(); ok {
			return floatingPointToBitVectorUint64(
				mode, width, exponentBits, significandBits, raw, signed,
			)
		}
	}
	rounded := floatingPointRoundToIntegral(mode, value)
	finite := decodeFloatingPointFinite(rounded)
	if finite.magnitude.Sign() == 0 {
		return NewBitVectorUint64(width, 0), true
	}
	if finite.negative && !signed {
		return NewBitVectorUint64(width, 0), false
	}
	magnitude := new(big.Int).Set(finite.magnitude)
	if finite.scale.Sign() >= 0 {
		if !finite.scale.IsInt64() ||
			finite.scale.Int64() > int64(width-magnitude.BitLen()+1) {
			return NewBitVectorUint64(width, 0), false
		}
		magnitude.Lsh(magnitude, uint(finite.scale.Int64()))
	} else {
		shift := new(big.Int).Neg(new(big.Int).Set(finite.scale))
		if !shift.IsInt64() || shift.Int64() >= int64(magnitude.BitLen()) {
			magnitude.SetInt64(0)
		} else {
			magnitude.Rsh(magnitude, uint(shift.Int64()))
		}
	}
	if !floatingPointIntegerFits(magnitude, finite.negative, width, signed) {
		return NewBitVectorUint64(width, 0), false
	}
	if finite.negative {
		magnitude.Neg(magnitude)
	}
	return bitVectorValueFromBig(width, magnitude), true
}

func floatingPointIntegerFits(
	magnitude *big.Int,
	negative bool,
	width int,
	signed bool,
) bool {
	if magnitude.Sign() == 0 {
		return true
	}
	if !signed {
		return !negative && magnitude.BitLen() <= width
	}
	if !negative {
		return magnitude.BitLen() < width
	}
	if magnitude.BitLen() < width {
		return true
	}
	return magnitude.BitLen() == width &&
		magnitude.Bit(width-1) != 0 &&
		magnitude.TrailingZeroBits() == uint(width-1)
}

func floatingPointToBitVectorUint64(
	mode uint8,
	width, exponentBits, significandBits int,
	raw uint64,
	signed bool,
) (BitVectorValue, bool) {
	rounded := floatingPointRoundToIntegralUint64(
		mode, exponentBits, significandBits, raw,
	)
	roundedRaw, _ := FloatingPointBits(rounded).Uint64()
	fractionBits := significandBits - 1
	exponentMask := uint64(1)<<exponentBits - 1
	exponentField := roundedRaw >> fractionBits & exponentMask
	fractionMask := uint64(1)<<fractionBits - 1
	magnitude := roundedRaw & fractionMask
	negative := roundedRaw>>(exponentBits+significandBits-1) != 0
	if exponentField == 0 {
		return NewBitVectorUint64(width, 0), true
	}
	magnitude |= uint64(1) << fractionBits
	bias := int64(uint64(1)<<(exponentBits-1) - 1)
	scale := int64(exponentField) - bias - int64(fractionBits)
	if scale >= 0 {
		if scale >= 64 || bits.Len64(magnitude)+int(scale) > 64 {
			return NewBitVectorUint64(width, 0), false
		}
		magnitude <<= uint(scale)
	} else {
		shift := -scale
		if shift >= 64 {
			magnitude = 0
		} else {
			magnitude >>= uint(shift)
		}
	}
	valid := false
	if magnitude == 0 {
		valid = true
	} else if !signed {
		valid = !negative && (width == 64 || magnitude < uint64(1)<<width)
	} else if negative {
		valid = width == 64 && magnitude <= uint64(1)<<63 ||
			width < 64 && magnitude <= uint64(1)<<(width-1)
	} else {
		valid = width == 64 && magnitude <= uint64(1)<<63-1 ||
			width < 64 && magnitude < uint64(1)<<(width-1)
	}
	if !valid {
		return NewBitVectorUint64(width, 0), false
	}
	if negative {
		magnitude = -magnitude
	}
	return NewBitVectorUint64(width, magnitude), true
}
