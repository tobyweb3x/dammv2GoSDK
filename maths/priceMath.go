package maths

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// decimalSqrt computes the square root of a decimal using big.Float for precision.
func decimalSqrt(d decimal.Decimal) (decimal.Decimal, error) {
	f, ok := new(big.Float).SetString(d.String())
	if !ok {
		return decimal.Zero, fmt.Errorf("bad decimal: %s", d)
	}
	f.SetPrec(256)
	root := new(big.Float).Sqrt(f) // √
	s := root.Text('f', -1)
	return decimal.NewFromString(s)
}

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

	amountADec := decimal.NewFromBigInt(tokenAAmount, 0)
	amountBDec := decimal.NewFromBigInt(tokenBAmount, 0)
	scale := decimal.NewFromBigInt(
		new(big.Int).Lsh(big.NewInt(1), 64), 0) // 2^64

	minSqrtPriceDec := decimal.NewFromBigInt(minSqrtPrice, 0).Div(scale)
	maxSqrtPriceDec := decimal.NewFromBigInt(maxSqrtPrice, 0).Div(scale)

	x := decimal.NewFromInt(1).Div(maxSqrtPriceDec)
	y := amountBDec.Div(amountADec)
	xy := x.Mul(y)

	paMinusXY := minSqrtPriceDec.Sub(xy)
	xyMinusPa := xy.Sub(minSqrtPriceDec)

	discriminant := xyMinusPa.Mul(xyMinusPa).Add(y.Mul(decimal.NewFromInt(4)))

	// sqrt_discriminant = √discriminant
	sqrtDiscriminant, err := decimalSqrt(discriminant)
	if err != nil {
		return nil, err
	}

	result := paMinusXY.Add(sqrtDiscriminant).Div(decimal.NewFromInt(2)).Mul(scale)

	resultInt := result.Floor().BigInt()
	return resultInt, nil
}
