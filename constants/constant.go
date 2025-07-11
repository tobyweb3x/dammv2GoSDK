package constants

import "math/big"

const (
	LiquidityScale            = 128
	ScaleOffset               = 64
	BasisPointMax             = 10_000
	MaxFeeNumerator           = 500_000_000
	FeeDenominator            = 1_000_000_000
	MinCuBuffer               = 50_000
	MaxCuBuffer               = 200_000
	DynamicFeeFilterPeriod    = 10
	DynamicFeeDecayPeriod     = 120
	DynamicFeeReductionFactor = 5_000 // 50%
	BinStepBpsDefault         = 1
	MaxPriceChangeBpsDefault  = 1_500 // 15%
)

// These are big.Int values, initialized via SetString
var (
	// MinSqrtPrice
	//  MinSqrtPrice = new(big.Int).SetUint64(4295048016)
	MinSqrtPrice = new(big.Int).SetUint64(4295048016)

	// MaxSqrtPrice
	//  MaxSqrtPrice  = new(big.Int).SetString("79226673521066979257578248091", 10)
	MaxSqrtPrice, _ = new(big.Int).SetString("79226673521066979257578248091", 10)

	// BinStepBpsU128Default
	//  BinStepBpsU128Default = new(big.Int).SetString("1844674407370955", 10)
	BinStepBpsU128Default, _ = new(big.Int).SetString("1844674407370955", 10)
)
