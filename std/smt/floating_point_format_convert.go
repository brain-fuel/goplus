package smt

// floatingPointConvertFormat implements SMT-LIB's floating-point to
// floating-point to_fp overload. Finite inputs are decoded as an exact binary
// integer and scale, then rounded once into the target format.
func floatingPointConvertFormat(
	mode uint8,
	targetExponentBits, targetSignificandBits int,
	value FloatingPointValue,
) FloatingPointValue {
	if mode < 1 || mode > 5 {
		panic("smt: invalid floating-point rounding mode")
	}
	if targetExponentBits < 2 || targetSignificandBits < 2 {
		panic("smt: invalid floating-point format")
	}
	negative := FloatingPointBits(value).Bit(
		FloatingPointExponentBits(value) +
			FloatingPointSignificandBits(value) - 1,
	)
	if FloatingPointIsNaN(value) {
		return FloatingPointNaN(targetExponentBits, targetSignificandBits)
	}
	if FloatingPointIsInfinite(value) {
		if negative {
			return FloatingPointNegativeInfinity(
				targetExponentBits, targetSignificandBits,
			)
		}
		return FloatingPointPositiveInfinity(
			targetExponentBits, targetSignificandBits,
		)
	}
	if FloatingPointIsZero(value) {
		return floatingPointZero(
			targetExponentBits, targetSignificandBits, negative,
		)
	}
	sourceExponentBits := FloatingPointExponentBits(value)
	sourceSignificandBits := FloatingPointSignificandBits(value)
	if sourceExponentBits+sourceSignificandBits <= 64 &&
		targetExponentBits+targetSignificandBits <= 64 {
		if raw, inline := FloatingPointBits(value).Uint64(); inline {
			fractionBits := sourceSignificandBits - 1
			fractionMask := uint64(1)<<fractionBits - 1
			magnitude := raw & fractionMask
			exponentMask := uint64(1)<<sourceExponentBits - 1
			exponentField := raw >> fractionBits & exponentMask
			bias := int64(uint64(1)<<(sourceExponentBits-1) - 1)
			unbiased := int64(1) - bias
			if exponentField != 0 {
				unbiased = int64(exponentField) - bias
				magnitude |= uint64(1) << fractionBits
			}
			return floatingPointRoundExactBinaryUint64(
				mode, targetExponentBits, targetSignificandBits,
				negative, magnitude, unbiased-int64(fractionBits),
			)
		}
	}
	finite := decodeFloatingPointFinite(value)
	return floatingPointRoundExactBinary(
		mode, targetExponentBits, targetSignificandBits,
		finite.negative, finite.magnitude, finite.scale,
	)
}
