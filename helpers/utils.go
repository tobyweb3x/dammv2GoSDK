package helpers

import (
	"dammv2GoSDK/constants"
	"fmt"
	"math/big"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/shopspring/decimal"
)

// GetMinAmountWithSlippage calculates the minimum amount receivable after slippage is applied.
// Returns the minimum amount receivable after applying the slippage.
//
// - amount: The original amount of tokens (as *big.Int).
//
// - rate: The slippage rate as a float64 percentage (e.g., 0.5 for 0.5%).
// Example:
//
//	GetMinAmountWithSlippage(big.NewInt(100000), 0.5) returns 99500 for 0.5% slippage.
func GetMinAmountWithSlippage(amount *big.Int, rate float64) *big.Int {
	slippage := new(big.Int).SetUint64(uint64((100 - rate) / 100 * constants.BasisPointMax))
	return new(big.Int).Div(
		new(big.Int).Mul(amount, slippage),
		big.NewInt(constants.BasisPointMax),
	)
}

func BigIntToUint128(b *big.Int) (ag_binary.Uint128, error) {
	if b.Sign() < 0 {
		return ag_binary.Uint128{}, fmt.Errorf("value must be unsigned")
	}

	if b.BitLen() > 128 {
		return ag_binary.Uint128{}, fmt.Errorf("value %s exceeds 128 bits", b.String())
	}

	var buf [16]byte
	b.FillBytes(buf[:]) // zero-pads on the left

	ag_binary.ReverseBytes(buf[:])

	var u ag_binary.Uint128
	if err := u.UnmarshalWithDecoder(ag_binary.NewBinDecoder(buf[:])); err != nil {
		return ag_binary.Uint128{}, err
	}
	return u, nil
}

// Must helper
func MustBigIntToUint128(b *big.Int) ag_binary.Uint128 {
	v, err := BigIntToUint128(b)
	if err != nil {
		panic(fmt.Errorf("cannot fit big.Int into Uint128: %s", err.Error()))
	}
	return v
}

// GetPriceImpact calculates the percentage difference between the current and next sqrt prices.
// TODO: take a another look.
func GetPriceImpact(nextSqrtPrice, currentSqrtPrice *big.Int) float64 {
	// (sqrtA^2 - sqrtB^2).Abs() * 100 / (sqrtB^2)

	// Compute (sqrtPrice)^2
	sqrtA2 := new(big.Int).Mul(nextSqrtPrice, nextSqrtPrice)
	sqrtB2 := new(big.Int).Mul(currentSqrtPrice, currentSqrtPrice)

	// Compute absolute difference
	diff := new(big.Int).Sub(sqrtA2, sqrtB2)
	if diff.Sign() < 0 {
		diff.Neg(diff)
	}

	// Convert to decimal for division and percentage
	diffDecimal := decimal.NewFromBigInt(diff, 0)
	sqrtB2Decimal := decimal.NewFromBigInt(sqrtB2, 0)

	// (|sqrtA² - sqrtB²| / sqrtB²) * 100
	priceImpact := diffDecimal.Div(sqrtB2Decimal).Mul(decimal.NewFromInt(100))

	return priceImpact.InexactFloat64()
}
