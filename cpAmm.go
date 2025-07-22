package dammv2gosdk

import (
	"context"
	"dammv2GoSDK/anchor"
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"dammv2GoSDK/helpers"
	"dammv2GoSDK/maths"
	"dammv2GoSDK/types"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

var (
	// cpAMM program ID.
	//  CpAMMProgramId = solana.MustPublicKeyFromBase58("cpamdpZCGKUy5JxQXB4dcpGPiikHawvSWAd6mEn1sGG")
	CpAMMProgramId = cp_amm.ProgramID
)

// CpAMM SDK class to interact with the DAMM-V2.
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
	TokenAAta, TokenBAta solana.PublicKey
	CreateATAIxns        []solana.Instruction
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
				TokenAAta     solana.PublicKey
				TokenBAta     solana.PublicKey
				CreateATAIxns []solana.Instruction
			}{},
			fmt.Errorf("err from token A— %s: err from tokenB— %s", handleNilErr(a.Err), handleNilErr(b.Err))
	}

	prepareATAIxns := func() []solana.Instruction {
		ixns := make([]solana.Instruction, 0, 2)
		if a.Ix != nil {
			ixns = append(ixns, a.Ix)
		}
		if b.Ix != nil {
			ixns = append(ixns, b.Ix)
		}

		return slices.Clip(ixns)
	}

	return struct {
		TokenAAta     solana.PublicKey
		TokenBAta     solana.PublicKey
		CreateATAIxns []solana.Instruction
	}{
		TokenAAta:     a.AtaPubkey,
		TokenBAta:     b.AtaPubkey,
		CreateATAIxns: prepareATAIxns(),
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
) (*cp_amm.Instruction, error) {
	addLiquidityPtr := cp_amm.NewAddLiquidityInstruction(
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
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := addLiquidityPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	return addLiquidityPtr.SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()
}

// buildRemoveAllLiquidityInstruction builds an instruction to remove all liquidity from a position.
func (cp CpAMM) buildRemoveAllLiquidityInstruction(
	param types.BuildRemoveAllLiquidityInstructionParams,
) (*cp_amm.Instruction, error) {
	removeLiquidityPtr := cp_amm.NewRemoveLiquidityInstruction(
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
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := removeLiquidityPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}
	return removeLiquidityPtr.SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()
}

// buildClaimPositionFeeInstruction builds an instruction to claim fees accumulated by a position.
func (cp CpAMM) buildClaimPositionFeeInstruction(
	param types.ClaimPositionFeeInstructionParams,
) (*cp_amm.Instruction, error) {
	claimPositionFeePtr := cp_amm.NewClaimPositionFeeInstruction(
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
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := claimPositionFeePtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}
	return claimPositionFeePtr.SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()
}

// buildClosePositionInstruction builds an instruction to close a position.
func (cp CpAMM) ClosePosition(
	param types.ClosePositionParams,
) (*cp_amm.Instruction, error) {
	closePositonPtr := cp_amm.NewClosePositionInstruction(
		param.PositionNftMint,
		param.PositionNftAccount,
		param.Pool,
		param.Position,
		param.PoolAuthority,
		param.Owner,
		param.Owner,
		solana.Token2022ProgramID,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := closePositonPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}
	return closePositonPtr.SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()
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
	claimPositionFeeInstruction, err := cp.buildClaimPositionFeeInstruction(
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
	if err != nil {
		return nil, err
	}

	ixns = append(ixns, claimPositionFeeInstruction)

	// 2. remove all liquidity
	removeAllLiquidityInstruction, err := cp.buildRemoveAllLiquidityInstruction(
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
	if err != nil {
		return nil, err
	}

	ixns = append(ixns, removeAllLiquidityInstruction)

	// 3. close position
	closePositionInstruction, err := cp.ClosePosition(
		types.ClosePositionParams{
			Owner:              param.Owner,
			PoolAuthority:      cp.poolAuthority,
			Pool:               param.PositionState.Pool,
			Position:           param.Position,
			PositionNftMint:    param.PositionState.NftMint,
			PositionNftAccount: param.PositionNftAccount,
		},
	)
	if err != nil {
		return nil, err
	}

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

	createPositionPtr := cp_amm.NewCreatePositionInstruction(
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
		CpAMMProgramId,
	)

	eventAuthPDA, _, err := createPositionPtr.FindEventAuthorityAddress()
	if err != nil {
		return struct {
			Position          solana.PublicKey
			PositonNftAccount solana.PublicKey
			Ix                *cp_amm.Instruction
		}{}, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	ix, err := createPositionPtr.SetEventAuthorityAccount(eventAuthPDA).
		ValidateAndBuild()
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
) (*types.PrepareCreatePoolResponse, error) {

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

	preInstructions := make([]solana.Instruction, 0, len(res.CreateATAIxns)+2+2)
	preInstructions = append(preInstructions, res.CreateATAIxns...)

	if param.TokenAMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Payer,
			res.TokenAAta,
			param.TokenAAmount,
		)
		preInstructions = append(preInstructions, wrapSOLIx...)
	}
	if param.TokenBMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Payer,
			res.TokenBAta,
			param.TokenBAmount,
		)
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	return &types.PrepareCreatePoolResponse{
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
	tokenAIsSOL := param.TokenAMint.Equals(solana.WrappedSol)
	tokenBIsSOL := param.TokenBMint.Equals(solana.WrappedSol)
	hasSolToken := tokenAIsSOL || tokenBIsSOL

	tokenAOwner, tokenBOwner := param.Owner, param.Owner

	if !param.Receiver.IsZero() {
		tokenAOwner = param.Receiver
		if tokenAIsSOL {
			tokenAOwner = param.TempWSolAccount
		}

		tokenBOwner = param.Receiver
		if tokenBIsSOL {
			tokenBOwner = param.TempWSolAccount
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

	preInstructions = append(preInstructions, accs.CreateATAIxns...)

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

//// ANCHOR: GETTER/FETCHER FUNCTIONS //////

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
	vestings []types.Vesting,
	currentPoint *big.Int,
) (canUnlock bool, reason string) {

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

// GetQuote calculates swap quote based on input amount and pool state.
func GetQuote(param types.GetQuoteParams) types.GetQuoteResult {

	if param.PoolState == nil {
		return types.GetQuoteResult{}
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

	return types.GetQuoteResult{
		SwapInAmount:     param.InAmount,
		ConsumedInAmount: actualAmountIn,
		SwapOutAmount:    actualAmountOut,
		MinSwapOutAmount: minSwapOutAmount,
		TotalFee:         out.TotalFee,
		PriceImpact:      helpers.GetPriceImpact(out.NextSqrtPrice, param.PoolState.SqrtPrice.BigInt()),
	}
}

// GetQuoteExactOut calculates swap quote based on desired output amount and pool state.
func GetQuoteExactOut(param types.GetQuoteExactOutParams) (types.QuoteExactOutResult, error) {

	bToA := param.PoolState.TokenAMint.Equals(param.OutputTokenMint)
	tradeDirection := types.AtoB
	if bToA {
		tradeDirection = types.BtoA
	}

	currentPoint := param.CurrentSlot
	if param.PoolState.ActivationType != 0 {
		currentPoint = uint64(param.CurrentTime)
	}

	// var dynamicFeeParams types.DynamicFeeParams
	// if h := param.PoolState.PoolFees.DynamicFee; h.Initialized != 0 {
	// 	dynamicFeeParams.VolatilityAccumulator = h.VolatilityAccumulator.BigInt()
	// 	dynamicFeeParams.BinStep = h.BinStep
	// 	dynamicFeeParams.VariableFeeControl = h.VariableFeeControl
	// }

	actualAmountOut := param.OutAmount
	if h := param.OutputTokenInfo; h != nil {
		actualAmountOut = helpers.CalculateTransferFeeExcludedAmount(
			param.OutAmount,
			h.Mint,
			h.CurrentEpoch,
		).Amount
	}

	feeMode := helpers.GetFeeMode(types.CollectFeeMode(param.PoolState.CollectFeeMode), bToA)

	out, err := helpers.GetSwapResultFromOutAmount(
		param.PoolState,
		actualAmountOut,
		feeMode,
		tradeDirection,
		currentPoint,
	)
	if err != nil {
		return types.QuoteExactOutResult{}, err
	}

	actualInputAmount := out.InputAmount
	if h := param.InputTokenInfo; h != nil {
		actualInputAmount = helpers.CalculateTransferFeeExcludedAmount(
			out.InputAmount,
			h.Mint,
			h.CurrentEpoch,
		).Amount
	}

	slippageFactor := 1 + (param.Slippage / 100.0)
	maxInputFloat := new(big.Float).Mul(
		new(big.Float).SetInt(actualInputAmount),
		big.NewFloat(slippageFactor),
	)

	maxInputAmount, _ := maxInputFloat.Int(nil)

	priceImpact := helpers.GetPriceImpact(
		out.SwapResult.NextSqrtPrice,
		param.PoolState.SqrtPrice.BigInt(),
	)

	return types.QuoteExactOutResult{
		SwapResult:     out.SwapResult,
		InputAmount:    actualInputAmount,
		MaxInputAmount: maxInputAmount,
		PriceImpact:    priceImpact,
	}, nil
}

// GetDepositQuote calculates the deposit quote for liquidity pool.
func GetDepositQuote(param types.GetDepositQuoteParams) types.DepositQuote {

	actualAmountIn := param.InAmount
	if param.InputTokenInfo != nil {
		actualAmountIn = helpers.CalculateTransferFeeExcludedAmount(
			param.InAmount,
			param.InputTokenInfo.Mint,
			param.InputTokenInfo.CurrentEpoch,
		).Amount
	}

	var (
		liquidityDelta *big.Int
		rawAmount      func(*big.Int) *big.Int
	)
	if param.IsTokenA {
		liquidityDelta = helpers.GetLiquidityDeltaFromAmountA(
			actualAmountIn,
			param.SqrtPrice,
			param.MaxSqrtPrice,
		)
		rawAmount = func(delta *big.Int) *big.Int {
			return helpers.GetAmountBFromLiquidityDelta(
				delta,
				param.SqrtPrice,
				param.MinSqrtPrice,
				types.RoundingUp,
			)
		}
	} else {
		liquidityDelta = helpers.GetLiquidityDeltaFromAmountB(
			actualAmountIn,
			param.MinSqrtPrice,
			param.SqrtPrice,
		)
		rawAmount = func(delta *big.Int) *big.Int {
			return helpers.GetAmountAFromLiquidityDelta(
				delta,
				param.SqrtPrice,
				param.MaxSqrtPrice,
				types.RoundingUp,
			)
		}
	}

	rawOutputAmount := rawAmount(liquidityDelta)
	outputAmount := rawOutputAmount
	if param.OutputTokenInfo != nil {
		outputAmount = helpers.CalculateTransferFeeIncludedAmount(
			rawOutputAmount,
			param.OutputTokenInfo.Mint,
			param.OutputTokenInfo.CurrentEpoch,
		).Amount
	}

	return types.DepositQuote{
		ActualInputAmount:   actualAmountIn,
		ConsumedInputAmount: param.InAmount,
		LiquidityDelta:      liquidityDelta,
		OutputAmount:        outputAmount,
	}
}

type WithdrawQuote struct {
	// amount of liquidity that will be removed from the pool
	LiquidityDelta *big.Int
	// calculated amount of token A to be received (after deducting transfer fees)
	OutAmountA *big.Int
	// calculated amount of token B to be received (after deducting transfer fees)
	OutAmountB *big.Int
}

// GetWithdrawQuote calculates the withdrawal quote for removing liquidity from a concentrated liquidity pool.
//
// params.tokenATokenInfo - must provide if token a is token2022.
//
// params.tokenBTokenInfo - must provide if token b is token2022.
func GetWithdrawQuote(param types.GetWithdrawQuoteParams) WithdrawQuote {
	amountA := helpers.GetAmountAFromLiquidityDelta(
		param.LiquidityDelta,
		param.SqrtPrice,
		param.MaxSqrtPrice,
		types.RoundingDown,
	)

	amountB := helpers.GetAmountBFromLiquidityDelta(
		param.LiquidityDelta,
		param.SqrtPrice,
		param.MinSqrtPrice,
		types.RoundingDown,
	)

	outAmountA := amountA
	if param.TokenATokenInfo != nil {
		outAmountA = helpers.CalculateTransferFeeExcludedAmount(
			amountA,
			param.TokenATokenInfo.Mint,
			param.TokenATokenInfo.CurrentEpoch,
		).Amount
	}

	outAmountB := amountB
	if param.TokenBTokenInfo != nil {
		outAmountB = helpers.CalculateTransferFeeExcludedAmount(
			amountB,
			param.TokenBTokenInfo.Mint,
			param.TokenBTokenInfo.CurrentEpoch,
		).Amount
	}

	return WithdrawQuote{
		LiquidityDelta: param.LiquidityDelta,
		OutAmountA:     outAmountA,
		OutAmountB:     outAmountB,
	}
}

// Calculates liquidity and corresponding token amounts for token A single-sided pool creation.
// Only supports initialization where initial price equals min sqrt price, returns Calculated liquidity delta
//
// params - Parameters for single-sided pool creation.
func PreparePoolCreationSingleSide(param *types.PreparePoolCreationSingleSideParams) (*big.Int, error) {

	if param.InitSqrtPrice.Cmp(param.MinSqrtPrice) != 0 {
		return nil, errors.New("only support single side for base token")
	}
	actualAmountIn := param.TokenAAmount
	if param.TokenAInfo != nil {
		actualAmountIn = new(big.Int).Sub(
			param.TokenAAmount,
			helpers.CalculateTransferFeeIncludedAmount(
				param.TokenAAmount,
				param.TokenAInfo.Mint,
				param.TokenAInfo.CurrentEpoch,
			).TransferFee,
		)
	}
	return helpers.GetLiquidityDeltaFromAmountA(
		actualAmountIn,
		param.InitSqrtPrice,
		param.MaxSqrtPrice,
	), nil
}

func PreparePoolCreationParams(
	param types.PreparePoolCreationParams,
) (struct{ InitSqrtPrice, LiquidityDelta *big.Int }, error) {

	if param.TokenAAmount.Cmp(big.NewInt(0)) == 0 &&
		param.TokenBAmount.Cmp(big.NewInt(0)) == 0 {
		return struct {
			InitSqrtPrice  *big.Int
			LiquidityDelta *big.Int
		}{}, errors.New("invalid input amount")
	}

	actualAmountAIn := param.TokenAAmount
	if param.TokenAInfo != nil {
		actualAmountAIn = new(big.Int).Sub(
			param.TokenAAmount,
			helpers.CalculateTransferFeeIncludedAmount(
				param.TokenAAmount,
				param.TokenAInfo.Mint,
				param.TokenAInfo.CurrentEpoch,
			).TransferFee,
		)
	}

	actualAmountBIn := param.TokenBAmount
	if param.TokenBInfo != nil {
		actualAmountBIn = new(big.Int).Sub(
			param.TokenBAmount,
			helpers.CalculateTransferFeeIncludedAmount(
				param.TokenBAmount,
				param.TokenBInfo.Mint,
				param.TokenBInfo.CurrentEpoch,
			).TransferFee,
		)
	}
	initSqrtPrice, err := maths.CalculateInitSqrtPrice(
		param.TokenAAmount,
		param.TokenBAmount,
		param.MinSqrtPrice,
		param.MaxSqrtPrice,
	)
	if err != nil {
		return struct {
			InitSqrtPrice  *big.Int
			LiquidityDelta *big.Int
		}{}, err
	}

	liquidityDeltaFromAmountA := helpers.GetLiquidityDeltaFromAmountA(
		actualAmountAIn,
		initSqrtPrice,
		param.MaxSqrtPrice,
	)
	liquidityDeltaFromAmountB := helpers.GetLiquidityDeltaFromAmountB(
		actualAmountBIn,
		param.MinSqrtPrice,
		initSqrtPrice,
	)

	liquidityDelta := liquidityDeltaFromAmountB
	if liquidityDeltaFromAmountA.Cmp(liquidityDeltaFromAmountB) == -1 {
		liquidityDelta = liquidityDeltaFromAmountA
	}

	return struct {
		InitSqrtPrice  *big.Int
		LiquidityDelta *big.Int
	}{
		InitSqrtPrice:  initSqrtPrice,
		LiquidityDelta: liquidityDelta,
	}, nil
}

//// ANCHOR: MAIN ENDPOINT //////

func (cp *CpAMM) CreatePool(
	ctx context.Context, param types.CreatePoolParams,
) ([]solana.Instruction, error) {

	pool := DerivePoolAddress(
		param.Config,
		param.TokenAMint,
		param.TokenBMint,
	)

	createPoolParams, err := cp.prepareCreatePoolParams(
		ctx,
		types.PrepareCustomizablePoolParams{
			Pool:          pool,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAAmount:  param.TokenAAmount.Uint64(),
			TokenBAmount:  param.TokenBAmount.Uint64(),
			Payer:         param.Payer,
			PositionNft:   param.PositionNFT,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)

	if err != nil {
		return nil, err
	}

	postInstruction := make([]solana.Instruction, 0, 1)
	if param.IsLockLiquidity {
		permanentLockIx, err := cp_amm.NewPermanentLockPositionInstruction(
			param.LiquidityDelta,
			pool,
			createPoolParams.Positon,
			createPoolParams.PositionNftAccount,
			param.Creator,
			solana.PublicKey{},
			CpAMMProgramId,
		).ValidateAndBuild()
		if err != nil {
			return nil, err
		}
		postInstruction = append(postInstruction, permanentLockIx)
	}

	initPoolPtr := cp_amm.NewInitializePoolInstruction(
		cp_amm.InitializePoolParameters{
			Liquidity:       param.LiquidityDelta,
			SqrtPrice:       param.InitSqrtPrice,
			ActivationPoint: param.ActivationPoint,
		},
		param.Creator,
		param.PositionNFT,
		createPoolParams.PositionNftAccount,
		param.Payer,
		param.Config,
		cp.poolAuthority,
		pool,
		createPoolParams.Positon,
		param.TokenAMint,
		param.TokenBMint,
		createPoolParams.TokenAVault,
		createPoolParams.TokenBVault,
		createPoolParams.PayerTokenA,
		createPoolParams.PayerTokenB,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.Token2022ProgramID,
		solana.SystemProgramID,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	accountMetaSlice := slices.Grow(initPoolPtr.AccountMetaSlice, len(createPoolParams.TokenBadgeAccounts))
	accountMetaSlice = append(accountMetaSlice, createPoolParams.TokenBadgeAccounts...)
	initPoolPtr.AccountMetaSlice = accountMetaSlice

	currentIx, err := initPoolPtr.ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, 1+1+len(createPoolParams.Ixns))
	ixns = append(ixns, createPoolParams.Ixns...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstruction...)

	return ixns, nil
}

// CreateCustomPool builds a transaction to create a customizable pool.
func (cp *CpAMM) CreateCustomPool(
	ctx context.Context,
	param types.InitializeCustomizeablePoolParams,
) (struct {
	Pool, Position solana.PublicKey
	Ixns           []solana.Instruction
}, error) {

	pool := DeriveCustomizablePoolAddress(param.TokenAMint, param.TokenBMint)

	tokenBAmount := param.TokenBAmount
	if param.TokenBMint.Equals(solana.WrappedSol) && tokenBAmount == 0 {
		tokenBAmount = 1
	}

	createPoolParams, err := cp.prepareCreatePoolParams(
		ctx,
		types.PrepareCustomizablePoolParams{
			Pool:          pool,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAAmount:  param.TokenAAmount,
			TokenBAmount:  tokenBAmount,
			Payer:         param.Payer,
			PositionNft:   param.PositionNFT,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return struct {
			Pool     solana.PublicKey
			Position solana.PublicKey
			Ixns     []solana.Instruction
		}{}, err
	}

	postInstruction := make([]solana.Instruction, 0, 1)

	if param.IsLockLiquidity {
		permanentLockPtr := cp_amm.NewPermanentLockPositionInstruction(
			param.LiquidityDelta,
			pool,
			createPoolParams.Positon,
			createPoolParams.PositionNftAccount,
			param.Creator,
			solana.PublicKey{},
			CpAMMProgramId,
		)
		eventAuthPDA, _, err := permanentLockPtr.FindEventAuthorityAddress()
		if err != nil {
			return struct {
				Pool     solana.PublicKey
				Position solana.PublicKey
				Ixns     []solana.Instruction
			}{}, fmt.Errorf("err deriving eventAuthPDA: %w", err)
		}

		permanentLockIx, err := permanentLockPtr.
			SetEventAuthorityAccount(eventAuthPDA).
			ValidateAndBuild()
		if err != nil {
			return struct {
				Pool     solana.PublicKey
				Position solana.PublicKey
				Ixns     []solana.Instruction
			}{}, err
		}

		postInstruction = append(postInstruction, permanentLockIx)
	}

	initCustomizablePoolPtr := cp_amm.NewInitializeCustomizablePoolInstruction(
		cp_amm.InitializeCustomizablePoolParameters{
			PoolFees:        param.PoolFees,
			SqrtMinPrice:    param.SqrtMinPrice,
			SqrtMaxPrice:    param.SqrtMaxPrice,
			HasAlphaVault:   param.HasAlphaVault,
			Liquidity:       param.LiquidityDelta,
			SqrtPrice:       param.InitSqrtPrice,
			ActivationType:  param.ActivationType,
			CollectFeeMode:  param.CollectFeeMode,
			ActivationPoint: param.ActivationPoint,
		},
		param.Creator,
		param.PositionNFT,
		createPoolParams.PositionNftAccount,
		param.Payer,
		cp.poolAuthority,
		pool,
		createPoolParams.Positon,
		param.TokenAMint,
		param.TokenBMint,
		createPoolParams.TokenAVault,
		createPoolParams.TokenBVault,
		createPoolParams.PayerTokenA,
		createPoolParams.PayerTokenB,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.Token2022ProgramID,
		solana.SystemProgramID,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := initCustomizablePoolPtr.FindEventAuthorityAddress()
	if err != nil {
		return struct {
			Pool     solana.PublicKey
			Position solana.PublicKey
			Ixns     []solana.Instruction
		}{}, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}
	initCustomizablePoolPtr.SetEventAuthorityAccount(eventAuthPDA)

	accountMetaSlice := slices.Grow(initCustomizablePoolPtr.AccountMetaSlice, len(createPoolParams.TokenBadgeAccounts))
	accountMetaSlice = append(accountMetaSlice, createPoolParams.TokenBadgeAccounts...)
	initCustomizablePoolPtr.AccountMetaSlice = accountMetaSlice

	currentIx, err := initCustomizablePoolPtr.ValidateAndBuild()
	if err != nil {
		return struct {
			Pool     solana.PublicKey
			Position solana.PublicKey
			Ixns     []solana.Instruction
		}{}, err
	}

	ixns := make([]solana.Instruction, 0, 1+1+len(createPoolParams.Ixns))

	ixns = append(ixns, createPoolParams.Ixns...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstruction...)

	return struct {
		Pool     solana.PublicKey
		Position solana.PublicKey
		Ixns     []solana.Instruction
	}{
		Position: createPoolParams.Positon,
		Pool:     pool,
		Ixns:     ixns,
	}, nil
}

func (cp *CpAMM) CreateCustomPoolWithDynamicConfig(
	ctx context.Context,
	param types.InitializeCustomizeablePoolWithDynamicConfigParams,
) (struct {
	Pool, Position solana.PublicKey
	Ixns           []solana.Instruction
}, error) {

	pool := DerivePoolAddress(param.Config, param.TokenAMint, param.TokenBMint)
	createPoolParams, err := cp.prepareCreatePoolParams(
		ctx,
		types.PrepareCustomizablePoolParams{
			Pool:          pool,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAAmount:  param.TokenAAmount,
			TokenBAmount:  param.TokenBAmount,
			Payer:         param.Payer,
			PositionNft:   param.PositionNFT,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return struct {
			Pool     solana.PublicKey
			Position solana.PublicKey
			Ixns     []solana.Instruction
		}{}, err
	}

	postInstruction := make([]solana.Instruction, 0, 1)
	if param.IsLockLiquidity {
		permanentLockIx, err := cp_amm.NewPermanentLockPositionInstruction(
			param.LiquidityDelta,
			pool,
			createPoolParams.Positon,
			createPoolParams.PositionNftAccount,
			param.Creator,
			solana.PublicKey{},
			CpAMMProgramId,
		).ValidateAndBuild()
		if err != nil {
			return struct {
				Pool     solana.PublicKey
				Position solana.PublicKey
				Ixns     []solana.Instruction
			}{}, err
		}
		postInstruction = append(postInstruction, permanentLockIx)
	}

	initPoolWithDynamicConfigPtr := cp_amm.NewInitializePoolWithDynamicConfigInstruction(
		cp_amm.InitializeCustomizablePoolParameters{
			PoolFees:        param.PoolFees,
			SqrtMinPrice:    param.SqrtMinPrice,
			SqrtMaxPrice:    param.SqrtMaxPrice,
			HasAlphaVault:   param.HasAlphaVault,
			Liquidity:       param.LiquidityDelta,
			SqrtPrice:       param.InitSqrtPrice,
			ActivationType:  param.ActivationType,
			ActivationPoint: param.ActivationPoint,
			CollectFeeMode:  param.CollectFeeMode,
		},
		param.Creator,
		param.PositionNFT,
		createPoolParams.PositionNftAccount,
		param.Payer,
		param.PoolCreatorAuthority,
		param.Config,
		cp.poolAuthority,
		pool,
		createPoolParams.Positon,
		param.TokenAMint,
		param.TokenBMint,
		createPoolParams.TokenAVault,
		createPoolParams.TokenBVault,
		createPoolParams.PayerTokenA,
		createPoolParams.PayerTokenB,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.Token2022ProgramID,
		solana.SystemProgramID,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	accountMetaSlice := slices.Grow(initPoolWithDynamicConfigPtr.AccountMetaSlice, len(createPoolParams.TokenBadgeAccounts))
	accountMetaSlice = append(accountMetaSlice, createPoolParams.TokenBadgeAccounts...)
	initPoolWithDynamicConfigPtr.AccountMetaSlice = accountMetaSlice

	currentIx, err := initPoolWithDynamicConfigPtr.ValidateAndBuild()
	if err != nil {
		return struct {
			Pool     solana.PublicKey
			Position solana.PublicKey
			Ixns     []solana.Instruction
		}{}, err
	}

	ixns := make([]solana.Instruction, 0, 1+1+len(createPoolParams.Ixns))
	ixns = append(ixns, createPoolParams.Ixns...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstruction...)

	return struct {
		Pool     solana.PublicKey
		Position solana.PublicKey
		Ixns     []solana.Instruction
	}{
		Position: createPoolParams.Positon,
		Pool:     pool,
		Ixns:     ixns,
	}, nil
}

// CreatePosition builds a instructions to create a position.
func (cp *CpAMM) CreatePosition(
	param types.CreatePositionParams,
) (struct {
	Position          solana.PublicKey
	PositonNftAccount solana.PublicKey
	Ix                *cp_amm.Instruction
}, error) {
	return cp.buildCreatePositionInstruction(param)
}

// AddLiquidity builds instruction to add liquidity to an existing position.
func (cp *CpAMM) AddLiquidity(
	ctx context.Context,
	param types.AddLiquidityParams,
) ([]solana.Instruction, error) {

	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Owner,
			TokenAOwner:   param.Owner,
			TokenBOwner:   param.Owner,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	preInstructions := make([]solana.Instruction, 0, 2)
	preInstructions = append(preInstructions, preparedTokenAccs.CreateATAIxns...)

	if param.TokenAMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Owner,
			preparedTokenAccs.TokenAAta,
			param.MaxAmountTokenA,
		)
		preInstructions = slices.Grow(preInstructions, len(wrapSOLIx))
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	if param.TokenBMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Owner,
			preparedTokenAccs.TokenBAta,
			param.MaxAmountTokenB,
		)
		preInstructions = slices.Grow(preInstructions, len(wrapSOLIx))
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.TokenAMint.Equals(solana.WrappedSol) ||
		param.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		if closeWrappedSOLIx != nil {
			postInstructions = append(postInstructions, closeWrappedSOLIx)
		}
	}

	addLiquidityInstruction, err := cp.buildAddLiquidityInstruction(
		types.BuildAddLiquidityParams{
			Pool:                  param.Pool,
			Position:              param.Position,
			PositionNftAccount:    param.PositionNftAccount,
			Owner:                 param.Owner,
			TokenAAccount:         preparedTokenAccs.TokenAAta,
			TokenBAccount:         preparedTokenAccs.TokenBAta,
			TokenAMint:            param.TokenAMint,
			TokenBMint:            param.TokenBMint,
			TokenAVault:           param.TokenAVault,
			TokenBVault:           param.TokenBVault,
			TokenAProgram:         param.TokenAProgram,
			TokenBProgram:         param.TokenBProgram,
			LiquidityDelta:        param.LiquidityDelta,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
		},
	)
	if err != nil {
		return nil, err
	}

	res := make([]solana.Instruction, 0, len(preInstructions)+len(postInstructions)+1)
	res = append(res, preInstructions...)
	res = append(res, addLiquidityInstruction)
	res = append(res, postInstructions...)
	return res, nil
}

// CreatePositionAndAddLiquidity creates a new position and add liquidity to position it in a single transaction.
// Handles both native SOL and other tokens, automatically wrapping/unwrapping SOL as needed.
func (cp *CpAMM) CreatePositionAndAddLiquidity(
	ctx context.Context,
	param types.CreatePositionAndAddLiquidity,
) ([]solana.Instruction, error) {

	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Owner,
			TokenAOwner:   param.Owner,
			TokenBOwner:   param.Owner,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}
	tokenAVault := DeriveTokenVaultAddress(param.TokenAMint, param.Pool)
	tokenBVault := DeriveTokenVaultAddress(param.TokenBMint, param.Pool)

	preInstructions := make([]solana.Instruction, 0, 2)
	preInstructions = append(preInstructions, preparedTokenAccs.CreateATAIxns...)
	if param.TokenAMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Owner,
			preparedTokenAccs.TokenAAta,
			param.MaxAmountTokenA,
		)
		preInstructions = slices.Grow(preInstructions, len(wrapSOLIx))
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	if param.TokenBMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Owner,
			preparedTokenAccs.TokenBAta,
			param.MaxAmountTokenB,
		)
		preInstructions = slices.Grow(preInstructions, len(wrapSOLIx))
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.TokenAMint.Equals(solana.WrappedSol) ||
		param.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		if closeWrappedSOLIx != nil {
			postInstructions = append(postInstructions, closeWrappedSOLIx)
		}
	}

	buildCreatePositionIns, err := cp.buildCreatePositionInstruction(
		types.CreatePositionParams{
			Owner:       param.Owner,
			Payer:       param.Owner,
			Pool:        param.Pool,
			PositionNft: param.PositionNFT,
		},
	)
	if err != nil {
		return nil, err
	}

	addLiquidityInstruction, err := cp.buildAddLiquidityInstruction(
		types.BuildAddLiquidityParams{
			Pool:                  param.Pool,
			Position:              buildCreatePositionIns.Position,
			PositionNftAccount:    buildCreatePositionIns.PositonNftAccount,
			Owner:                 param.Owner,
			TokenAAccount:         preparedTokenAccs.TokenAAta,
			TokenBAccount:         preparedTokenAccs.TokenBAta,
			TokenAMint:            param.TokenAMint,
			TokenBMint:            param.TokenBMint,
			TokenAVault:           tokenAVault,
			TokenBVault:           tokenBVault,
			TokenAProgram:         param.TokenAProgram,
			TokenBProgram:         param.TokenBProgram,
			LiquidityDelta:        param.LiquidityDelta,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
		},
	)

	if err != nil {
		return nil, err
	}

	res := make([]solana.Instruction, 0, len(preInstructions)+len(postInstructions)+1)
	res = append(res, preInstructions...)
	res = append(res, addLiquidityInstruction)
	res = append(res, postInstructions...)
	return res, nil
}

// RemoveLiquidity builds instruction to remove liquidity from a position.
func (cp *CpAMM) RemoveLiquidity(
	ctx context.Context,
	param types.RemoveLiquidityParams,
) ([]solana.Instruction, error) {

	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Owner,
			TokenAOwner:   param.Owner,
			TokenBOwner:   param.Owner,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.TokenAMint.Equals(solana.WrappedSol) ||
		param.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		if closeWrappedSOLIx != nil {
			postInstructions = append(postInstructions, closeWrappedSOLIx)
		}
	}

	preInstructions := make([]solana.Instruction, 0, 2)
	if len(param.Vestings) > 0 {
		vestingAccouts := make([]solana.PublicKey, len(param.Vestings))
		for i, v := range param.Vestings {
			vestingAccouts[i] = v.Account
		}
		refreshVestingInstruction, err := cp.buildRefreshVestingInstruction(
			types.RefreshVestingParams{
				Owner:              param.Owner,
				Position:           param.Position,
				PositionNftAccount: param.PositionNftAccount,
				Pool:               param.Pool,
				VestingAccounts:    vestingAccouts,
			},
		)
		if err != nil {
			return nil, err
		}

		preInstructions = append(preInstructions, refreshVestingInstruction)
	}

	removeLiquidityPtr := cp_amm.NewRemoveLiquidityInstruction(
		cp_amm.RemoveLiquidityParameters{
			LiquidityDelta:        param.LiquidityDelta,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
		},
		cp.poolAuthority,
		param.Pool,
		param.Position,
		preparedTokenAccs.TokenAAta,
		preparedTokenAccs.TokenBAta,
		param.TokenAVault,
		param.TokenBVault,
		param.TokenAMint,
		param.TokenBMint,
		param.PositionNftAccount,
		param.Owner,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := removeLiquidityPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	currentIx, err := removeLiquidityPtr.
		SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(preInstructions)+1+len(postInstructions))
	ixns = append(ixns, preInstructions...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}

// RemoveaAllLiquidity builds instruction to remove all liquidity from a position.
func (cp *CpAMM) RemoveALLLiquidity(
	ctx context.Context,
	param types.RemoveAllLiquidityParams,
) ([]solana.Instruction, error) {

	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Owner,
			TokenAOwner:   param.Owner,
			TokenBOwner:   param.Owner,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.TokenAMint.Equals(solana.WrappedSol) ||
		param.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		if closeWrappedSOLIx != nil {
			postInstructions = append(postInstructions, closeWrappedSOLIx)
		}
	}

	preInstructions := make([]solana.Instruction, 0, 2)
	if len(param.Vestings) > 0 {
		vestingAccouts := make([]solana.PublicKey, len(param.Vestings))
		for i, v := range param.Vestings {
			vestingAccouts[i] = v.Account
		}
		refreshVestingInstruction, err := cp.buildRefreshVestingInstruction(
			types.RefreshVestingParams{
				Owner:              param.Owner,
				Position:           param.Position,
				PositionNftAccount: param.PositionNftAccount,
				Pool:               param.Pool,
				VestingAccounts:    vestingAccouts,
			},
		)
		if err != nil {
			return nil, err
		}

		preInstructions = append(preInstructions, refreshVestingInstruction)
	}

	removeLiquidityPtr := cp_amm.NewRemoveAllLiquidityInstruction(
		param.TokenAAmountThreshold,
		param.TokenBAmountThreshold,
		cp.poolAuthority,
		param.Pool,
		param.Position,
		preparedTokenAccs.TokenAAta,
		preparedTokenAccs.TokenBAta,
		param.TokenAVault,
		param.TokenBVault,
		param.TokenAMint,
		param.TokenBMint,
		param.PositionNftAccount,
		param.Owner,
		param.TokenAProgram,
		param.TokenBProgram,
		solana.PublicKey{},
		CpAMMProgramId,
	)

	eventAuthPDA, _, err := removeLiquidityPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	currentIx, err := removeLiquidityPtr.
		SetEventAuthorityAccount(eventAuthPDA).
		ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(preInstructions)+1+len(postInstructions))
	ixns = append(ixns, preInstructions...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}

// Swap builds instruction to perform a swap in the pool.
func (cp *CpAMM) Swap(
	ctx context.Context,
	param types.SwapParams,
) ([]solana.Instruction, error) {

	inputTokenProgram, outputTokenProgram := param.TokenBProgram, param.TokenAProgram
	if param.InputTokenMint.Equals(param.TokenAMint) {
		inputTokenProgram, outputTokenProgram = param.TokenAProgram, param.TokenBProgram
	}

	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Payer,
			TokenAOwner:   param.Payer,
			TokenBOwner:   param.Payer,
			TokenAMint:    param.InputTokenMint,
			TokenBMint:    param.OutputTokenMint,
			TokenAProgram: inputTokenProgram,
			TokenBProgram: outputTokenProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	preInstructions := make([]solana.Instruction, 0, 2)
	preInstructions = append(preInstructions, preparedTokenAccs.CreateATAIxns...)

	if param.InputTokenMint.Equals(solana.WrappedSol) {
		wrapSOLIx := helpers.WrapSOLInstruction(
			param.Payer,
			preparedTokenAccs.TokenAAta,
			param.AmountIn,
		)
		preInstructions = slices.Grow(preInstructions, len(wrapSOLIx))
		preInstructions = append(preInstructions, wrapSOLIx...)
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.TokenAMint.Equals(solana.WrappedSol) ||
		param.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Payer, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}

		postInstructions = append(postInstructions, closeWrappedSOLIx)
	}

	swapPtr := cp_amm.NewSwapInstruction(
		cp_amm.SwapParameters{
			AmountIn:         param.AmountIn,
			MinimumAmountOut: param.MinimumAmountOut,
		},
		cp.poolAuthority,
		param.Pool,
		preparedTokenAccs.TokenAAta,
		preparedTokenAccs.TokenBAta,
		param.TokenAVault,
		param.TokenBVault,
		param.TokenAMint,
		param.TokenBMint,
		param.Payer,
		param.TokenAProgram,
		param.TokenBProgram,
		param.ReferralTokenAccount,
		solana.PublicKey{},
		CpAMMProgramId,
	)

	// swapPtr := cp_amm.NewSwapInstructionBuilder().
	// 	SetParams(cp_amm.SwapParameters{
	// 		AmountIn:         param.AmountIn,
	// 		MinimumAmountOut: param.MinimumAmountOut,
	// 	}).
	// 	SetPoolAuthorityAccount(cp.poolAuthority).
	// 	SetPoolAccount(param.Pool).
	// 	SetInputTokenAccountAccount(preparedTokenAccs.TokenAAta).
	// 	SetOutputTokenAccountAccount(preparedTokenAccs.TokenBAta).
	// 	SetTokenAVaultAccount(param.TokenAVault).
	// 	SetTokenBVaultAccount(param.TokenBVault).
	// 	SetTokenAMintAccount(param.TokenAMint).
	// 	SetTokenBMintAccount(param.TokenBMint).
	// 	SetPayerAccount(param.Payer).
	// 	SetTokenAProgramAccount(param.TokenAProgram).
	// 	SetTokenBProgramAccount(param.TokenBProgram).
	// 	SetProgramAccount(CpAMMProgramId)

	if param.ReferralTokenAccount.IsZero() {
		swapPtr.AccountMetaSlice[11] = nil
		fmt.Println("it hit here")
	}

	// 	swapPtr.SetReferralTokenAccountAccount(param.ReferralTokenAccount)
	// }
	// if param.ReferralTokenAccount.IsZero() {
	// 	// drop slot 11 completely (indexing starts at 0)
	// 	slice := swapPtr.AccountMetaSlice
	// 	swapPtr.AccountMetaSlice = append(slice[:11], slice[12:]...)
	// } else {
	// 	swapPtr.SetReferralTokenAccountAccount(param.ReferralTokenAccount)
	// }

	eventAuthPDA, _, err := swapPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	swapIx, err := swapPtr.SetEventAuthorityAccount(eventAuthPDA).
		ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(preInstructions)+1+len(postInstructions))
	ixns = append(ixns, preInstructions...)
	ixns = append(ixns, swapIx)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}

func (cp *CpAMM) LockPosition(
	param types.LockPositionParams,
) (*cp_amm.Instruction, error) {
	lockPositionPtr := cp_amm.NewLockPositionInstruction(
		cp_amm.VestingParameters{
			CliffPoint:           param.CliffPoint,
			PeriodFrequency:      param.PeriodFrequency,
			CliffUnlockLiquidity: param.CliffUnlockLiquidity,
			LiquidityPerPeriod:   param.LiquidityPerPeriod,
			NumberOfPeriod:       param.NumberOfPeriod,
		},
		param.Pool,
		param.Position,
		param.VestingAccount,
		param.PositionNftAccount,
		param.Owner,
		param.Payer,
		solana.SystemProgramID,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := lockPositionPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	return lockPositionPtr.SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()

}

// PermanentLockPosition builds a transaction to permanently lock a position.
func (cp *CpAMM) PermanentLockPosition(
	param types.PermanentLockParams,
) (*cp_amm.Instruction, error) {
	permanentLockPositionPtr := cp_amm.NewPermanentLockPositionInstruction(
		param.UnlockedLiquidity,
		param.Pool,
		param.Position,
		param.PositionNftAccount,
		param.Owner,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	eventAuthPDA, _, err := permanentLockPositionPtr.FindEventAuthorityAddress()
	if err != nil {
		return nil, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}

	return permanentLockPositionPtr.SetEventAuthorityAccount(eventAuthPDA).ValidateAndBuild()
}

// RefreshVesting builds a transaction to refresh vesting status of a position.
func (cp *CpAMM) RefreshVesting(
	param types.RefreshVestingParams,
) (*cp_amm.Instruction, error) {
	return cp.buildRefreshVestingInstruction(param)
}

//	RemoveAllLiquidityAndClosePosition builds instructions to remove all liquidity from a position and close it.
//
// This combines several operations in a single transaction:
//
// 1. Claims any accumulated fees
//
// 2. Removes all liquidity
//
// 3. Closes the position
func (cp *CpAMM) RemoveAllLiquidityAndClosePosition(
	ctx context.Context,
	param types.RemoveAllLiquidityAndClosePositionParams,
) ([]solana.Instruction, error) {

	canUnlock, reason := cp.CanUnlockPosition(
		param.PositionState,
		param.Vestings,
		new(big.Int).SetUint64(param.CurrentPoint),
	)

	if !canUnlock {
		return nil, fmt.Errorf("cannot remove liquidity: %s", reason)
	}

	tokenAProgram := helpers.GetTokenProgram(param.PoolState.TokenAFlag)
	tokenBProgram := helpers.GetTokenProgram(param.PoolState.TokenBFlag)
	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Owner,
			TokenAOwner:   param.Owner,
			TokenBOwner:   param.Owner,
			TokenAMint:    param.PoolState.TokenAMint,
			TokenBMint:    param.PoolState.TokenBMint,
			TokenAProgram: tokenAProgram,
			TokenBProgram: tokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.PoolState.TokenAMint.Equals(solana.WrappedSol) ||
		param.PoolState.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		postInstructions = append(postInstructions, closeWrappedSOLIx)
	}

	// 1. refresh vesting if vesting account provided
	preInstructions := make([]solana.Instruction, 0, 3)
	preInstructions = append(preInstructions, preparedTokenAccs.CreateATAIxns...)
	if len(param.Vestings) > 0 {
		vestingAccouts := make([]solana.PublicKey, len(param.Vestings))
		for i, v := range param.Vestings {
			vestingAccouts[i] = v.Account
		}
		refreshVestingInstruction, err := cp.buildRefreshVestingInstruction(
			types.RefreshVestingParams{
				Owner:              param.Owner,
				Position:           param.Position,
				PositionNftAccount: param.PositionNftAccount,
				Pool:               param.PositionState.Pool,
				VestingAccounts:    vestingAccouts,
			},
		)
		if err != nil {
			return nil, err
		}
		preInstructions = append(preInstructions, refreshVestingInstruction)
	}

	// 2. claim fee, remove liquidity and close position
	liquidatePositionInstructions, err := cp.buildRemoveAllLiquidityInstruction(
		types.BuildRemoveAllLiquidityInstructionParams{
			PoolAuthority:         cp.poolAuthority,
			Owner:                 param.Owner,
			Pool:                  param.PositionState.Pool,
			Position:              param.Position,
			PositionNftAccount:    param.PositionNftAccount,
			TokenAAccount:         preparedTokenAccs.TokenAAta,
			TokenBAccount:         preparedTokenAccs.TokenBAta,
			TokenAAmountThreshold: param.TokenAAmountThreshold,
			TokenBAmountThreshold: param.TokenBAmountThreshold,
		},
	)
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(preInstructions)+1+len(postInstructions))
	ixns = append(ixns, preInstructions...)
	ixns = append(ixns, liquidatePositionInstructions)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}

// MergePosition builds instructions to merge liquidity from one position into another.
//
// This process:
//
// 1. Claims fees from the source position.
//
// 2. Removes all liquidity from the source position.
//
// 3. Adds that liquidity to the target position.
//
// 4. Closes the source position.
//
// an error means either the position is locked or incompatible.
func (cp *CpAMM) MergePosition(
	ctx context.Context,
	param types.MergePositionParams,
) ([]solana.Instruction, error) {

	canUnlock, reason := cp.CanUnlockPosition(
		param.PositionBState,
		param.PositionBVestings,
		new(big.Int).SetUint64(param.CurrentPoint),
	)

	if !canUnlock {
		return nil, fmt.Errorf("cannot remove liquidity: %s", reason)
	}

	tokenAProgram := helpers.GetTokenProgram(param.PoolState.TokenAFlag)
	tokenBProgram := helpers.GetTokenProgram(param.PoolState.TokenBFlag)
	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         param.Owner,
			TokenAOwner:   param.Owner,
			TokenBOwner:   param.Owner,
			TokenAMint:    param.PoolState.TokenAMint,
			TokenBMint:    param.PoolState.TokenBMint,
			TokenAProgram: tokenAProgram,
			TokenBProgram: tokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, 3)
	ixns = append(ixns, preparedTokenAccs.CreateATAIxns...)

	positionBLiquidityDelta := param.PositionBState.UnlockedLiquidity.BigInt()

	// 1. refresh vesting position B if vesting account provided
	if len(param.PositionBVestings) > 0 {
		vestingAccouts := make([]solana.PublicKey, 0, len(param.PositionBVestings))
		currentPoint, totalAvailableVestingLiquidity := new(big.Int).SetUint64(param.CurrentPoint), big.NewInt(0)
		for _, v := range param.PositionBVestings {
			available := helpers.GetAvailableVestingLiquidity(
				v.VestingState,
				currentPoint,
			)

			totalAvailableVestingLiquidity.Add(totalAvailableVestingLiquidity, available)
			vestingAccouts = append(vestingAccouts, v.Account)
		}

		positionBLiquidityDelta = new(big.Int).Add(
			positionBLiquidityDelta,
			totalAvailableVestingLiquidity,
		)

		refreshVestingInstruction, err := cp.buildRefreshVestingInstruction(
			types.RefreshVestingParams{
				Owner:              param.Owner,
				Position:           param.PositionB,
				PositionNftAccount: param.PositionBNftAccount,
				Pool:               param.PositionBState.Pool,
				VestingAccounts:    vestingAccouts,
			},
		)
		if err != nil {
			return nil, err
		}

		ixns = append(ixns, refreshVestingInstruction)
	}

	// recalculate liquidity delta
	tokenAWithdrawAmount := helpers.GetAmountAFromLiquidityDelta(
		positionBLiquidityDelta,
		param.PoolState.SqrtPrice.BigInt(),
		param.PoolState.SqrtMaxPrice.BigInt(),
		types.RoundingDown,
	)

	tokenBWithdrawAmount := helpers.GetAmountBFromLiquidityDelta(
		positionBLiquidityDelta,
		param.PoolState.SqrtPrice.BigInt(),
		param.PoolState.SqrtMinPrice.BigInt(),
		types.RoundingDown,
	)

	newLiquidityDelta := cp.GetLiquidityDelta(
		types.LiquidityDeltaParams{
			MaxAmountTokenA: tokenAWithdrawAmount,
			MaxAmountTokenB: tokenBWithdrawAmount,
			SqrtMaxPrice:    param.PoolState.SqrtMaxPrice.BigInt(),
			SqrtMinPrice:    param.PoolState.SqrtMinPrice.BigInt(),
			SqrtPrice:       param.PoolState.SqrtPrice.BigInt(),
		},
	)

	newLiquidityDeltaU128, err := helpers.BigIntToUint128(newLiquidityDelta)
	if err != nil {
		return nil, err
	}

	// 2. claim fee, remove liquidity and close position
	liquidatePositionInstructions, err := cp.buildLiquidatePositionInstruction(
		types.BuildLiquidatePositionInstructionParams{
			Owner:                 param.Owner,
			Position:              param.PositionB,
			PositionNftAccount:    param.PositionBNftAccount,
			PositionState:         param.PositionBState,
			PoolState:             param.PoolState,
			TokenAAccount:         preparedTokenAccs.TokenAAta,
			TokenBAccount:         preparedTokenAccs.TokenBAta,
			TokenAAmountThreshold: param.TokenAAmountRemoveLiquidityThreshold,
			TokenBAmountThreshold: param.TokenBAmountRemoveLiquidityThreshold,
		},
	)
	if err != nil {
		return nil, err
	}

	tempIxns := make([]solana.Instruction, len(liquidatePositionInstructions))
	for i, v := range liquidatePositionInstructions {
		tempIxns[i] = v
	}

	ixns = slices.Grow(ixns, len(tempIxns)+2)
	ixns = append(ixns, tempIxns...)

	// 3. add liquidity from position B to positon A
	addLiquidityInstruction, err := cp.buildAddLiquidityInstruction(
		types.BuildAddLiquidityParams{
			Pool:                  param.PositionBState.Pool,
			Position:              param.PositionA,
			PositionNftAccount:    param.PositionANftAccount,
			Owner:                 param.Owner,
			TokenAAccount:         preparedTokenAccs.TokenAAta,
			TokenBAccount:         preparedTokenAccs.TokenBAta,
			TokenAMint:            param.PoolState.TokenAMint,
			TokenBMint:            param.PoolState.TokenBMint,
			TokenAVault:           param.PoolState.TokenAVault,
			TokenBVault:           param.PoolState.TokenBVault,
			TokenAProgram:         tokenAProgram,
			TokenBProgram:         tokenBProgram,
			LiquidityDelta:        newLiquidityDeltaU128,
			TokenAAmountThreshold: param.TokenAAmountAddLiquidityThreshold,
			TokenBAmountThreshold: param.TokenBAmountAddLiquidityThreshold,
		},
	)
	if err != nil {
		return nil, err
	}

	ixns = append(ixns, addLiquidityInstruction)

	if param.PoolState.TokenAMint.Equals(solana.WrappedSol) ||
		param.PoolState.TokenBMint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		ixns = append(ixns, closeWrappedSOLIx)
	}

	return slices.Clip(ixns), nil
}

// UpdateRewardDuration builds instruction to update reward duration.
func (cp *CpAMM) UpdateRewardDuration(
	param types.UpdateRewardDurationParams,
) (*cp_amm.Instruction, error) {
	return cp_amm.NewUpdateRewardDurationInstruction(
		param.RewardIndex,
		param.NewDuration,
		param.Pool,
		param.Admin,
		solana.PublicKey{},
		CpAMMProgramId,
	).ValidateAndBuild()
}

// UpdateRewardDuration builds instruction tto update reward funder address.
func (cp *CpAMM) UpdateRewardFunder(
	param types.UpdateRewardFunderParams,
) (*cp_amm.Instruction, error) {
	return cp_amm.NewUpdateRewardFunderInstruction(
		param.RewardIndex,
		param.NewFunder,
		param.Pool,
		param.Admin,
		solana.PublicKey{},
		CpAMMProgramId,
	).ValidateAndBuild()
}

// fundReward builds instructions to fund rewards in a pool.
//
// TODO: UNCLEAR*
func (cp *CpAMM) fundReward(
	ctx context.Context,
	param types.FundRewardParams,
) ([]solana.Instruction, error) {

	poolState, err := cp.FetchPoolState(ctx, param.Pool)
	if err != nil {
		return nil, err
	}

	preInstructions := make([]solana.Instruction, 0, 3)

	rewardInfo := poolState.RewardInfos[param.RewardIndex]
	tokenProgram := helpers.GetTokenProgram(param.RewardIndex)
	funderTokenAccount, createFunderTokenAccountIx, err := helpers.GetOrCreateATAInstruction(
		ctx,
		cp.conn,
		rewardInfo.Mint,
		param.Funder,
		param.Funder,
		true,
		tokenProgram,
	)
	if err != nil {
		return nil, err
	}

	preInstructions = append(preInstructions, createFunderTokenAccountIx)

	// TODO: check case reward mint is wSOL && carryForward is true => total amount > amount
	if rewardInfo.Mint.Equals(solana.WrappedSol) ||
		param.Amount != 0 {
		closeWrappedSOLIx := helpers.WrapSOLInstruction(
			param.Funder, funderTokenAccount, param.Amount,
		)

		preInstructions = append(preInstructions, closeWrappedSOLIx...)
	}

	ix, err := cp_amm.NewFundRewardInstruction(
		param.RewardIndex,
		param.Amount,
		param.CarryForward,
		param.Pool,
		rewardInfo.Vault,
		rewardInfo.Mint,
		funderTokenAccount,
		param.Funder,
		tokenProgram,
		solana.PublicKey{},
		CpAMMProgramId,
	).ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	preInstructions = append(preInstructions, ix)
	return preInstructions, nil
}

// WithdrawIneligibleReward builds instructions to withdraw ineligible rewards from a pool.
func (cp *CpAMM) WithdrawIneligibleReward(
	ctx context.Context,
	param types.WithdrawIneligibleRewardParams,
) ([]solana.Instruction, error) {
	poolState, err := cp.FetchPoolState(ctx, param.Pool)
	if err != nil {
		return nil, err
	}

	preInstructions := make([]solana.Instruction, 0, 1)
	postInstructions := make([]solana.Instruction, 0, 1)

	rewardInfo := poolState.RewardInfos[param.RewardIndex]
	tokenProgram := helpers.GetTokenProgram(rewardInfo.RewardTokenFlag)
	funderTokenAccount, createFunderTokenAccountIx, err := helpers.GetOrCreateATAInstruction(
		ctx,
		cp.conn,
		rewardInfo.Mint,
		param.Funder,
		param.Funder,
		true,
		tokenProgram,
	)
	if err != nil {
		return nil, err
	}
	preInstructions = append(preInstructions, createFunderTokenAccountIx)

	// TODO: check case reward mint is wSOL && carryForward is true => total amount > amount
	if rewardInfo.Mint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Funder, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		postInstructions = append(postInstructions, closeWrappedSOLIx)
	}

	withdrawIneligibleRewardPtr := cp_amm.NewWithdrawIneligibleRewardInstruction(
		param.RewardIndex,
		cp.poolAuthority,
		param.Pool,
		rewardInfo.Vault,
		rewardInfo.Mint,
		funderTokenAccount,
		poolState.Partner,
		tokenProgram,
		solana.PublicKey{},
		CpAMMProgramId,
	)

	currentIx, err := withdrawIneligibleRewardPtr.ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(preInstructions)+1+len(postInstructions))
	ixns = append(ixns, preInstructions...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}

func (cp *CpAMM) ClaimPartnerFee(
	ctx context.Context,
	param types.ClaimPartnerFeeParams,
) ([]solana.Instruction, error) {

	poolState, err := cp.FetchPoolState(ctx, param.Pool)
	if err != nil {
		return nil, err
	}

	tokenAProgram := helpers.GetTokenProgram(poolState.TokenAFlag)
	tokenBProgram := helpers.GetTokenProgram(poolState.TokenBFlag)

	payer := param.Partner
	if !param.FeePayer.IsZero() {
		payer = param.FeePayer
	}

	out, err := cp.setupFeeClaimAccounts(
		ctx,
		types.SetupFeeClaimAccountsParams{
			Payer:           payer,
			Owner:           param.Partner,
			TokenAMint:      poolState.TokenAMint,
			TokenBMint:      poolState.TokenBMint,
			TokenAProgram:   tokenAProgram,
			TokenBProgram:   tokenBProgram,
			Receiver:        param.Receiver,
			TempWSolAccount: param.TempWSolAccount,
		},
	)
	if err != nil {
		return nil, err
	}

	claimPartnerFeePtr := cp_amm.NewClaimPartnerFeeInstruction(
		param.MaxAmountA,
		param.MaxAmountB,
		cp.poolAuthority,
		param.Pool,
		out.TokenAAccount,
		out.TokenBAccount,
		poolState.TokenAVault,
		poolState.TokenBVault,
		poolState.TokenAMint,
		poolState.TokenBMint,
		param.Partner,
		tokenAProgram,
		tokenBProgram,
		solana.PublicKey{},
		CpAMMProgramId,
	)
	currentIx, err := claimPartnerFeePtr.ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(out.PreInstructions)+1+len(out.PostInstructions))
	ixns = append(ixns, out.PreInstructions...)
	ixns = append(ixns, currentIx)
	ixns = append(ixns, out.PostInstructions...)

	return ixns, nil
}

// ClaimPositionFee builds instructions to claim position fee rewards.
func (cp *CpAMM) ClaimPositionFee(
	ctx context.Context,
	param types.ClaimPositionFeeParams,
) ([]solana.Instruction, error) {

	payer := param.Owner
	if !param.FeePayer.IsZero() {
		payer = param.FeePayer
	}

	out, err := cp.setupFeeClaimAccounts(
		ctx,
		types.SetupFeeClaimAccountsParams{
			Payer:           payer,
			Owner:           param.Owner,
			TokenAMint:      param.TokenAMint,
			TokenBMint:      param.TokenBMint,
			TokenAProgram:   param.TokenAProgram,
			TokenBProgram:   param.TokenBProgram,
			Receiver:        param.Receiver,
			TempWSolAccount: param.TempWSolAccount,
		},
	)
	if err != nil {
		return nil, err
	}

	claimPositionFeeIx, err := cp.buildClaimPositionFeeInstruction(
		types.ClaimPositionFeeInstructionParams{
			Owner:              param.Owner,
			PoolAuthority:      cp.poolAuthority,
			Pool:               param.Pool,
			Position:           param.Position,
			PositionNftAccount: param.PositionNftAccount,
			TokenAAccount:      out.TokenAAccount,
			TokenBAccount:      out.TokenBAccount,
			TokenAVault:        param.TokenAVault,
			TokenBVault:        param.TokenBVault,
			TokenAMint:         param.TokenAMint,
			TokenBMint:         param.TokenBMint,
			TokenAProgram:      param.TokenAProgram,
			TokenBProgram:      param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, len(out.PreInstructions)+1+len(out.PostInstructions))
	ixns = append(ixns, out.PreInstructions...)
	ixns = append(ixns, claimPositionFeeIx)
	ixns = append(ixns, out.PostInstructions...)

	return ixns, nil

}

// ClaimPositionFee2 builds instructions to claim position fee rewards.
func (cp *CpAMM) ClaimPositionFee2(
	ctx context.Context,
	param types.ClaimPositionFeeParams2,
) ([]solana.Instruction, error) {

	payer := param.Owner
	if !param.FeePayer.IsZero() {
		payer = param.FeePayer
	}

	tokenAOwner, tokenBOwner := param.Receiver, param.Receiver

	if param.TokenAMint.Equals(solana.WrappedSol) {
		tokenAOwner = param.Owner
	}

	if param.TokenBMint.Equals(solana.WrappedSol) {
		tokenBOwner = param.Owner
	}

	preparedTokenAccs, err := cp.prepareTokenAccounts(
		ctx,
		types.PrepareTokenAccountParams{
			Payer:         payer,
			TokenAOwner:   tokenAOwner,
			TokenBOwner:   tokenBOwner,
			TokenAMint:    param.TokenAMint,
			TokenBMint:    param.TokenBMint,
			TokenAProgram: param.TokenAProgram,
			TokenBProgram: param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	postInstructions := make([]solana.Instruction, 0, 1)
	if param.TokenAMint.Equals(solana.WrappedSol) ||
		param.TokenBMint.Equals(solana.WrappedSol) {
		// unwarp sol to receiver
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.Owner, param.Receiver, false,
		)
		if err != nil {
			return nil, err
		}
		postInstructions = append(postInstructions, closeWrappedSOLIx)
	}

	claimPositionFeeIx, err := cp.buildClaimPositionFeeInstruction(
		types.ClaimPositionFeeInstructionParams{
			Owner:              param.Owner,
			PoolAuthority:      cp.poolAuthority,
			Pool:               param.Pool,
			Position:           param.Position,
			PositionNftAccount: param.PositionNftAccount,
			TokenAAccount:      preparedTokenAccs.TokenAAta,
			TokenBAccount:      preparedTokenAccs.TokenBAta,
			TokenAVault:        param.TokenAVault,
			TokenBVault:        param.TokenBVault,
			TokenAMint:         param.TokenAMint,
			TokenBMint:         param.TokenBMint,
			TokenAProgram:      param.TokenAProgram,
			TokenBProgram:      param.TokenBProgram,
		},
	)
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, 2+1+len(postInstructions))
	ixns = append(ixns, preparedTokenAccs.CreateATAIxns...)
	ixns = append(ixns, claimPositionFeeIx)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}

// ClaimReward builds instruction to claim reward from a position.
func (cp *CpAMM) ClaimReward(
	ctx context.Context,
	param types.ClaimRewardParams,
) ([]solana.Instruction, error) {

	rewardInfo := param.PoolState.RewardInfos[param.RewardIndex]
	tokenProgram := helpers.GetTokenProgram(rewardInfo.RewardTokenFlag)

	postInstructions := make([]solana.Instruction, 0, 2)

	feePayer := param.User
	if !param.FeePayer.IsZero() {
		feePayer = param.FeePayer
	}

	userTokenAccount, createUserTokenAccountIx, err := helpers.GetOrCreateATAInstruction(
		ctx,
		cp.conn,
		rewardInfo.Mint,
		param.User,
		feePayer,
		true,
		tokenProgram,
	)
	if err != nil {
		return nil, err
	}
	postInstructions = append(postInstructions, createUserTokenAccountIx)

	if rewardInfo.Mint.Equals(solana.WrappedSol) {
		closeWrappedSOLIx, err := helpers.UnwrapSOLInstruction(
			param.User, solana.PublicKey{}, false,
		)
		if err != nil {
			return nil, err
		}
		postInstructions = append(postInstructions, closeWrappedSOLIx)
	}

	claimRewardPtr := cp_amm.NewClaimRewardInstruction(
		param.RewardIndex,
		cp.poolAuthority,
		param.PositionState.Pool,
		param.Position,
		rewardInfo.Vault,
		rewardInfo.Mint,
		userTokenAccount,
		param.PositionNftAccount,
		param.User,
		tokenProgram,
		solana.PublicKey{},
		CpAMMProgramId,
	)

	currentIx, err := claimRewardPtr.ValidateAndBuild()
	if err != nil {
		return nil, err
	}

	ixns := make([]solana.Instruction, 0, 1+len(postInstructions))
	ixns = append(ixns, currentIx)
	ixns = append(ixns, postInstructions...)

	return ixns, nil
}
