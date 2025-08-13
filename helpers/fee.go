package helpers

import (
	"dammv2GoSDK/constants"
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"dammv2GoSDK/maths"
	"dammv2GoSDK/types"
	"errors"
	"fmt"
	"math"
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

	hold := new(big.Int).Quo(
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

	if feeSchedulerMode == types.FeeSchedulerModeLinear {
		return new(big.Int).Sub(
			cliffFeeNumerator,
			new(big.Int).Mul(reductionFactor, period),
		)
	}

	bps := new(big.Int).Quo(
		new(big.Int).Lsh(reductionFactor, constants.ScaleOffset),
		big.NewInt(constants.BasisPointMax),
	)

	base := new(big.Int).Sub(maths.One, bps)
	result := maths.Pow(base, period)

	return new(big.Int).Rsh(
		new(big.Int).Mul(cliffFeeNumerator, result),
		constants.ScaleOffset,
	)
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

	return new(big.Int).Quo(
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
	feeOnInput := bToA && collectFeeMode == types.CollectFeeModeOnlyB
	feesOnTokenA := bToA && collectFeeMode == types.CollectFeeModeBothToken
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

func GetBaseFeeParams(
	maxBaseFeeBps, minBaseFeeBps uint64,
	feeSchedulerMode types.FeeSchedulerMode,
	numberOfPeriod uint16, totalDuration uint64,
) (cp_amm.BaseFeeParameters, error) {
	if maxBaseFeeBps == minBaseFeeBps {
		if numberOfPeriod != 0 || totalDuration != 0 {
			return cp_amm.BaseFeeParameters{}, errors.New("numberOfPeriod and totalDuration must both be zero")
		}

		cliffFeeNumerator := BpsToFeeNumerator(maxBaseFeeBps)
		if !cliffFeeNumerator.IsUint64() {
			return cp_amm.BaseFeeParameters{}, fmt.Errorf("cannot fit cliffFeeNumerator(%s) into uint64", cliffFeeNumerator)
		}
		return cp_amm.BaseFeeParameters{CliffFeeNumerator: cliffFeeNumerator.Uint64()}, nil
	}

	if numberOfPeriod <= 0 {
		return cp_amm.BaseFeeParameters{}, errors.New("total periods must be greater than zero")
	}

	if hold := FeeNumeratorToBps(big.NewInt(constants.MaxFeeNumerator)); maxBaseFeeBps > hold {
		return cp_amm.BaseFeeParameters{}, fmt.Errorf("maxBaseFeeBps %d bps exceeds maximum allowed value of %d bps",
			maxBaseFeeBps, hold)
	}

	if minBaseFeeBps > maxBaseFeeBps {
		return cp_amm.BaseFeeParameters{}, errors.New("minBaseFee bps must be less than or equal to maxBaseFee bps")
	}

	if numberOfPeriod == 0 || totalDuration == 0 {
		return cp_amm.BaseFeeParameters{}, errors.New("numberOfPeriod and totalDuration must both greater than zero")
	}

	maxBaseFeeNumerator, minBaseFeeNumerator, periodFrequency :=
		BpsToFeeNumerator(maxBaseFeeBps),
		BpsToFeeNumerator(minBaseFeeBps),
		new(big.Int).SetUint64(totalDuration/uint64(numberOfPeriod))

	if !maxBaseFeeNumerator.IsUint64() || !periodFrequency.IsUint64() {
		return cp_amm.BaseFeeParameters{}, fmt.Errorf("either maxBaseFeeNumerator(%s) or periodFrequency(%s) cannot fit into uint64",
			maxBaseFeeNumerator, periodFrequency)
	}

	if feeSchedulerMode == types.FeeSchedulerModeLinear {
		reductionFactor := new(big.Int).Quo(
			new(big.Int).Sub(maxBaseFeeNumerator, minBaseFeeNumerator),
			new(big.Int).SetUint64(uint64(numberOfPeriod)),
		)

		if !reductionFactor.IsUint64() {
			return cp_amm.BaseFeeParameters{}, fmt.Errorf("cannot fit reductionFactor(%s) into uint64", reductionFactor)
		}

		return cp_amm.BaseFeeParameters{
			CliffFeeNumerator: maxBaseFeeNumerator.Uint64(),
			NumberOfPeriod:    numberOfPeriod,
			PeriodFrequency:   periodFrequency.Uint64(),
			ReductionFactor:   reductionFactor.Uint64(),
			FeeSchedulerMode:  uint8(feeSchedulerMode),
		}, nil

	}

	ratio := float64(minBaseFeeNumerator.Uint64()) / float64(maxBaseFeeNumerator.Uint64())
	decayBase := math.Pow(ratio, 1.0/float64(numberOfPeriod))
	reduction := float64(constants.BasisPointMax) * (1 - decayBase)

	reductionFactor := big.NewInt(int64(reduction))

	return cp_amm.BaseFeeParameters{
		CliffFeeNumerator: maxBaseFeeNumerator.Uint64(),
		NumberOfPeriod:    numberOfPeriod,
		PeriodFrequency:   periodFrequency.Uint64(),
		ReductionFactor:   reductionFactor.Uint64(),
		FeeSchedulerMode:  uint8(feeSchedulerMode),
	}, nil
}

// Converts basis points (bps) to a fee numerator and returns the equivalent fee numerator.
// 1 bps = 0.01% = 0.0001 in decimal
//
// bps - The value in basis points [1-10_000]
func BpsToFeeNumerator(bps uint64) *big.Int {
	return new(big.Int).Quo(
		new(big.Int).SetUint64(bps*constants.FeeDenominator),
		big.NewInt(constants.BasisPointMax),
	)
}

func FeeNumeratorToBps(feeNumerator *big.Int) uint64 {
	return new(big.Int).Quo(
		new(big.Int).Mul(feeNumerator, big.NewInt(constants.BasisPointMax)),
		big.NewInt(constants.FeeDenominator),
	).Uint64()
}

func GetDynamicFeeParams(
	baseFeeBps uint64, maxPriceChangeBps uint64,
) (types.DynamicFee, error) {
	if maxPriceChangeBps == 0 {
		maxPriceChangeBps = constants.MaxPriceChangeBpsDefault // default 15%
	}

	if maxPriceChangeBps > constants.MaxPriceChangeBpsDefault {
		return types.DynamicFee{}, fmt.Errorf("maxPriceChangeBps (%d bps) must be less than or equal to %d", maxPriceChangeBps, constants.MaxPriceChangeBpsDefault)
	}

	priceRatio := new(big.Float).Add(
		new(big.Float).Quo(
			new(big.Float).SetUint64(maxPriceChangeBps),
			new(big.Float).SetUint64(constants.BasisPointMax),
		),
		big.NewFloat(1),
	)

	twoTo64, _ := new(big.Float).SetPrec(256).SetString("18446744073709551616") // 2^64
	sqrtPriceRatio := new(big.Float).Mul(
		new(big.Float).Sqrt(priceRatio),
		twoTo64,
	)

	if !sqrtPriceRatio.IsInt() {
		return types.DynamicFee{}, errors.New("sqrtPriceRatio cannot be roundedOff")
	}

	sqrtPriceRatioFloored, _ := sqrtPriceRatio.Int(nil)
	deltaBinId := new(big.Int).Mul(
		new(big.Int).Quo(
			new(big.Int).Sub(sqrtPriceRatioFloored, maths.One),
			constants.BinStepBpsU128Default,
		),
		big.NewInt(2),
	)

	maxVolatilityAccumulator := new(big.Int).Mul(deltaBinId, big.NewInt(constants.BasisPointMax))

	squareVfaBin := new(big.Int).Mul(maxVolatilityAccumulator, big.NewInt(constants.BinStepBpsDefault))
	squareVfaBin.Mul(squareVfaBin, squareVfaBin)

	baseFeeNumerator := BpsToFeeNumerator(baseFeeBps)

	maxDynamicFeeNumerator := new(big.Int).Quo(
		new(big.Int).Mul(baseFeeNumerator, big.NewInt(20)), // default max dynamic fee = 20% of base fee.
		big.NewInt(100),
	)

	vFee := new(big.Int).Sub(
		new(big.Int).Mul(maxDynamicFeeNumerator, big.NewInt(100_000_000_000)),
		big.NewInt(99_999_999_999),
	)

	variableFeeControl := new(big.Int).Quo(vFee, squareVfaBin)

	if !maxVolatilityAccumulator.IsInt64() {
		return types.DynamicFee{}, errors.New("maxVolatilityAccumulator could not fit into uint64")
	}

	if !variableFeeControl.IsInt64() {
		return types.DynamicFee{}, errors.New("variableFeeControl could not fit into uint64")
	}

	return types.DynamicFee{
		BinStep:                  constants.BinStepBpsDefault,
		BinStepU128:              constants.BinStepBpsU128Default,
		FilterPeriod:             constants.DynamicFeeFilterPeriod,
		DecayPeriod:              constants.DynamicFeeDecayPeriod,
		ReductionFactor:          constants.DynamicFeeReductionFactor,
		MaxVolatilityAccumulator: maxVolatilityAccumulator.Uint64(),
		VariableFeeControl:       variableFeeControl.Uint64(),
	}, nil
}

// GetExcludedFeeAmount calculates the excluded fee amount and trading fee from an included fee amount.
func GetExcludedFeeAmount(
	tradeFeeNumerator, includedFeeAmount *big.Int,
) struct{ ExcludedFeeAmount, TradingFee *big.Int } {
	tradingFee := maths.MulDiv(
		includedFeeAmount,
		tradeFeeNumerator,
		big.NewInt(constants.FeeDenominator),
		types.RoundingUp,
	)

	excludededFeeAmount := new(big.Int).Sub(
		includedFeeAmount,
		tradingFee,
	)
	return struct {
		ExcludedFeeAmount *big.Int
		TradingFee        *big.Int
	}{
		ExcludedFeeAmount: excludededFeeAmount,
		TradingFee:        tradingFee,
	}
}

// GetIncludedFeeAmount calculates the included fee amount from an excluded fee amount.
func GetIncludedFeeAmount(
	tradeFeeNumerator, excludedFeeAmount *big.Int,
) (*big.Int, error) {
	denominator := new(big.Int).Sub(
		big.NewInt(constants.FeeDenominator),
		tradeFeeNumerator,
	)
	if denominator.Sign() <= 0 {
		return nil, errors.New("invalid fee numerator")
	}

	includedFeeAmount := maths.MulDiv(
		excludedFeeAmount,
		tradeFeeNumerator,
		big.NewInt(constants.FeeDenominator),
		types.RoundingUp,
	)

	// Sanity check
	out := GetExcludedFeeAmount(tradeFeeNumerator, includedFeeAmount)
	if out.ExcludedFeeAmount.Cmp(excludedFeeAmount) < 0 {
		return nil, errors.New("inverse amount is less than excluded_fee_amount")
	}

	return includedFeeAmount, nil
}

// GetInAmountFromAToB calculates the input amount required from A to B for a given output amount.
func GetInAmountFromAToB(
	pool *cp_amm.PoolAccount, outAmount *big.Int,
) (types.SwapAmount, error) {
	nextSqrtPrice, err := GetNextSqrtPriceFromOutput(
		pool.SqrtPrice.BigInt(), pool.Liquidity.BigInt(), outAmount, true,
	)
	if err != nil {
		return types.SwapAmount{}, err
	}

	if nextSqrtPrice.Cmp(pool.SqrtMinPrice.BigInt()) < 0 {
		return types.SwapAmount{}, errors.New("price range is violated")
	}

	return types.SwapAmount{
		NextSqrtPrice: nextSqrtPrice,
		OutputAmount: GetAmountAFromLiquidityDelta(
			pool.Liquidity.BigInt(),
			nextSqrtPrice,
			pool.SqrtPrice.BigInt(),
			types.RoundingUp,
		),
	}, nil
}

// GetInAmountFromBToA calculates the input amount required from B to A for a given output amount.
func GetInAmountFromBToA(
	pool *cp_amm.PoolAccount, outAmount *big.Int,
) (types.SwapAmount, error) {
	// finding new target price
	nextSqrtPrice, err := GetNextSqrtPriceFromOutput(
		pool.SqrtPrice.BigInt(),
		pool.Liquidity.BigInt(),
		outAmount,
		false,
	)
	if err != nil {
		return types.SwapAmount{}, err
	}

	if nextSqrtPrice.Cmp(pool.SqrtMaxPrice.BigInt()) > 0 {
		return types.SwapAmount{}, errors.New("price range is violated")
	}
	return types.SwapAmount{
		NextSqrtPrice: nextSqrtPrice,
		OutputAmount: GetAmountBFromLiquidityDelta(
			pool.Liquidity.BigInt(),
			pool.SqrtPrice.BigInt(),
			nextSqrtPrice,
			types.RoundingUp,
		),
	}, nil
}

// GetSwapResultFromOutAmount calculates the swap result from a given output amount.
func GetSwapResultFromOutAmount(
	pool *cp_amm.PoolAccount,
	outAmount *big.Int, feeMode types.FeeMode,
	tradeDirection types.TradeDirection,
	currentPoint uint64,
) (struct {
	SwapResult  types.SwapResult
	InputAmount *big.Int
}, error) {

	dynamicFeeParam := types.DynamicFeeParams{}
	if h := pool.PoolFees.DynamicFee; h.Initialized == 1 {
		dynamicFeeParam = types.DynamicFeeParams{
			VolatilityAccumulator: h.VolatilityAccumulator.BigInt(),
			BinStep:               h.BinStep,
			VariableFeeControl:    h.VariableFeeControl,
		}
	}
	tradeFeeNumerator := GetFeeNumerator(
		currentPoint,
		new(big.Int).SetUint64(pool.ActivationPoint),
		pool.PoolFees.BaseFee.NumberOfPeriod,
		new(big.Int).SetUint64(pool.PoolFees.BaseFee.PeriodFrequency),
		types.FeeSchedulerMode(pool.PoolFees.BaseFee.FeeSchedulerMode),
		new(big.Int).SetUint64(pool.PoolFees.BaseFee.CliffFeeNumerator),
		new(big.Int).SetUint64(pool.PoolFees.BaseFee.ReductionFactor),
		dynamicFeeParam,
	)
	var (
		actualReferralFee    = big.NewInt(0)
		actualProtocolFee    = big.NewInt(0)
		actualPartnerFee     = big.NewInt(0)
		actualLpFee          = big.NewInt(0)
		includedFeeOutAmount = outAmount
		err                  error
	)
	if !feeMode.FeeOnInput {
		if includedFeeOutAmount, err = GetIncludedFeeAmount(
			tradeFeeNumerator, outAmount); err != nil {
			return struct {
				SwapResult  types.SwapResult
				InputAmount *big.Int
			}{}, err
		}

		totalFee := GetTotalFeeOnAmount(outAmount, tradeFeeNumerator)
		actualProtocolFee = maths.MulDiv(
			totalFee,
			new(big.Int).SetUint64(uint64(pool.PoolFees.ProtocolFeePercent)),
			big.NewInt(100),
			types.RoundingDown,
		)

		if feeMode.HasReferral {
			actualReferralFee = maths.MulDiv(
				actualProtocolFee,
				new(big.Int).SetUint64(uint64(pool.PoolFees.ReferralFeePercent)),
				big.NewInt(100),
				types.RoundingDown,
			)
		}

		protocolFeeAfterReferral := new(big.Int).Sub(
			actualProtocolFee,
			actualReferralFee,
		)

		actualPartnerFee = maths.MulDiv(
			protocolFeeAfterReferral,
			new(big.Int).SetUint64(uint64(pool.PoolFees.PartnerFeePercent)),
			big.NewInt(100),
			types.RoundingDown,
		)

		actualLpFee = new(big.Int).Sub(
			actualProtocolFee,
			actualPartnerFee,
		)
	}

	var tDirection types.SwapAmount
	if tradeDirection == types.TradeDirectionAtoB {
		if tDirection, err = GetInAmountFromAToB(pool, includedFeeOutAmount); err != nil {
			return struct {
				SwapResult  types.SwapResult
				InputAmount *big.Int
			}{}, err
		}
	} else {
		if tDirection, err = GetInAmountFromBToA(pool, includedFeeOutAmount); err != nil {
			return struct {
				SwapResult  types.SwapResult
				InputAmount *big.Int
			}{}, err
		}
	}

	includedFeeInAmount := tDirection.OutputAmount
	if feeMode.FeeOnInput {
		if includedFeeInAmount, err = GetIncludedFeeAmount(
			tradeFeeNumerator, tDirection.OutputAmount); err != nil {
			return struct {
				SwapResult  types.SwapResult
				InputAmount *big.Int
			}{}, err
		}

		totalFee := GetTotalFeeOnAmount(includedFeeInAmount, tradeFeeNumerator)
		actualProtocolFee = maths.MulDiv(
			totalFee,
			new(big.Int).SetUint64(uint64(pool.PoolFees.ProtocolFeePercent)),
			big.NewInt(100),
			types.RoundingDown,
		)

		if feeMode.HasReferral {
			actualReferralFee = maths.MulDiv(
				actualProtocolFee,
				new(big.Int).SetUint64(uint64(pool.PoolFees.ReferralFeePercent)),
				big.NewInt(100),
				types.RoundingDown,
			)
		}

		protocolFeeAfterReferral := new(big.Int).Sub(
			actualProtocolFee,
			actualReferralFee,
		)

		actualPartnerFee = maths.MulDiv(
			protocolFeeAfterReferral,
			new(big.Int).SetUint64(uint64(pool.PoolFees.PartnerFeePercent)),
			big.NewInt(100),
			types.RoundingDown,
		)

		actualLpFee = new(big.Int).Sub(
			actualProtocolFee,
			actualPartnerFee,
		)
	}

	return struct {
		SwapResult  types.SwapResult
		InputAmount *big.Int
	}{
		SwapResult: types.SwapResult{
			OutputAmount:  outAmount,
			NextSqrtPrice: tDirection.NextSqrtPrice,
			LPFee:         actualLpFee,
			ProtocolFee:   actualProtocolFee,
			ReferralFee:   actualReferralFee,
			PartnerFee:    actualPartnerFee,
		},
		InputAmount: includedFeeInAmount,
	}, nil
}
