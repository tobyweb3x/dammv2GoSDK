package types

import (
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"math/big"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
)

type PrepareTokenAccountParams struct {
	// owner of the token accounts
	Payer       solana.PublicKey
	TokenAOwner solana.PublicKey
	TokenBOwner solana.PublicKey
	// Mint address of token A
	TokenAMint solana.PublicKey
	// Mint address of token B
	TokenBMint solana.PublicKey
	// Program ID for token A (Token or Token2022)
	TokenAProgram solana.PublicKey
	// Program ID for token B (Token or Token2022)
	TokenBProgram solana.PublicKey
}

type BuildAddLiquidityParams struct {
	Owner                 solana.PublicKey
	Position              solana.PublicKey
	Pool                  solana.PublicKey
	PositionNftAccount    solana.PublicKey
	LiquidityDelta        ag_binary.Uint128
	TokenAAccount         solana.PublicKey
	TokenBAccount         solana.PublicKey
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
	TokenAMint            solana.PublicKey
	TokenBMint            solana.PublicKey
	TokenAVault           solana.PublicKey
	TokenBVault           solana.PublicKey
	TokenAProgram         solana.PublicKey
	TokenBProgram         solana.PublicKey
}

type BuildRemoveAllLiquidityInstructionParams struct {
	PoolAuthority         solana.PublicKey
	Owner                 solana.PublicKey
	Position              solana.PublicKey
	Pool                  solana.PublicKey
	PositionNftAccount    solana.PublicKey
	TokenAAccount         solana.PublicKey
	TokenBAccount         solana.PublicKey
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
	TokenAMint            solana.PublicKey
	TokenBMint            solana.PublicKey
	TokenAVault           solana.PublicKey
	TokenBVault           solana.PublicKey
	TokenAProgram         solana.PublicKey
	TokenBProgram         solana.PublicKey
}

type ClaimPositionFeeInstructionParams struct {
	Owner              solana.PublicKey
	PoolAuthority      solana.PublicKey
	Pool               solana.PublicKey
	Position           solana.PublicKey
	PositionNftAccount solana.PublicKey
	TokenAAccount      solana.PublicKey
	TokenBAccount      solana.PublicKey
	TokenAVault        solana.PublicKey
	TokenBVault        solana.PublicKey
	TokenAMint         solana.PublicKey
	TokenBMint         solana.PublicKey
	TokenAProgram      solana.PublicKey
	TokenBProgram      solana.PublicKey
}

type ClosePositionParams struct {
	Owner              solana.PublicKey
	PoolAuthority      solana.PublicKey
	Pool               solana.PublicKey
	Position           solana.PublicKey
	PositionNftMint    solana.PublicKey
	PositionNftAccount solana.PublicKey
}

type RefreshVestingParams struct {
	Owner              solana.PublicKey
	Position           solana.PublicKey
	PositionNftAccount solana.PublicKey
	Pool               solana.PublicKey
	VestingAccounts    []solana.PublicKey
}

type BuildLiquidatePositionInstructionParams struct {
	Owner                 solana.PublicKey
	Position              solana.PublicKey
	PositionNftAccount    solana.PublicKey
	PositionState         *cp_amm.PositionAccount
	PoolState             *cp_amm.PoolAccount
	TokenAAccount         solana.PublicKey
	TokenBAccount         solana.PublicKey
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
}

type CreatePositionParams struct {
	Owner       solana.PublicKey
	Payer       solana.PublicKey
	Pool        solana.PublicKey
	PositionNft solana.PublicKey
}

type PrepareCustomizablePoolParams struct {
	Pool          solana.PublicKey
	TokenAMint    solana.PublicKey
	TokenBMint    solana.PublicKey
	TokenAAmount  uint64
	TokenBAmount  uint64
	Payer         solana.PublicKey
	PositionNft   solana.PublicKey
	TokenAProgram solana.PublicKey
	TokenBProgram solana.PublicKey
}

type PrepareCreatePoolResponse struct {
	Positon, PositionNftAccount,
	TokenAVault, TokenBVault,
	PayerTokenA, PayerTokenB solana.PublicKey
	Ixns               []solana.Instruction
	TokenBadgeAccounts solana.AccountMetaSlice
}

type SetupFeeClaimAccountsParams struct {
	Payer           solana.PublicKey
	Owner           solana.PublicKey
	TokenAMint      solana.PublicKey
	TokenBMint      solana.PublicKey
	TokenAProgram   solana.PublicKey
	TokenBProgram   solana.PublicKey
	Receiver        solana.PublicKey
	TempWSolAccount solana.PublicKey
}

type LiquidityDeltaParams struct {
	MaxAmountTokenA *big.Int
	MaxAmountTokenB *big.Int
	SqrtPrice       *big.Int
	SqrtMinPrice    *big.Int
	SqrtMaxPrice    *big.Int
	TokenAInfo      *struct {
		Mint         token.Mint
		CurrentEpoch uint64
	}
	TokenBInfo *struct {
		Mint         token.Mint
		CurrentEpoch uint64
	}
}

type DynamicFeeParams struct {
	VolatilityAccumulator *big.Int
	BinStep               uint16
	VariableFeeControl    uint32
}

type FeeMode struct {
	FeeOnInput   bool
	FeesOnTokenA bool
	HasReferral  bool
}
type GetQuoteParams struct {
	InAmount        *big.Int
	InputTokenMint  solana.PublicKey
	Slippage        float64
	PoolState       *cp_amm.PoolAccount
	CurrentTime     uint64
	CurrentSlot     uint64
	InputTokenInfo  *TokenEpochInfo
	OutputTokenInfo *TokenEpochInfo
}

type TokenEpochInfo struct {
	Mint         token.Mint
	CurrentEpoch uint64
}

type GetQuoteResult struct {
	SwapInAmount     *big.Int
	ConsumedInAmount *big.Int
	SwapOutAmount    *big.Int
	MinSwapOutAmount *big.Int
	TotalFee         *big.Int
	PriceImpact      float64
}

type GetWithdrawQuoteParams struct {
	LiquidityDelta  *big.Int
	MinSqrtPrice    *big.Int
	MaxSqrtPrice    *big.Int
	SqrtPrice       *big.Int
	TokenATokenInfo *TokenEpochInfo
	TokenBTokenInfo *TokenEpochInfo
}

type DepositQuote struct {
	// The actual amount used as input (after deducting transfer fees).
	ActualInputAmount   *big.Int
	ConsumedInputAmount *big.Int
	// The calculated corresponding amount of the other token.
	OutputAmount *big.Int
	// The amount of liquidity that will be added to the pool.
	LiquidityDelta *big.Int
}

type GetDepositQuoteParams struct {
	InAmount        *big.Int
	IsTokenA        bool
	MinSqrtPrice    *big.Int
	MaxSqrtPrice    *big.Int
	SqrtPrice       *big.Int
	InputTokenInfo  *TokenEpochInfo
	OutputTokenInfo *TokenEpochInfo
}

type PreparePoolCreationSingleSideParams struct {
	TokenAAmount  *big.Int
	MinSqrtPrice  *big.Int
	MaxSqrtPrice  *big.Int
	InitSqrtPrice *big.Int
	TokenAInfo    *TokenEpochInfo
}

type PreparePoolCreationParams struct {
	TokenAAmount *big.Int
	TokenBAmount *big.Int
	MinSqrtPrice *big.Int
	MaxSqrtPrice *big.Int
	TokenAInfo   *TokenEpochInfo
	TokenBInfo   *TokenEpochInfo
}

type CreatePoolParams struct {
	Creator         solana.PublicKey
	Payer           solana.PublicKey
	Config          solana.PublicKey
	PositionNFT     solana.PublicKey
	TokenAMint      solana.PublicKey
	TokenBMint      solana.PublicKey
	InitSqrtPrice   ag_binary.Uint128
	LiquidityDelta  ag_binary.Uint128
	TokenAAmount    uint64
	TokenBAmount    uint64
	ActivationPoint *uint64
	TokenAProgram   solana.PublicKey
	TokenBProgram   solana.PublicKey
	IsLockLiquidity bool
}

type PoolFeesParams struct {
	BaseFee            BaseFee
	ProtocolFeePercent float64
	PartnerFeePercent  float64
	ReferralFeePercent float64
	DynamicFee         *DynamicFee
}

type DynamicFee struct {
	BinStep                  uint8
	BinStepU128              *big.Int
	FilterPeriod             uint64
	DecayPeriod              uint64
	ReductionFactor          uint64
	MaxVolatilityAccumulator uint64
	VariableFeeControl       uint64
}

type BaseFee struct {
	CliffFeeNumerator *big.Int
	NumberOfPeriod    uint64
	PeriodFrequency   *big.Int
	ReductionFactor   *big.Int
	FeeSchedulerMode  FeeSchedulerMode
}

type InitializeCustomizeablePoolParams struct {
	Payer           solana.PublicKey
	Creator         solana.PublicKey
	PositionNFT     solana.PublicKey
	TokenAMint      solana.PublicKey
	TokenBMint      solana.PublicKey
	TokenAAmount    uint64
	TokenBAmount    uint64
	SqrtMinPrice    *big.Int
	SqrtMaxPrice    *big.Int
	LiquidityDelta  *big.Int
	InitSqrtPrice   *big.Int
	PoolFees        cp_amm.PoolFeeParameters
	HasAlphaVault   bool
	ActivationType  uint8
	CollectFeeMode  uint8
	ActivationPoint *uint64
	TokenAProgram   solana.PublicKey
	TokenBProgram   solana.PublicKey
	IsLockLiquidity bool
}

type InitializeCustomizeablePoolWithDynamicConfigParams struct {
	InitializeCustomizeablePoolParams
	Config               solana.PublicKey
	PoolCreatorAuthority solana.PublicKey
}

type AddLiquidityParams struct {
	Owner                 solana.PublicKey
	Position              solana.PublicKey
	Pool                  solana.PublicKey
	PositionNftAccount    solana.PublicKey
	LiquidityDelta        ag_binary.Uint128
	MaxAmountTokenA       uint64
	MaxAmountTokenB       uint64
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
	TokenAMint            solana.PublicKey
	TokenBMint            solana.PublicKey
	TokenAVault           solana.PublicKey
	TokenBVault           solana.PublicKey
	TokenAProgram         solana.PublicKey
	TokenBProgram         solana.PublicKey
}

type CreatePositionAndAddLiquidity struct {
	Owner                 solana.PublicKey
	Pool                  solana.PublicKey
	PositionNFT           solana.PublicKey
	LiquidityDelta        ag_binary.Uint128
	MaxAmountTokenA       uint64
	MaxAmountTokenB       uint64
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
	TokenAMint            solana.PublicKey
	TokenBMint            solana.PublicKey
	TokenAProgram         solana.PublicKey
	TokenBProgram         solana.PublicKey
}

type Vesting struct {
	Account      solana.PublicKey
	VestingState *cp_amm.VestingAccount
}

type RemoveLiquidityParams struct {
	Owner                 solana.PublicKey
	Position              solana.PublicKey
	Pool                  solana.PublicKey
	PositionNftAccount    solana.PublicKey
	LiquidityDelta        ag_binary.Uint128
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
	TokenAMint            solana.PublicKey
	TokenBMint            solana.PublicKey
	TokenAVault           solana.PublicKey
	TokenBVault           solana.PublicKey
	TokenAProgram         solana.PublicKey
	TokenBProgram         solana.PublicKey
	Vestings              []Vesting
	CurrentPoint          *big.Int
}

type RemoveAllLiquidityParams struct {
	AddLiquidityParams
	Vestings     []Vesting
	CurrentPoint *big.Int
}

type SwapParams struct {
	Payer                solana.PublicKey
	Pool                 solana.PublicKey
	InputTokenMint       solana.PublicKey
	OutputTokenMint      solana.PublicKey
	AmountIn             uint64
	MinimumAmountOut     uint64
	TokenAMint           solana.PublicKey
	TokenBMint           solana.PublicKey
	TokenAVault          solana.PublicKey
	TokenBVault          solana.PublicKey
	TokenAProgram        solana.PublicKey
	TokenBProgram        solana.PublicKey
	ReferralTokenAccount solana.PublicKey
}

type LockPositionParams struct {
	Owner                solana.PublicKey
	Payer                solana.PublicKey
	VestingAccount       solana.PublicKey
	Position             solana.PublicKey
	PositionNftAccount   solana.PublicKey
	Pool                 solana.PublicKey
	CliffPoint           *uint64
	PeriodFrequency      uint64
	CliffUnlockLiquidity ag_binary.Uint128
	LiquidityPerPeriod   ag_binary.Uint128
	NumberOfPeriod       uint16
}

type PermanentLockParams struct {
	Owner              solana.PublicKey
	Position           solana.PublicKey
	PositionNftAccount solana.PublicKey
	Pool               solana.PublicKey
	UnlockedLiquidity  ag_binary.Uint128
}
type RemoveAllLiquidityAndClosePositionParams struct {
	Owner                 solana.PublicKey
	Position              solana.PublicKey
	PositionNftAccount    solana.PublicKey
	PoolState             *cp_amm.PoolAccount
	PositionState         *cp_amm.PositionAccount
	TokenAAmountThreshold uint64
	TokenBAmountThreshold uint64
	Vestings              []Vesting
	CurrentPoint          *big.Int
}

type MergePositionParams struct {
	Owner                                solana.PublicKey
	PositionA                            solana.PublicKey
	PositionB                            solana.PublicKey
	PoolState                            *cp_amm.PoolAccount
	PositionBNftAccount                  solana.PublicKey
	PositionANftAccount                  solana.PublicKey
	PositionBState                       *cp_amm.PositionAccount
	TokenAAmountAddLiquidityThreshold    uint64
	TokenBAmountAddLiquidityThreshold    uint64
	TokenAAmountRemoveLiquidityThreshold uint64
	TokenBAmountRemoveLiquidityThreshold uint64
	PositionBVestings                    []Vesting
	CurrentPoint                         *big.Int
}

type UpdateRewardDurationParams struct {
	Pool        solana.PublicKey
	Admin       solana.PublicKey
	RewardIndex uint8
	NewDuration uint64
}

type UpdateRewardFunderParams struct {
	Pool        solana.PublicKey
	Admin       solana.PublicKey
	RewardIndex uint8
	NewFunder   solana.PublicKey
}

type FundRewardParams struct {
	Funder       solana.PublicKey
	RewardIndex  uint8
	Pool         solana.PublicKey
	CarryForward bool
	Amount       uint64
}

type WithdrawIneligibleRewardParams struct {
	RewardIndex uint8
	Pool        solana.PublicKey
	Funder      solana.PublicKey
}

type ClaimPartnerFeeParams struct {
	Partner         solana.PublicKey
	Pool            solana.PublicKey
	MaxAmountA      uint64
	MaxAmountB      uint64
	Receiver        solana.PublicKey
	FeePayer        solana.PublicKey
	TempWSolAccount solana.PublicKey
}

type ClaimPositionFeeParams struct {
	Owner              solana.PublicKey
	Position           solana.PublicKey
	Pool               solana.PublicKey
	PositionNftAccount solana.PublicKey
	TokenAMint         solana.PublicKey
	TokenBMint         solana.PublicKey
	TokenAVault        solana.PublicKey
	TokenBVault        solana.PublicKey
	TokenAProgram      solana.PublicKey
	TokenBProgram      solana.PublicKey
	Receiver           solana.PublicKey
	FeePayer           solana.PublicKey
	TempWSolAccount    solana.PublicKey
}

type ClaimPositionFeeParams2 struct {
	Owner              solana.PublicKey
	Position           solana.PublicKey
	Pool               solana.PublicKey
	PositionNftAccount solana.PublicKey
	TokenAMint         solana.PublicKey
	TokenBMint         solana.PublicKey
	TokenAVault        solana.PublicKey
	TokenBVault        solana.PublicKey
	TokenAProgram      solana.PublicKey
	TokenBProgram      solana.PublicKey
	Receiver           solana.PublicKey
	FeePayer           solana.PublicKey
}

type ClaimRewardParams struct {
	User                    solana.PublicKey
	Position                solana.PublicKey
	PoolState               *cp_amm.PoolAccount
	PositionState           *cp_amm.PositionAccount
	PositionNftAccount      solana.PublicKey
	RewardIndex, SkipReward uint8
	FeePayer                solana.PublicKey
}

type SwapAmount struct {
	OutputAmount, NextSqrtPrice *big.Int
}
type SwapResult struct {
	OutputAmount  *big.Int
	NextSqrtPrice *big.Int
	LPFee         *big.Int
	ProtocolFee   *big.Int
	ReferralFee   *big.Int
	PartnerFee    *big.Int
}

type GetQuoteExactOutParams struct {
	OutAmount       *big.Int
	OutputTokenMint solana.PublicKey
	Slippage        float64
	PoolState       *cp_amm.PoolAccount
	CurrentTime     uint64
	CurrentSlot     uint64
	InputTokenInfo  *TokenEpochInfo
	OutputTokenInfo *TokenEpochInfo
}

type QuoteExactOutResult struct {
	SwapResult     SwapResult
	InputAmount    *big.Int
	MaxInputAmount *big.Int
	PriceImpact    float64
}

type SplitPositionParams struct {
	FirstPositionOwner                 solana.PublicKey
	SecondPositionOwner                solana.PublicKey
	Pool                               solana.PublicKey
	FirstPosition                      solana.PublicKey
	FirstPositionNftAccount            solana.PublicKey
	SecondPosition                     solana.PublicKey
	SecondPositionNftAccount           solana.PublicKey
	PermanentLockedLiquidityPercentage float64
	UnlockedLiquidityPercentage        float64
	FeeAPercentage                     float64
	FeeBPercentage                     float64
	Reward0Percentage                  float64
	Reward1Percentage                  float64
}
