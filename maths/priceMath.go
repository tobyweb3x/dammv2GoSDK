package maths

import (
	"errors"
	"math"
	"math/big"
)

// CalculateInitSqrtPrice calculates the initial sqrt price based on token amounts and price bounds.
//
// a = L * (1/s - 1/pb)
//
// b = L * (s - pa)
//
// b/a = (s - pa) / (1/s - 1/pb)
//
// With: x = 1 / pb and y = b/a
//
// => s ^ 2 + s * (-pa + x * y) - y = 0
//
// s = [(pa - xy) + √((xy - pa)² + 4y)]/2
func CalculateInitSqrtPrice(
	tokenAAmount, tokenBAmount, minSqrtPrice, maxSqrtPrice *big.Int,
) (*big.Int, error) {
	if tokenAAmount.Sign() == 0 || tokenBAmount.Sign() == 0 {
		return nil, errors.New("amount cannot be zero")
	}

	amountADecimal, amountBDecimal :=
		new(big.Float).SetInt(tokenAAmount), new(big.Float).SetInt(tokenBAmount)

	minSqrtPriceDecimal := new(big.Float).Quo(
		new(big.Float).SetInt(minSqrtPrice), big.NewFloat(math.Pow(2, 64)),
	)

	maxSqrtPriceDecimal := new(big.Float).Quo(
		new(big.Float).SetInt(maxSqrtPrice), big.NewFloat(math.Pow(2, 64)),
	)

	x, y :=
		new(big.Float).Quo(big.NewFloat(1), maxSqrtPriceDecimal),
		new(big.Float).Quo(amountBDecimal, amountADecimal)
	xy := new(big.Float).Mul(x, y)

	paMinusXY, xyMinusPa :=
		new(big.Float).Sub(minSqrtPriceDecimal, xy),
		new(big.Float).Sub(xy, minSqrtPriceDecimal)

	fourY := new(big.Float).Mul(big.NewFloat(4), y)
	discriminant := new(big.Float).Add(
		new(big.Float).Mul(xyMinusPa, xyMinusPa),
		fourY,
	)

	// sqrt_discriminant = √discriminant
	discriminant.Sqrt(discriminant)

	result := new(big.Float).Mul(
		new(big.Float).Quo(new(big.Float).Add(discriminant, paMinusXY), big.NewFloat(2)),
		big.NewFloat(math.Pow(2, 64)),
	)

	r, _ := result.Int(nil)
	return r, nil
}
