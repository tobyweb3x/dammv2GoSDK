package maths

import (
	"dammv2GoSDK/constants"
	"math/big"
)

var (
	// One
	//   One = new(big.Int).Lsh(big.NewInt(1), constants.ScaleOffset)
	One = new(big.Int).Lsh(big.NewInt(1), constants.ScaleOffset)
	Pow = func(base, exp *big.Int) *big.Int {
		return nil
	}
)
