package maths

import (
	"dammv2GoSDK/constants"
	"math/big"
)

var (
	// MaxExponential
	//  MaxExponential = big.NewInt(0x80000) i.e 524288
	MaxExponential = big.NewInt(0x80000)
	// One
	//   One = new(big.Int).Lsh(big.NewInt(1), constants.ScaleOffset)
	One = new(big.Int).Lsh(big.NewInt(1), constants.ScaleOffset)

	// MAX = (2^128) - 1
	Max = new(big.Int).Sub(
		new(big.Int).Exp(
			big.NewInt(2),
			big.NewInt(128),
			nil,
		),
		big.NewInt(1),
	)
	Pow = func(base, exp *big.Int) *big.Int {
		if exp.Sign() == 0 {
			return One
		}

		invert := exp.Sign() < 0
		if invert {
			exp = new(big.Int).Abs(exp)
		}

		if exp.Cmp(MaxExponential) > 0 {
			return big.NewInt(0)
		}

		result := new(big.Int).Set(One)
		squaredBase := new(big.Int).Set(base)

		if squaredBase.Cmp(One) >= 0 {
			squaredBase = new(big.Int).Div(Max, squaredBase)
			invert = !invert
		}

		expBits := exp.BitLen()
		for i := range expBits {
			if exp.Bit(i) == 1 {
				result.Mul(result, squaredBase)
				result.Rsh(result, constants.ScaleOffset)
			}
			
			squaredBase.Mul(squaredBase, squaredBase)
			squaredBase.Rsh(squaredBase, constants.ScaleOffset)

		}

		if result.Sign() == 0 {
			return big.NewInt(0)
		}

		if invert {
			result = new(big.Int).Div(Max, result)
		}

		return result
	}
)
