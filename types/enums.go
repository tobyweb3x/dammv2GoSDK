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

type Rounding int

const (
	RoundingDown Rounding = iota
	RoundingUp
)
