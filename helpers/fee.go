package helpers

import (
	"dammv2GoSDK/constants"
	"dammv2GoSDK/maths"
	"dammv2GoSDK/types"
	"math/big"
	"reflect"
)

// GetFeeNumerator calculates the fee numerator based on current market conditions and fee schedule configuration, returns calculated fee numerator, capped at MAX_FEE_NUMERATOR.
//
// currentPoint - The current price point in the liquidity curve.
//
// activationPoint - The price point at which the fee schedule is activated.
//
// numberOfPeriod - The total number of periods in the fee schedule.
//
// periodFrequency - The frequency at which periods change.
//
// feeSchedulerMode - The mode determining how fees are calculated (0 = constant, 1 = linear, etc.).
//
// cliffFeeNumerator - The initial fee numerator at the cliff point.
//
// reductionFactor - The factor by which fees are reduced in each period.
//
// dynamicFeeParams - Optional parameters for dynamic fee calculation.
//
// dynamicFeeParams.volatilityAccumulator - Measure of accumulated market volatility.
//
// dynamicFeeParams.binStep - Size of price bins in the liquidity distribution.
//
//	dynamicFeeParams.variableFeeControl - Parameter controlling the impact of volatility.
func GetFeeNumerator(
	currentPoint uint64,
	activationPoint *big.Int,
	numberOfPeriod uint16,
	periodFrequency *big.Int,
	feeSchedulerMode types.FeeSchedulerMode,
	cliffFeeNumerator *big.Int,
	reductionFactor *big.Int,
	dynamicFeeParams types.DynamicFeeParams,
) *big.Int {

	if periodFrequency.Cmp(big.NewInt(0)) == 0 ||
		new(big.Int).SetUint64(currentPoint).Cmp(activationPoint) == -1 {
		return cliffFeeNumerator
	}

	hold := new(big.Int).Div(
		new(big.Int).Sub(new(big.Int).SetUint64(currentPoint), activationPoint),
		periodFrequency,
	)

	period := new(big.Int).SetUint64(uint64(numberOfPeriod))
	if period.Cmp(hold) == -1 {
		period = hold
	}

	feeNumerator := GetBaseFeeNumerator(
		feeSchedulerMode,
		cliffFeeNumerator,
		period,
		reductionFactor,
	)
	dynamicFeeNumberator := big.NewInt(0)
	if !reflect.ValueOf(dynamicFeeParams).IsZero() {
		dynamicFeeNumberator = GetDynamicFeeNumerator(
			dynamicFeeParams.VolatilityAccumulator,
			new(big.Int).SetUint64(uint64(dynamicFeeParams.BinStep)),
			new(big.Int).SetUint64(uint64(dynamicFeeParams.VariableFeeControl)),
		)
	}

	feeNumerator = new(big.Int).Add(feeNumerator, dynamicFeeNumberator)

	if feeNumerator.Cmp(big.NewInt(constants.MaxFeeNumerator)) > 0 {
		return big.NewInt(constants.MaxFeeNumerator)
	}
	return feeNumerator
}

// GetBaseFeeNumerator
//
// # Fee scheduler
//
// Linear: cliffFeeNumerator - period * reductionFactor
//
// Exponential: cliffFeeNumerator * (1 -reductionFactor/BASIS_POINT_MAX)^period
func GetBaseFeeNumerator(
	feeSchedulerMode types.FeeSchedulerMode,
	cliffFeeNumerator *big.Int,
	period *big.Int,
	reductionFactor *big.Int,
) *big.Int {

	if feeSchedulerMode == types.Linear {
		return new(big.Int).Sub(
			cliffFeeNumerator,
			new(big.Int).Mul(reductionFactor, period),
		)
	}

	bps := new(big.Int).Div(
		new(big.Int).Lsh(reductionFactor, constants.ScaleOffset),
		big.NewInt(constants.BasisPointMax),
	)
	base := new(big.Int).Sub(maths.One, bps)
	result := maths.Pow(base, period)

	return new(big.Int).Rsh(new(big.Int).Mul(cliffFeeNumerator, result), constants.ScaleOffset)
}

// GetDynamicFeeNumerator calculates the dynamic fee numerator based on market volatility metrics, returns the calculated dynamic fee numerator.
//
// volatilityAccumulator - A measure of accumulated market volatility.
//
// binStep - The size of price bins in the liquidity distribution.
//
// variableFeeControl - Parameter controlling the impact of volatility on fees.
func GetDynamicFeeNumerator(
	volatilityAccumulator, binStep, variableFeeControl *big.Int,
) *big.Int {
	if variableFeeControl.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	}

	squareVfaBin := new(big.Int).Exp(
		new(big.Int).Mul(volatilityAccumulator, binStep),
		big.NewInt(2),
		nil,
	)

	vFee := new(big.Int).Mul(variableFeeControl, squareVfaBin)

	return new(big.Int).Div(
		new(big.Int).Add(vFee, big.NewInt(99_999_999_999)),
		big.NewInt(100_000_000_000),
	)
}

//	GetFeeMode determines the fee mode based on the swap direction and fee collection configuration
//
// collectFeeMode - The fee collection mode (e.g., OnlyB, BothToken).
//
// btoA - Boolean indicating if the swap is from token B to token A.
func GetFeeMode(collectFeeMode types.CollectFeeMode, bToA bool) types.FeeMode {
	feeOnInput := bToA && collectFeeMode == types.OnlyB
	feesOnTokenA := bToA && collectFeeMode == types.BothToken

	return types.FeeMode{
		FeeOnInput:   feeOnInput,
		FeesOnTokenA: feesOnTokenA,
	}
}

// GetTotalFeeOnAmount calculates the total fee amount based on the transaction amount and fee numerator.
//
// amount - The transaction amount.
//
// tradeFeeNumerator - The fee numerator to apply.
func GetTotalFeeOnAmount(amount, tradeFeeNumerator *big.Int) *big.Int {
	return maths.MulDiv(
		amount,
		tradeFeeNumerator,
		big.NewInt(constants.FeeDenominator),
		types.RoundingUp,
	)
}

// GetSwapAmount calculates the output amount and fees for a swap operation in a concentrated liquidity pool.
// Returns a struct containing the actual output amount after fees and the total fee amount.
//
// inAmount - The input amount of tokens the user is swapping.
//
// sqrtPrice - The current square root price of the pool.
//
// liquidity - The current liquidity available in the pool.
//
// tradeFeeNumerator - The fee numerator used to calculate trading fees.
//
// aToB - Direction of the swap: true for token A to token B, false for token B to token A.
//
// collectFeeMode - Determines how fees are collected (0: both tokens, 1: only token B).
func GetSwapAmount(
	inAmount, sqrtPrice, liquidity, tradeFeeNumerator *big.Int,
	aToB bool, collectFeeMode types.CollectFeeMode,
) *struct{ AmountOut, TotalFee, NextSqrtPrice *big.Int } {

	feeMode, actualInAmount, totalFee := GetFeeMode(collectFeeMode, !aToB),
		inAmount, big.NewInt(0)
	if feeMode.FeeOnInput {
		totalFee = GetTotalFeeOnAmount(inAmount, tradeFeeNumerator)
		actualInAmount = new(big.Int).Sub(inAmount, totalFee)
	}
	nextSqrtPrice := GetNextSqrtPrice(
		actualInAmount,
		sqrtPrice,
		liquidity,
		aToB,
	)

	// calculate the output amount based on swap direction
	outAmount := GetAmountAFromLiquidityDelta(
		liquidity,
		sqrtPrice,
		nextSqrtPrice,
		types.RoundingDown,
	)
	if aToB {
		outAmount = GetAmountBFromLiquidityDelta(
			liquidity,
			sqrtPrice,
			nextSqrtPrice,
			types.RoundingDown,
		)
	}

	// apply fees to output amount if fee is taken on output
	amountOut := outAmount
	if !feeMode.FeeOnInput {
		totalFee = GetTotalFeeOnAmount(outAmount, tradeFeeNumerator)
		amountOut = new(big.Int).Sub(outAmount, totalFee)
	}

	return &struct {
		AmountOut     *big.Int
		TotalFee      *big.Int
		NextSqrtPrice *big.Int
	}{
		AmountOut: amountOut, TotalFee: totalFee, NextSqrtPrice: nextSqrtPrice,
	}
}
