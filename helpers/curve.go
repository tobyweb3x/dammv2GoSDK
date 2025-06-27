package helpers

import (
	"dammv2GoSDK/constants"
	"dammv2GoSDK/types"
	"math/big"
)

// GetNextSqrtPrice
//
// aToB
//
// √P' = √P * L / (L + Δx*√P)
//
// bToA
//
// √P' = √P + Δy / L
func GetNextSqrtPrice(
	amount, sqrtPrice, liquidity *big.Int,
	aToB bool,
) *big.Int {

	if aToB {
		product := new(big.Int).Mul(amount, sqrtPrice)
		denominator := new(big.Int).Add(liquidity, product)
		numerator := new(big.Int).Mul(liquidity, sqrtPrice)
		return new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Sub(denominator, big.NewInt(1)),
				numerator,
			),
			denominator,
		)
	}

	quotient := new(big.Int).Div(
		new(big.Int).Rsh(amount, constants.ScaleOffset*2),
		liquidity,
	)
	return new(big.Int).Add(sqrtPrice, quotient)
}

// GetLiquidityDeltaFromAmountA
//
// lowerSqrtPrice - current sqrt price
//
// upperSqrtPrice -  max sqrt price
//
// Δa = L * (1 / √P_lower - 1 / √P_upper)
//
// Δa = L * (√P_upper - √P_lower) / (√P_upper * √P_lower)
//
// L = Δa * √P_upper * √P_lower / (√P_upper - √P_lower)
func GetLiquidityDeltaFromAmountA(
	amountA, lowerSqrtPrice, upperSqrtPrice *big.Int,
) *big.Int {
	product := new(big.Int).Mul(
		upperSqrtPrice,
		new(big.Int).Mul(lowerSqrtPrice, amountA),
	) // Q128.128
	denominator := new(big.Int).Sub(upperSqrtPrice, lowerSqrtPrice) // Q64.64

	return new(big.Int).Div(product, denominator)
}

// GetLiquidityDeltaFromAmountB
//
// lowerSqrtPrice - min sqrt price
//
// upperSqrtPrice -  current sqrt price
//
// Δb = L (√P_upper - √P_lower)
// L = Δb / (√P_upper - √P_lower)
func GetLiquidityDeltaFromAmountB(
	amountB, lowerSqrtPrice, upperSqrtPrice *big.Int,
) *big.Int {
	product := new(big.Int).Lsh(amountB, 128)
	denominator := new(big.Int).Sub(upperSqrtPrice, lowerSqrtPrice)

	return new(big.Int).Div(product, denominator)
}

// GetAmountAFromLiquidityDelta
//
// L = Δa * √P_upper * √P_lower / (√P_upper - √P_lower)
//
// Δa = L * (√P_upper - √P_lower) / √P_upper * √P_lower
func GetAmountAFromLiquidityDelta(
	liquidity, currentSqrtPrice, maxSqrtPrice *big.Int,
	rounding types.Rounding,
) *big.Int {
	product := new(big.Int).Mul(
		liquidity,
		new(big.Int).Sub(maxSqrtPrice, currentSqrtPrice),
	) // Q128.128

	denominator := new(big.Int).Mul(currentSqrtPrice, maxSqrtPrice) // Q128.128

	if rounding == types.RoundingUp {
		return new(big.Int).Div(
			new(big.Int).Add(product, new(big.Int).Sub(denominator, big.NewInt(1))),
			denominator,
		)
	}

	return new(big.Int).Div(product, denominator)
}

// GetAmountBFromLiquidityDelta
//
// L = Δb / (√P_upper - √P_lower)
//
// Δb = L * (√P_upper - √P_lower)
func GetAmountBFromLiquidityDelta(
	liquidity, currentSqrtPrice, minSqrtPrice *big.Int,
	rounding types.Rounding,
) *big.Int {
	one := new(big.Int).Lsh(big.NewInt(1), 128) // 1 << 128
	deltaPrice := new(big.Int).Sub(currentSqrtPrice, minSqrtPrice)
	result := new(big.Int).Mul(liquidity, deltaPrice) // Q128

	if rounding == types.RoundingUp {
		return new(big.Int).Div(
			new(big.Int).Add(result, new(big.Int).Sub(one, big.NewInt(1))),
			one,
		)
	}

	return new(big.Int).Rsh(result, 128) // result >> 128
}
