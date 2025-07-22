package maths

import (
	"dammv2GoSDK/types"
	"math/big"
)

func MulDiv(x, y, denominator *big.Int, rounding types.Rounding) *big.Int {
	div, mod := new(big.Int).QuoRem(
		new(big.Int).Mul(x, y),
		denominator,
		new(big.Int))

	if rounding == types.RoundingUp && mod.Sign() != 0 {
		return div.Add(div, big.NewInt(1))
	}

	return div
}
