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

type ClosePositionInstructionParams struct {
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
	PositionState         *cp_amm.Position
	PoolState             *cp_amm.Pool
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

type PrepareCreatePoolParamsResponse struct {
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
