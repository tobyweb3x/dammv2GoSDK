package types

type CollectFeeMode uint8

const (
	BothToken CollectFeeMode = iota
	OnlyB
)

type FeeSchedulerMode uint8

const (
	Linear FeeSchedulerMode = iota
	Exponential
)

type Rounding uint8

const (
	RoundingDown Rounding = iota
	RoundingUp
)

type TradeDirection uint8

const (
	AtoB TradeDirection = iota
	BtoA
)
