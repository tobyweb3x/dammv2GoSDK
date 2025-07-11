package helpers

import (
	"math/big"

	"github.com/gagliardetto/solana-go/programs/token"
)

func CalculateTransferFeeExcludedAmount(
	transferFeeIncludedAmount *big.Int,
	mint token.Mint,
	currentEpoch uint64,
) struct{ Amount, TransferFee *big.Int } {
	return struct {
		Amount      *big.Int
		TransferFee *big.Int
	}{
		Amount:      big.NewInt(0),
		TransferFee: big.NewInt(0),
	}
}

func CalculateTransferFeeIncludedAmount(
	transferFeeExcludedAmount *big.Int,
	mint token.Mint,
	currentEpoch uint64,
) struct{ Amount, TransferFee *big.Int } {
	return struct {
		Amount      *big.Int
		TransferFee *big.Int
	}{
		Amount:      big.NewInt(0),
		TransferFee: big.NewInt(0),
	}
}
