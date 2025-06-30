package maths

import (
	"errors"
	"math/big"

	"github.com/shopspring/decimal"
)

// decimalSqrt computes the square root of a decimal using big.Float for precision.
func decimalSqrt(d decimal.Decimal) decimal.Decimal {
	f, _ := new(big.Float).SetString(d.String())
	sqrt := new(big.Float).Sqrt(f)
	resultStr := sqrt.Text('f', -1)
	result, _ := decimal.NewFromString(resultStr)
	return result
}

// CalculateInitSqrtPrice calculates the initial sqrt price based on token amounts and price bounds.
func CalculateInitSqrtPrice(
	tokenAAmount, tokenBAmount, minSqrtPrice, maxSqrtPrice *big.Int,
) (*big.Int, error) {
	if tokenAAmount.Sign() == 0 || tokenBAmount.Sign() == 0 {
		return nil, errors.New("amount cannot be zero")
	}

	amountADec := decimal.NewFromBigInt(tokenAAmount, 0)
	amountBDec := decimal.NewFromBigInt(tokenBAmount, 0)
	scale := decimal.NewFromInt(2).Pow(decimal.NewFromInt(64))

	minSqrtPriceDec := decimal.NewFromBigInt(minSqrtPrice, 0).Div(scale)
	maxSqrtPriceDec := decimal.NewFromBigInt(maxSqrtPrice, 0).Div(scale)

	x := decimal.NewFromInt(1).Div(maxSqrtPriceDec)
	y := amountBDec.Div(amountADec)
	xy := x.Mul(y)

	paMinusXY := minSqrtPriceDec.Sub(xy)
	xyMinusPa := xy.Sub(minSqrtPriceDec)

	discriminant := xyMinusPa.Mul(xyMinusPa).Add(y.Mul(decimal.NewFromInt(4)))
	sqrtDiscriminant := decimalSqrt(discriminant)

	result := paMinusXY.Add(sqrtDiscriminant).Div(decimal.NewFromInt(2)).Mul(scale)

	resultInt := result.Floor().BigInt()
	return resultInt, nil
}
