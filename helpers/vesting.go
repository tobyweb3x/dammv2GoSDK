package helpers

import (
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"math/big"
)

// IsVestingComplete checks if a vesting schedule is ready for full release.
func IsVestingComplete(
	vestingData *cp_amm.VestingAccount,
	currentPoint *big.Int,
) bool {
	endPoint := new(big.Int).Add(
		new(big.Int).SetUint64(vestingData.CliffPoint),
		new(big.Int).Mul(
			new(big.Int).SetUint64(vestingData.PeriodFrequency),
			new(big.Int).SetUint64(uint64(vestingData.NumberOfPeriod))),
	)

	return currentPoint.Cmp(endPoint) >= 0
}
