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

func GetAvailableVestingLiquidity(
	vesting *cp_amm.VestingAccount,
	currentPoint *big.Int,
) *big.Int {

	if currentPoint.Cmp(new(big.Int).SetUint64(vesting.CliffPoint)) < 0 {
		return big.NewInt(0)
	}

	if vesting.PeriodFrequency == 0 {
		return vesting.CliffUnlockLiquidity.BigInt()
	}

	passedPeriod := new(big.Int).Div(
		new(big.Int).Sub(currentPoint, new(big.Int).SetUint64(vesting.CliffPoint)),
		new(big.Int).SetUint64(vesting.PeriodFrequency),
	)

	if numberOfPeriodBigInt := new(big.Int).SetUint64(uint64(vesting.NumberOfPeriod)); passedPeriod.Cmp(numberOfPeriodBigInt) > 0 {
		passedPeriod = numberOfPeriodBigInt
	}

	// total unlocked liquidity: cliff + (periods * per_period)
	unlockedLiquidity := new(big.Int).Add(
		vesting.CliffUnlockLiquidity.BigInt(),
		new(big.Int).Mul(passedPeriod, vesting.LiquidityPerPeriod.BigInt()),
	)

	availableReleasingLiquidity := new(big.Int).Sub(
		unlockedLiquidity,
		vesting.TotalReleasedLiquidity.BigInt(),
	)

	return availableReleasingLiquidity
}
