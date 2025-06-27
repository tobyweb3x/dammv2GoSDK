package maths

import (
	"dammv2GoSDK/types"
	"math/big"
)

func MulDiv(x, y, denominator *big.Int, rounding types.Rounding) *big.Int {
	product := new(big.Int).Mul(x, y)

	div, mod := new(big.Int).DivMod(product, denominator, new(big.Int))

	if rounding == types.RoundingUp && mod.Sign() != 0 {
		div.Add(div, big.NewInt(1))
	}

	return div
}
