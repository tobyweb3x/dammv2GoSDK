package dammv2gosdk

import (
	"context"
	"dammv2GoSDK/anchor"
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"dammv2GoSDK/helpers"
	"dammv2GoSDK/types"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type CpAMM struct {
	poolAuthority solana.PublicKey
	conn          *rpc.Client
}

func NewCpAMM(conn *rpc.Client) *CpAMM {
	return &CpAMM{
		conn:          conn,
		poolAuthority: DerivePoolAuthority(),
	}
}

// prepareTokenAccounts prepares token accounts for a transaction by retrieving or creating ix for creating the associated token accounts.
func (cp *CpAMM) prepareTokenAccounts(
	ctx context.Context,
	param types.PrepareTokenAccountParams,
) (struct {
	TokenAAta, TokenBAta                 solana.PublicKey
	GetOrCreateAtaIxA, GetOrCreateAtaIxB *solana.GenericInstruction
}, error) {

	type res struct {
		AtaPubkey solana.PublicKey
		Ix        *solana.GenericInstruction
		Err       error
	}
	var (
		wg   sync.WaitGroup
		a, b res
	)
	wg.Add(2)

	go func(p *res, wg *sync.WaitGroup) {
		defer wg.Done()
		ata, ix, err := helpers.GetOrCreateATAInstruction(
			ctx,
			cp.conn,
			param.TokenAMint,
			param.TokenAOwner,
			param.Payer,
			true,
			param.TokenAProgram,
		)
		if err != nil {
			p.Err = err
			return
		}
		p.AtaPubkey = ata
		p.Ix = ix
	}(&a, &wg)

	go func(p *res, wg *sync.WaitGroup) {
		defer wg.Done()
		ata, ix, err := helpers.GetOrCreateATAInstruction(
			ctx,
			cp.conn,
			param.TokenBMint,
			param.TokenBOwner,
			param.Payer,
			true,
			param.TokenBProgram,
		)
		if err != nil {
			p.Err = err
			return
		}
		p.AtaPubkey = ata
		p.Ix = ix
	}(&b, &wg)

	wg.Wait()

	handleNilErr := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}

	if a.Err != nil || b.Err != nil {
		return struct {
				TokenAAta         solana.PublicKey
				TokenBAta         solana.PublicKey
				GetOrCreateAtaIxA *solana.GenericInstruction
				GetOrCreateAtaIxB *solana.GenericInstruction
			}{},
			fmt.Errorf("err from token A— %s: err from tokenB— %s", handleNilErr(a.Err), handleNilErr(b.Err))
	}

	return struct {
		TokenAAta         solana.PublicKey
		TokenBAta         solana.PublicKey
		GetOrCreateAtaIxA *solana.GenericInstruction
		GetOrCreateAtaIxB *solana.GenericInstruction
	}{
		TokenAAta:         a.AtaPubkey,
		TokenBAta:         b.AtaPubkey,
		GetOrCreateAtaIxA: a.Ix,
		GetOrCreateAtaIxB: b.Ix,
	}, nil
}

// GetTokenBadgeAccounts derives token badge account metadata.
func (cp CpAMM) GetTokenBadgeAccounts(
	tokenAMint,
	tokenBMint solana.PublicKey,
) solana.AccountMetaSlice {

	return append(
		make(solana.AccountMetaSlice, 0, 2),
		&solana.AccountMeta{
			PublicKey: DeriveTokenBadgeAddress(tokenAMint),
		},
		&solana.AccountMeta{
			PublicKey: DeriveTokenBadgeAddress(tokenBMint),
		},
	)
}

// buildAddLiquidityInstruction builds an instruction to add liquidity to a position.
func (cp CpAMM) buildAddLiquidityInstruction(
	param types.BuildAddLiquidityParams,
) *cp_amm.Instruction {
	return cp_amm.NewAddLiquidityInstruction(
		cp_amm.AddLiquidityParameters{
			LiquidityDelta:        param.LiquidityDelta,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
		},
		param.Pool,
		param.Position,
		param.TokenAAccount,
		param.TokenBAccount,
		param.TokenAVault,
		param.TokenBVault,
		param.TokenAMint,
		param.TokenBMint,
		param.PositionNftAccount,
		param.Owner,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.PublicKey{},
		solana.PublicKey{},
	).Build()
}

// buildRemoveAllLiquidityInstruction builds an instruction to remove all liquidity from a position.
func (cp CpAMM) buildRemoveAllLiquidityInstruction(
	param types.BuildRemoveAllLiquidityInstructionParams,
) *cp_amm.Instruction {
	return cp_amm.NewRemoveLiquidityInstruction(
		cp_amm.RemoveLiquidityParameters{
			// LiquidityDelta: ,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
		},
		param.PoolAuthority,
		param.Pool,
		param.Position,
		param.TokenAAccount,
		param.TokenBAccount,
		param.TokenAVault,
		param.TokenBVault,
		param.TokenAMint,
		param.TokenBMint,
		param.PositionNftAccount,
		param.Owner,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.PublicKey{},
		solana.PublicKey{},
	).Build()
}

// buildClaimPositionFeeInstruction builds an instruction to claim fees accumulated by a position.
func (cp CpAMM) buildClaimPositionFeeInstruction(
	param types.ClaimPositionFeeInstructionParams,
) *cp_amm.Instruction {
	return cp_amm.NewClaimPositionFeeInstruction(
		param.PoolAuthority,
		param.Pool,
		param.Position,
		param.TokenAAccount,
		param.TokenBAccount,
		param.TokenAVault,
		param.TokenBVault,
		param.TokenAMint,
		param.TokenBMint,
		param.PositionNftAccount,
		param.Owner,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.PublicKey{},
		solana.PublicKey{},
	).Build()
}

// buildClosePositionInstruction builds an instruction to close a position.
func (cp CpAMM) buildClosePositionInstruction(
	param types.ClosePositionInstructionParams,
) *cp_amm.Instruction {
	return cp_amm.NewClosePositionInstruction(
		param.PositionNftMint,
		param.PositionNftAccount,
		param.Pool,
		param.Position,
		param.PoolAuthority,
		param.Owner,
		param.Owner,
		solana.Token2022ProgramID,
		solana.PublicKey{},
		solana.PublicKey{},
	).Build()
}

func (cp CpAMM) buildRefreshVestingInstruction(
	param types.RefreshVestingParams,
) (*cp_amm.Instruction, error) {

	if len(param.VestingAccounts) == 0 {
		return nil, errors.New("empty VestingAccounts")
	}
	ix := cp_amm.NewRefreshVestingInstruction(
		param.Pool,
		param.Position,
		param.PositionNftAccount,
		param.Owner,
	)

	ix.AccountMetaSlice = slices.Grow(ix.AccountMetaSlice, len(param.VestingAccounts))
	for i := range len(param.VestingAccounts) {
		ix.AccountMetaSlice = append(
			ix.AccountMetaSlice,
			solana.Meta(param.VestingAccounts[i]).WRITE(),
		)
	}
	return ix.Build(), nil
}

// buildLiquidatePositionInstruction builds instructions to claim fees, remove liquidity, and close a position
func (cp *CpAMM) buildLiquidatePositionInstruction(
	param types.BuildLiquidatePositionInstructionParams,
) ([]*cp_amm.Instruction, error) {

	ixns := make([]*cp_amm.Instruction, 0, 3)
	// 1. claim position fee
	claimPositionFeeInstruction := cp.buildClaimPositionFeeInstruction(
		types.ClaimPositionFeeInstructionParams{
			Owner:              param.Owner,
			PoolAuthority:      cp.poolAuthority,
			Pool:               param.PositionState.Pool,
			Position:           param.Position,
			PositionNftAccount: param.PositionNftAccount,
			TokenAAccount:      param.TokenAAccount,
			TokenBAccount:      param.TokenBAccount,
			TokenAVault:        param.PoolState.TokenAVault,
			TokenBVault:        param.PoolState.TokenBVault,
			TokenAMint:         param.PoolState.TokenAMint,
			TokenBMint:         param.PoolState.TokenBMint,
			TokenAProgram:      helpers.GetTokenProgram(param.PoolState.TokenAFlag),
			TokenBProgram:      helpers.GetTokenProgram(param.PoolState.TokenBFlag),
		},
	)
	ixns = append(ixns, claimPositionFeeInstruction)

	// 2. remove all liquidity
	removeAllLiquidityInstruction := cp.buildRemoveAllLiquidityInstruction(
		types.BuildRemoveAllLiquidityInstructionParams{
			PoolAuthority:         cp.poolAuthority,
			Owner:                 param.Owner,
			Pool:                  param.PositionState.Pool,
			Position:              param.Position,
			PositionNftAccount:    param.PositionNftAccount,
			TokenAAccount:         param.TokenAAccount,
			TokenBAccount:         param.TokenBAccount,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
			TokenAMint:            param.PoolState.TokenAMint,
			TokenBMint:            param.PoolState.TokenBMint,
			TokenAVault:           param.PoolState.TokenAVault,
			TokenBVault:           param.PoolState.TokenBVault,
			TokenAProgram:         helpers.GetTokenProgram(param.PoolState.TokenAFlag),
			TokenBProgram:         helpers.GetTokenProgram(param.PoolState.TokenBFlag),
		},
	)
	ixns = append(ixns, removeAllLiquidityInstruction)

	// 3. close position
	closePositionInstruction := cp.buildClosePositionInstruction(
		types.ClosePositionInstructionParams{
			Owner:              param.Owner,
			PoolAuthority:      cp.poolAuthority,
			Pool:               param.PositionState.Pool,
			Position:           param.Position,
			PositionNftMint:    param.PositionState.NftMint,
			PositionNftAccount: param.PositionNftAccount,
		},
	)
	ixns = append(ixns, closePositionInstruction)

	return ixns, nil
}

// buildCreatePositionInstruction builds a instruction to create a position.
func (cp *CpAMM) buildCreatePositionInstruction(
	param types.CreatePositionParams,
) (struct {
	Position, PositonNftAccount solana.PublicKey
	Ix                          *cp_amm.Instruction
}, error) {
	position := DerivePositionAddress(param.PositionNft)
	positionNftAccount := DerivePositionNftAccount(param.PositionNft)

	ix, err := cp_amm.NewCreatePositionInstruction(
		param.Owner,
		param.PositionNft,
		positionNftAccount,
		param.Pool,
		position,
		cp.poolAuthority,
		param.Payer,
		solana.Token2022ProgramID,
		solana.SystemProgramID,
		solana.PublicKey{},
		solana.PublicKey{},
	).ValidateAndBuild()
	if err != nil {
		return struct {
			Position          solana.PublicKey
			PositonNftAccount solana.PublicKey
			Ix                *cp_amm.Instruction
		}{}, err
	}

	return struct {
		Position          solana.PublicKey
		PositonNftAccount solana.PublicKey
		Ix                *cp_amm.Instruction
	}{
		Position:          position,
		PositonNftAccount: positionNftAccount,
		Ix:                ix,
	}, nil
}

// prepareCreatePoolParams prepares common customizable pool creation logic.
func (cp *CpAMM) prepareCreatePoolParams(
	ctx context.Context,
	param types.PrepareCustomizablePoolParams,
) (*types.PrepareCreatePoolParamsResponse, error) {

	res, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Payer,
			TokenAOwner:   param.Payer,
			TokenBOwner:   param.Payer,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	preInstructions := make([]solana.Instruction, 0, 2+2+2)
	preInstructions = append(preInstructions, res.GetOrCreateAtaIxA, res.GetOrCreateAtaIxB)

	if param.TokenAMint.Equals(helpers.NativeMint) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Payer,
			res.TokenAAta,
			param.TokenAAmount,
		)
		preInstructions = append(preInstructions, wrapSOLIx...)
	}
	if param.TokenBMint.Equals(helpers.NativeMint) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Payer,
			res.TokenBAta,
			param.TokenAAmount,
		)
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	return &types.PrepareCreatePoolParamsResponse{
			Positon:            DerivePositionAddress(param.PositionNft),
			PositionNftAccount: DerivePositionNftAccount(param.PositionNft),
			TokenAVault:        DeriveTokenVaultAddress(param.TokenAMint, param.Pool),
			TokenBVault:        DeriveTokenVaultAddress(param.TokenBMint, param.Pool),
			PayerTokenA:        res.TokenAAta,
			PayerTokenB:        res.TokenBAta,
			Ixns:               preInstructions,
			TokenBadgeAccounts: cp.GetTokenBadgeAccounts(
				param.TokenAMint,
				param.TokenBMint,
			),
		},
		nil
}

type setupFeeClaimAccountsResult struct {
	TokenAAccount    solana.PublicKey
	TokenBAccount    solana.PublicKey
	PreInstructions  []solana.Instruction
	PostInstructions []solana.Instruction
}

func (cp *CpAMM) setupFeeClaimAccounts(
	ctx context.Context,
	param types.SetupFeeClaimAccountsParams,
) (setupFeeClaimAccountsResult, error) {
	tokenAIsSOL := param.TokenAMint.Equals(helpers.NativeMint)
	tokenBIsSOL := param.TokenBMint.Equals(helpers.NativeMint)
	hasSolToken := tokenAIsSOL || tokenBIsSOL

	tokenAOwner := param.Owner
	tokenBOwner := param.Owner

	if !param.Receiver.IsZero() {
		if tokenAIsSOL && !param.TempWSolAccount.IsZero() {
			tokenAOwner = param.TempWSolAccount
		} else {
			tokenAOwner = param.Receiver
		}

		if tokenBIsSOL && !param.TempWSolAccount.IsZero() {
			tokenBOwner = param.TempWSolAccount
		} else {
			tokenBOwner = param.Receiver
		}
	}

	accs, err := cp.prepareTokenAccounts(ctx, types.PrepareTokenAccountParams{
		Payer:         param.Payer,
		TokenAOwner:   tokenAOwner,
		TokenBOwner:   tokenBOwner,
		TokenAMint:    param.TokenAMint,
		TokenBMint:    param.TokenBMint,
		TokenAProgram: param.TokenAProgram,
		TokenBProgram: param.TokenBProgram,
	})
	if err != nil {
		return setupFeeClaimAccountsResult{}, fmt.Errorf("prepareTokenAccounts failed: %w", err)
	}

	preInstructions := make([]solana.Instruction, 0, 3)
	postInstructions := make([]solana.Instruction, 0, 3)

	preInstructions = append(preInstructions, accs.GetOrCreateAtaIxA, accs.GetOrCreateAtaIxB)

	if hasSolToken {
		var (
			oWner = param.Owner
			recv  = param.Owner
		)
		if !param.TempWSolAccount.IsZero() {
			oWner = param.TempWSolAccount
		}
		if !param.Receiver.IsZero() {
			recv = param.Receiver
		}

		unwrapIx, err := helpers.UnwrapSOLInstruction(
			oWner,
			recv,
			false,
		)
		if err != nil {
			return setupFeeClaimAccountsResult{}, err
		}

		postInstructions = append(postInstructions, unwrapIx)
	}

	return setupFeeClaimAccountsResult{
		TokenAAccount:    accs.TokenAAta,
		TokenBAccount:    accs.TokenBAta,
		PreInstructions:  preInstructions,
		PostInstructions: postInstructions,
	}, nil
}

// FetchConfigState fetches the Config state of the program.
func (cp *CpAMM) FetchConfigState(ctx context.Context, config solana.PublicKey) (*cp_amm.ConfigAccount, error) {
	configState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.ConfigAccount { return &cp_amm.ConfigAccount{} },
	).Fetch(
		ctx,
		config,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if configState == nil {
		return nil, fmt.Errorf("config account: %s not found", config.String())
	}

	return configState, nil
}

// FetchPoolState fetches the Pool state.
func (cp *CpAMM) FetchPoolState(ctx context.Context, pool solana.PublicKey) (*cp_amm.PoolAccount, error) {
	poolState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PoolAccount { return &cp_amm.PoolAccount{} },
	).Fetch(
		ctx,
		pool,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if poolState == nil {
		return nil, fmt.Errorf("pool account: %s not found", pool.String())
	}

	return poolState, nil
}

// FetchPoolState fetches the Position state.
func (cp *CpAMM) FetchPositionState(ctx context.Context, position solana.PublicKey) (*cp_amm.PositionAccount, error) {
	positionState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PositionAccount { return &cp_amm.PositionAccount{} },
	).Fetch(
		ctx,
		position,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if positionState == nil {
		return nil, fmt.Errorf("position account: %s not found", position.String())
	}

	return positionState, nil
}

// GetAllConfigs retrieves all config accounts.
func (cp *CpAMM) GetAllConfigs(ctx context.Context, config solana.PublicKey) ([]anchor.ProgramAccount[*cp_amm.ConfigAccount], error) {
	configState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.ConfigAccount { return &cp_amm.ConfigAccount{} },
	).All(
		ctx,
		CpAMMProgramId,
		cp_amm.ConfigAccountDiscriminator,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(configState) == 0 {
		return nil, fmt.Errorf("config account: %s not found", config.String())
	}

	return configState, nil
}

// GetAllPools retrieves all pool accounts.
func (cp *CpAMM) GetAllPools(ctx context.Context, pool solana.PublicKey) ([]anchor.ProgramAccount[*cp_amm.PoolAccount], error) {
	poolState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PoolAccount { return &cp_amm.PoolAccount{} },
	).All(
		ctx,
		CpAMMProgramId,
		cp_amm.PoolAccountDiscriminator,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(poolState) == 0 {
		return nil, fmt.Errorf("pool account: %s not found", pool.String())
	}

	return poolState, nil
}

// GetAllPositions retrieves all position accounts.
func (cp *CpAMM) GetAllPositions(ctx context.Context, position solana.PublicKey) ([]anchor.ProgramAccount[*cp_amm.PositionAccount], error) {
	positionState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PositionAccount { return &cp_amm.PositionAccount{} },
	).All(
		ctx,
		CpAMMProgramId,
		cp_amm.PositionAccountDiscriminator,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(positionState) == 0 {
		return nil, fmt.Errorf("position account: %s not found", position.String())
	}

	return positionState, nil
}

// GetAllPositionsByPool gets all positions a specific pool.
func (cp *CpAMM) GetUserPositionByPool(
	ctx context.Context,
	pool, user solana.PublicKey,
) ([]GetPositionsByUserResult, error) {

	allUserPositions, err := cp.GetPositionsByUser(ctx, user)
	if err != nil {
		return nil, err
	}
	res := make([]GetPositionsByUserResult, 0, len(allUserPositions))
	for _, position := range allUserPositions {
		if position.PositionState.Pool.Equals(pool) {
			res = append(res, position)
		}
	}
	return slices.Clip(res), nil
}

type GetPositionsByUserResult struct {
	PositionNftAccount, Position solana.PublicKey
	PositionState                *cp_amm.PositionAccount
}

// GetPositionsByUser all positions of a user across all pools.
func (cp *CpAMM) GetPositionsByUser(
	ctx context.Context, user solana.PublicKey,
) ([]GetPositionsByUserResult, error) {

	userPositionAccounts, err := helpers.GetAllPositionNftAccountByOwner(ctx, cp.conn, user)
	if err != nil {
		return nil, err
	}
	if len(userPositionAccounts) == 0 {
		return nil, errors.New("empty result from rpc call")
	}

	positionAddresses := make([]solana.PublicKey, 0, len(userPositionAccounts))
	for _, v := range userPositionAccounts {
		positionAddresses = append(positionAddresses, DerivePositionAddress(v.PositionNft))
	}

	positionStates, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PositionAccount { return &cp_amm.PositionAccount{} },
	).FetchMultiple(
		ctx,
		positionAddresses,
		nil,
	)
	if err != nil {
		return nil, err
	}

	positionResult := make([]GetPositionsByUserResult, len(userPositionAccounts))
	for idx, account := range userPositionAccounts {
		positionState := positionStates[idx]
		if positionState != nil {
			positionResult = append(positionResult, GetPositionsByUserResult{
				PositionNftAccount: account.PositionNftAccount,
				Position:           positionAddresses[idx],
				PositionState:      positionState,
			})
		}
	}

	slices.SortFunc(positionResult, func(a, b GetPositionsByUserResult) int {
		liquidityA := new(big.Int).Add(
			new(big.Int).Add(
				a.PositionState.VestedLiquidity.BigInt(),
				a.PositionState.PermanentLockedLiquidity.BigInt(),
			),
			a.PositionState.UnlockedLiquidity.BigInt(),
		)

		liquidityB := new(big.Int).Add(
			new(big.Int).Add(
				b.PositionState.VestedLiquidity.BigInt(),
				b.PositionState.PermanentLockedLiquidity.BigInt(),
			),
			b.PositionState.UnlockedLiquidity.BigInt(),
		)

		return liquidityB.Cmp(liquidityA) // Descending
	})

	return positionResult, nil
}

func (cp *CpAMM) GetAllVestingsByPosition(ctx context.Context, position solana.PublicKey) ([]anchor.ProgramAccount[*cp_amm.PositionAccount], error) {
	positionState, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PositionAccount { return &cp_amm.PositionAccount{} },
	).All(
		ctx,
		CpAMMProgramId,
		cp_amm.PositionAccountDiscriminator,
		[]rpc.RPCFilter{
			helpers.VestingByPositionFilter(position),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(positionState) == 0 {
		return nil, fmt.Errorf("position account: %s not found", position.String())
	}

	return positionState, nil
}

func (cp CpAMM) IsLockedPosition(position *cp_amm.PositionAccount) bool {
	totalLockedLiquidity := new(big.Int).Add(position.VestedLiquidity.BigInt(), position.PermanentLockedLiquidity.BigInt())
	return totalLockedLiquidity.Cmp(big.NewInt(0)) == 1
}

func (cp CpAMM) IsPermanentLockedPosition(position *cp_amm.PositionAccount) bool {
	return position.PermanentLockedLiquidity.BigInt().Cmp(big.NewInt(0)) == 1
}

// CanUnlockPosition checks if a position can be unlocked based on its locking state and vesting schedules.
// This method evaluates whether a position is eligible for operations that require
// unlocked liquidity, such as removing all liquidity or closing the position. It checks both
// permanent locks and time-based vesting schedules.
func (cp CpAMM) CanUnlockPosition(
	positionState *cp_amm.PositionAccount,
	vestings []struct {
		Account      solana.PublicKey
		VestingState *cp_amm.VestingAccount
	},
	currentPoint *big.Int,
) (canUnlock bool, reason string) {
	if len(vestings) == 0 {
		return false, "len of vesting is zero"
	}

	// check if permanently locked
	if cp.IsPermanentLockedPosition(positionState) {
		return false, "position is permanently locked"
	}

	// we expect that should have only one vesting per position
	for _, vesting := range vestings {
		if !helpers.IsVestingComplete(vesting.VestingState, currentPoint) {
			return false, "position has incomplete vesting schedule"
		}
	}

	return true, ""
}

func (cp *CpAMM) IsPoolExist(ctx context.Context, pool solana.PublicKey) bool {
	out, err := anchor.NewPgAccounts(
		cp.conn,
		func() *cp_amm.PoolAccount { return &cp_amm.PoolAccount{} },
	).Fetch(
		ctx,
		pool,
		nil,
	)
	if err != nil || out == nil {
		return false
	}

	return true
}

// GetLiquidityDelta computes the liquidity delta based on the provided token amounts and sqrt price.
func (cp CpAMM) GetLiquidityDelta(param types.LiquidityDeltaParams) *big.Int {

	liquidityDeltaFromAmountA := helpers.GetLiquidityDeltaFromAmountA(
		param.MaxAmountTokenA,
		param.SqrtPrice,
		param.SqrtMaxPrice,
	)
	liquidityDeltaFromAmountB := helpers.GetLiquidityDeltaFromAmountB(
		param.MaxAmountTokenB,
		param.SqrtMinPrice,
		param.SqrtPrice,
	)

	if liquidityDeltaFromAmountA.Cmp(liquidityDeltaFromAmountB) <= 0 {
		return liquidityDeltaFromAmountA
	}
	return liquidityDeltaFromAmountB
}

// func GetSwapAmount(
// 	inAmount, sqrtPrice, liquidity, tradeFeeNumerator *big.Int,
// 	aToB bool,
// 	collectFeeMode uint,
// )

type GetQuoteResult struct {
	SwapInAmount     *big.Int
	ConsumedInAmount *big.Int
	SwapOutAmount    *big.Int
	MinSwapOutAmount *big.Int
	TotalFee         *big.Int
	PriceImpact      float64
}

func GetQuote(param types.GetQuoteParams) GetQuoteResult {

	if param.PoolState == nil {
		return GetQuoteResult{}
	}

	currentPoint := param.CurrentSlot
	if param.PoolState.ActivationPoint != 0 {
		currentPoint = param.CurrentTime
	}

	dynamicFeeParams := types.DynamicFeeParams{}
	dynamicFee := param.PoolState.PoolFees.DynamicFee
	if dynamicFee.Initialized != 0 {
		dynamicFeeParams = types.DynamicFeeParams{
			VolatilityAccumulator: dynamicFee.VolatilityAccumulator.BigInt(),
			BinStep:               dynamicFee.BinStep,
			VariableFeeControl:    dynamicFee.VariableFeeControl,
		}
	}

	tradeFeeNumerator := helpers.GetFeeNumerator(
		currentPoint,
		new(big.Int).SetUint64(param.PoolState.ActivationPoint),
		param.PoolState.PoolFees.BaseFee.NumberOfPeriod,
		new(big.Int).SetUint64(param.PoolState.PoolFees.BaseFee.PeriodFrequency),
		types.FeeSchedulerMode(param.PoolState.PoolFees.BaseFee.FeeSchedulerMode),
		new(big.Int).SetUint64(param.PoolState.PoolFees.BaseFee.CliffFeeNumerator),
		new(big.Int).SetUint64(param.PoolState.PoolFees.BaseFee.ReductionFactor),
		dynamicFeeParams,
	)

	actualAmountIn := param.InAmount
	if param.InputTokenInfo != nil {
		actualAmountIn = helpers.CalculateTransferFeeExcludedAmount(
			param.InAmount,
			param.InputTokenInfo.Mint,
			param.InputTokenInfo.CurrentEpoch,
		).Amount
	}
	aToB := param.PoolState.TokenAMint.Equals(param.InputTokenMint)

	out := helpers.GetSwapAmount(
		actualAmountIn,
		param.PoolState.SqrtPrice.BigInt(),
		param.PoolState.Liquidity.BigInt(),
		tradeFeeNumerator,
		aToB,
		types.CollectFeeMode(param.PoolState.CollectFeeMode),
	)

	actualAmountOut := param.InAmount
	if param.OutputTokenInfo != nil {
		actualAmountOut = helpers.CalculateTransferFeeExcludedAmount(
			param.InAmount,
			param.OutputTokenInfo.Mint,
			param.OutputTokenInfo.CurrentEpoch,
		).Amount
	}

	minSwapOutAmount := helpers.GetMinAmountWithSlippage(
		actualAmountOut,
		param.Slippage,
	)

	return GetQuoteResult{
		SwapInAmount:     param.InAmount,
		ConsumedInAmount: actualAmountIn,
		SwapOutAmount:    actualAmountOut,
		MinSwapOutAmount: minSwapOutAmount,
		TotalFee:         out.TotalFee,
		PriceImpact:      helpers.GetPriceImpact(out.NextSqrtPrice, param.PoolState.SqrtPrice.BigInt()),
	}

}
