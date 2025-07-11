package testUtils

import (
	"log"
	"math/big"
)

const (
	Decimals                         = 6
	MinCuBuffer                      = 50_000
	MaxCuBuffer                      = 200_000
	DynamicFeeFilterPeriodDefault    = 10
	DynamicFeeDecayPeriodDefault     = 120
	DynamicFeeReductionFactorDefault = 5000 // 50%
	BinStepBpsDefault                = 1
)

var (
	// MaxSqrtPrice
	//  MaxSqrtPric = new(big.Int).SetString("79226673521066979257578248091", 10)
	MaxSqrtPrice *big.Int

	// BinStepBpsU128Default
	//  BinStepBpsU128Default = new(big.Int).SetString("1844674407370955", 10)
	BinStepBpsU128Default *big.Int

	// MinSqrtPrice
	//  MinSqrtPrice = big.NewInt(4295048016)
	MinSqrtPrice = big.NewInt(4_295_048_016)
)

func init() {

	var success bool
	
	MaxSqrtPrice, success = new(big.Int).SetString("79226673521066979257578248091", 10)
	if !success {
		log.Fatalln("error setting MaxSqrtPrice")
	}

	BinStepBpsU128Default, success = new(big.Int).SetString("1844674407370955", 10)
	if !success {
		log.Fatalln("error setting BinStepBpsU128Default")
	}
}
