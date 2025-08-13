package types

type CollectFeeMode uint8

const (
	CollectFeeModeBothToken CollectFeeMode = iota
	CollectFeeModeOnlyB
)

type FeeSchedulerMode uint8

const (
	FeeSchedulerModeLinear FeeSchedulerMode = iota
	FeeSchedulerModeExponential
)

type Rounding uint8

const (
	RoundingDown Rounding = iota
	RoundingUp
)

type TradeDirection uint8

const (
	TradeDirectionAtoB TradeDirection = iota
	TradeDirectionBtoA
)
