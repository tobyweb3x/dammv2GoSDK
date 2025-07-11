package dammv2gosdk_test

import (
	"context"
	dammv2gosdk "dammv2GoSDK"
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"dammv2GoSDK/helpers"
	"dammv2GoSDK/types"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/stretchr/testify/assert"

	testUtils "dammv2GoSDK/internal/test/utils"
)

const (
	surfPoolRPCClient = "http://127.0.0.1:8899"
	surfPoolWSlient   = "ws://127.0.0.1:8900"
)

func TestAddLiquidity(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	rootKeypair := solana.NewWallet().PrivateKey
	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	t.Logf("actors: \n%+v\n\n", actors)

	tokenAAmount := new(big.Int).Mul(
		big.NewInt(1_000),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(testUtils.Decimals), new(big.Int)),
	)
	tokenBAmount := new(big.Int).Mul(
		big.NewInt(1_000),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(testUtils.Decimals), new(big.Int)),
	)

	if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
		t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
	}

	poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
		types.PreparePoolCreationParams{
			TokenAAmount: tokenAAmount,
			TokenBAmount: tokenBAmount,
			MinSqrtPrice: testUtils.MinSqrtPrice,
			MaxSqrtPrice: testUtils.MaxSqrtPrice,
		},
	)
	if err != nil {
		t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
	}

	positionNFT := solana.NewWallet().PrivateKey
	customizeablePoolParams := types.InitializeCustomizeablePoolParams{
		Payer:          actors.Payer.PublicKey(),
		Creator:        actors.PoolCreator.PublicKey(),
		PositionNFT:    positionNFT.PublicKey(),
		TokenAMint:     actors.TokenAMint.PublicKey(),
		TokenBMint:     actors.TokenBMint.PublicKey(),
		TokenAAmount:   tokenAAmount.Uint64(),
		TokenBAmount:   tokenBAmount.Uint64(),
		SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
		SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
		LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
		InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
		PoolFees: cp_amm.PoolFeeParameters{
			BaseFee: cp_amm.BaseFeeParameters{
				CliffFeeNumerator: 1_000_000, //1%
				NumberOfPeriod:    10,
				PeriodFrequency:   10,
				ReductionFactor:   2,
				FeeSchedulerMode:  0, // linear
			},
			ProtocolFeePercent: 20,
			PartnerFeePercent:  0,
			ReferralFeePercent: 20,
			DynamicFee:         nil,
		},
		ActivationType:  1, // 0 slot, 1 timestap
		ActivationPoint: nil,
		TokenAProgram:   solana.TokenProgramID,
		TokenBProgram:   solana.TokenProgramID,
	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	txnSig, err := testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	)
	if err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.Fatal("err from SendAndConfirmTxn")
	}

	assert.NotNil(t, txnSig)
}


func TestClaimFee(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)

	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}
	t.Logf("actors: \n%+v\n\n", actors)

	{
		tokenAAmount := new(big.Int).Mul(
			big.NewInt(1_000_000),
			new(big.Int).Exp(big.NewInt(10), big.NewInt(testUtils.Decimals), new(big.Int)),
		)
		tokenBAmount := new(big.Int).Mul(
			big.NewInt(100),
			new(big.Int).Exp(big.NewInt(10), big.NewInt(testUtils.Decimals), new(big.Int)),
		)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.Payer.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     solana.WrappedSol,
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 500_000_000, // 50%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  1,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.Fatal("err from SendAndConfirmTxn")
	}

	poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
	if err != nil {
		t.Fatalf("err from GetPool: %s", err.Error())
	}

	t.Run("claim position fee to owner", func(t *testing.T) {
		// swap A -> B
		toAccount, err := helpers.GetAssociatedTokenAddressSync(
			poolState.TokenBMint,
			actors.Payer.PublicKey(),
			false,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)
		if err != nil {
			t.Fatalf("err from helpers.GetAssociatedTokenAddressSync: %s", err.Error())
		}
		createAtaIx := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
			actors.Payer.PublicKey(),
			toAccount,
			actors.Payer.PublicKey(),
			poolState.TokenBMint,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)

		swapAtoBIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenAMint,
				OutputTokenMint:      poolState.TokenBMint,
				AmountIn:             100_000_000_000,
				MinimumAmountOut:     0,
				TokenAMint:           poolState.TokenAMint,
				TokenBMint:           poolState.TokenBMint,
				TokenAVault:          poolState.TokenAVault,
				TokenBVault:          poolState.TokenBVault,
				TokenAProgram:        helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:        helpers.GetTokenProgram(poolState.TokenBFlag),
				ReferralTokenAccount: toAccount,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap: %s", err.Error())
		}

		newIxns := append([]solana.Instruction{createAtaIx}, swapAtoBIxns...)

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			newIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.Fatal("err from SendAndConfirmTx")
		}

		// claim position fee
		claimFeeIxns, err := ammInstance.ClaimPositionFee(
			context.Background(),
			types.ClaimPositionFeeParams{
				Receiver:           toAccount,
				TempWSolAccount:    solana.PublicKey{},
				Owner:              actors.Payer.PublicKey(),
				Pool:               createCustomPoolResult.Pool,
				Position:           createCustomPoolResult.Position,
				PositionNftAccount: dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
				TokenAMint:         poolState.TokenAMint,
				TokenBMint:         poolState.TokenBMint,
				TokenAVault:        poolState.TokenAVault,
				TokenBVault:        poolState.TokenBVault,
				TokenAProgram:      helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:      helpers.GetTokenProgram(poolState.TokenBFlag),
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.ClaimPositionFee: %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			claimFeeIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.Fatal("err from SendAndConfirmTx")
		}
	})

	t.Run("Claim position fee to receiver", func(t *testing.T) {
		// swap A -> B
		toAccount, err := helpers.GetAssociatedTokenAddressSync(
			poolState.TokenBMint,
			actors.Payer.PublicKey(),
			false,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)
		if err != nil {
			t.Fatalf("err from helpers.GetAssociatedTokenAddressSync: %s", err.Error())
		}
		createAtaIx := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
			actors.Payer.PublicKey(),
			toAccount,
			actors.Payer.PublicKey(),
			poolState.TokenBMint,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)

		swapAtoBIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenAMint,
				OutputTokenMint:      poolState.TokenBMint,
				AmountIn:             100_000_000_000,
				MinimumAmountOut:     0,
				TokenAMint:           poolState.TokenAMint,
				TokenBMint:           poolState.TokenBMint,
				TokenAVault:          poolState.TokenAVault,
				TokenBVault:          poolState.TokenBVault,
				TokenAProgram:        helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:        helpers.GetTokenProgram(poolState.TokenBFlag),
				ReferralTokenAccount: toAccount,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap: %s", err.Error())
		}

		newIxns := append([]solana.Instruction{createAtaIx}, swapAtoBIxns...)

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			newIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.Fatal("err from SendAndConfirmTx")
		}

		tempWSolAccountKP := solana.NewWallet().PrivateKey
		recipientKP := solana.NewWallet().PrivateKey

		ixns := make([]solana.Instruction, 0, 2)

		for _, v := range []solana.PublicKey{tempWSolAccountKP.PublicKey(), recipientKP.PublicKey()} {
			ix := system.NewTransferInstruction(
				1*solana.LAMPORTS_PER_SOL,
				rootKeypair.PublicKey(),
				v,
			).Build()

			ixns = append(ixns, ix)
		}

		// claim position fee
		claimFeeIxns, err := ammInstance.ClaimPositionFee(
			context.Background(),
			types.ClaimPositionFeeParams{
				Receiver:           recipientKP.PublicKey(),
				TempWSolAccount:    tempWSolAccountKP.PublicKey(),
				Owner:              actors.Payer.PublicKey(),
				Pool:               createCustomPoolResult.Pool,
				Position:           createCustomPoolResult.Position,
				PositionNftAccount: dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
				TokenAMint:         poolState.TokenAMint,
				TokenBMint:         poolState.TokenBMint,
				TokenAVault:        poolState.TokenAVault,
				TokenBVault:        poolState.TokenBVault,
				TokenAProgram:      helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:      helpers.GetTokenProgram(poolState.TokenBFlag),
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.ClaimPositionFee: %s", err.Error())
		}
		ixns = slices.Grow(ixns, len(claimFeeIxns))
		ixns = append(ixns, claimFeeIxns...)

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			ixns,
			actors.Payer,
			rootKeypair,
			tempWSolAccountKP,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.Fatal("err from SendAndConfirmTx")
		}
	})

	t.Run("claim fee 2: claim position fee to receiver", func(t *testing.T) {
		// swap A -> B
		toAccount, err := helpers.GetAssociatedTokenAddressSync(
			poolState.TokenBMint,
			actors.Payer.PublicKey(),
			false,
			helpers.GetTokenProgram(poolState.TokenBFlag),
			solana.PublicKey{},
		)
		if err != nil {
			t.Fatalf("err from helpers.GetAssociatedTokenAddressSync: %s", err.Error())
		}
		createAtaIxforMintB := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
			actors.Payer.PublicKey(),
			toAccount,
			actors.Payer.PublicKey(),
			poolState.TokenBMint,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)

		swapAtoBIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenAMint,
				OutputTokenMint:      poolState.TokenBMint,
				AmountIn:             100_000_000_000,
				MinimumAmountOut:     0,
				TokenAMint:           poolState.TokenAMint,
				TokenBMint:           poolState.TokenBMint,
				TokenAVault:          poolState.TokenAVault,
				TokenBVault:          poolState.TokenBVault,
				TokenAProgram:        helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:        helpers.GetTokenProgram(poolState.TokenBFlag),
				ReferralTokenAccount: toAccount,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap: %s", err.Error())
		}

		// swap B -> A
		if toAccount, err = helpers.GetAssociatedTokenAddressSync(
			poolState.TokenAMint,
			actors.Payer.PublicKey(),
			false,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		); err != nil {
			t.Fatalf("err from helpers.GetAssociatedTokenAddressSync: %s", err.Error())
		}

		createAtaIxforMintA := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
			actors.Payer.PublicKey(),
			toAccount,
			actors.Payer.PublicKey(),
			poolState.TokenBMint,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)

		swapBtoAIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenBMint,
				OutputTokenMint:      poolState.TokenAMint,
				AmountIn:             100_000_000_000,
				MinimumAmountOut:     0,
				TokenAMint:           poolState.TokenAMint,
				TokenBMint:           poolState.TokenBMint,
				TokenAVault:          poolState.TokenAVault,
				TokenBVault:          poolState.TokenBVault,
				TokenAProgram:        helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:        helpers.GetTokenProgram(poolState.TokenBFlag),
				ReferralTokenAccount: toAccount,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap: %s", err.Error())
		}

		newIxns := []solana.Instruction{createAtaIxforMintB, createAtaIxforMintA}
		newIxns = slices.Grow(newIxns, len(swapAtoBIxns)+len(swapBtoAIxns))
		newIxns = append(newIxns, swapAtoBIxns...)
		newIxns = append(newIxns, swapBtoAIxns...)

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			newIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.Fatal("err from SendAndConfirmTx")
		}

		// claim position fee
		claimFeeIxns, err := ammInstance.ClaimPositionFee(
			context.Background(),
			types.ClaimPositionFeeParams{
				Receiver: solana.NewWallet().PublicKey(),
				// TempWSolAccount:    solana.PublicKey{},
				Owner:              actors.Payer.PublicKey(),
				Pool:               createCustomPoolResult.Pool,
				Position:           createCustomPoolResult.Position,
				PositionNftAccount: dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
				TokenAMint:         poolState.TokenAMint,
				TokenBMint:         poolState.TokenBMint,
				TokenAVault:        poolState.TokenAVault,
				TokenBVault:        poolState.TokenBVault,
				TokenAProgram:      helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:      helpers.GetTokenProgram(poolState.TokenBFlag),
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.ClaimPositionFee: %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			claimFeeIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.Fatal("err from SendAndConfirmTx")
		}
	})

}

func TestClosePosition(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)

	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  1,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("CreateCustomPool was successful ✅")

	poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
	if err != nil {
		t.Fatalf("err from GetPool: %s", err.Error())
	}

	// add liquidity
	liquidityDelta := dammv2gosdk.GetDepositQuote(types.GetDepositQuoteParams{
		InAmount:     new(big.Int).SetUint64(1_000 * 1_000_000),
		IsTokenA:     true,
		SqrtPrice:    poolState.SqrtPrice.BigInt(),
		MinSqrtPrice: poolState.SqrtMinPrice.BigInt(),
		MaxSqrtPrice: poolState.SqrtMaxPrice.BigInt(),
	})

	addLiquidityParams := types.AddLiquidityParams{
		Owner:                 actors.PoolCreator.PublicKey(),
		Pool:                  createCustomPoolResult.Pool,
		Position:              createCustomPoolResult.Position,
		PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
		LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
		MaxAmountTokenA:       1_000 * 1_000_000,
		MaxAmountTokenB:       1_000 * 1_000_000,
		TokenAAmountThreshold: math.MaxUint,
		TokenBAmountThreshold: math.MaxUint,
		TokenAMint:            poolState.TokenAMint,
		TokenBMint:            poolState.TokenBMint,
		TokenAVault:           poolState.TokenAVault,
		TokenBVault:           poolState.TokenBVault,
		TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
		TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
	}

	addLiquidityIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParams)
	if err != nil {
		t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		addLiquidityIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("AddLiquidity was successful ✅")

	// remove liquidiy
	removeLiquidityParams := types.RemoveAllLiquidityParams{
		AddLiquidityParams: addLiquidityParams,
		Vestings:           nil,
		CurrentPoint:       0,
	}

	// remove all liquidity
	removeLiquidityParams.TokenAAmountThreshold = 0
	removeLiquidityParams.TokenBAmountThreshold = 0

	removeAllLiquidityIxns, err := ammInstance.RemoveALLLiquidity(context.Background(), removeLiquidityParams)
	if err != nil {
		t.Fatalf("err from ammInstance.RemoveALLLiquidity: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		removeAllLiquidityIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("RemoveALLLiquidity was successful ✅")

	// close position
	closePositionIx, err := ammInstance.ClosePosition(types.ClosePositionParams{
		Owner:              actors.PoolCreator.PublicKey(),
		Pool:               createCustomPoolResult.Pool,
		Position:           createCustomPoolResult.Position,
		PoolAuthority:      dammv2gosdk.DerivePoolAuthority(),
		PositionNftMint:    positionNFT.PublicKey(),
		PositionNftAccount: dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
	})
	if err != nil {
		t.Fatalf("err from ammInstance.ClosePosition: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		[]solana.Instruction{closePositionIx},
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}
}


func TestCreateCustomizablePool(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)

	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  0,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	computebudgetIx := computebudget.NewSetComputeUnitPriceInstruction(400_000).Build()

	newIxns := slices.AppendSeq([]solana.Instruction{computebudgetIx}, slices.Values(createCustomPoolResult.Ixns))
	txnSig, err := testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		newIxns,
		actors.Payer,
		positionNFT,
	)
	if err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	assert.NotNil(t, txnSig)
}


func TestCreateCustomizablePoolWithConfig(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	config, err := testUtils.CreateDynamicConfig(
		conn, wsClient,
		actors.Payer, 1,
		actors.PoolCreator.PublicKey(),
	)
	if err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.Fatalf("err from testUtils.CreateDynamicConfig:")
	}

	t.Log("CreateDynamicConfig was successful ✅")

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  0,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolWithConfigResult, err := ammInstance.CreateCustomPoolWithDynamicConfig(
		context.Background(),
		types.InitializeCustomizeablePoolWithDynamicConfigParams{
			InitializeCustomizeablePoolParams: customizeablePoolParams,
			Config:                            config,
			PoolCreatorAuthority:              actors.PoolCreator.PublicKey(),
		},
	)
	if err != nil {
		t.Fatalf("\nerr from CreateCustomPoolWithDynamicConfig:\n %s", err.Error())
	}

	computebudgetIx := computebudget.NewSetComputeUnitPriceInstruction(400_000).Build()

	newIxns := slices.AppendSeq([]solana.Instruction{computebudgetIx}, slices.Values(createCustomPoolWithConfigResult.Ixns))
	txnSig, err := testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		newIxns,
		actors.Payer,
		positionNFT,
	)
	if err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	assert.NotNil(t, txnSig)
}

func TestCreatePosition(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  0,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("CreateCustomPool was successful ✅")

	userPositionNft := solana.NewWallet().PrivateKey
	createPositionParams := types.CreatePositionParams{
		Owner:       actors.User.PublicKey(),
		Payer:       actors.User.PublicKey(),
		Pool:        createCustomPoolResult.Pool,
		PositionNft: userPositionNft.PublicKey(),
	}

	createPositionIx, err := ammInstance.CreatePosition(createPositionParams)
	if err != nil {
		t.Fatalf("\nerr from CreatePosition:\n %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		[]solana.Instruction{createPositionIx.Ix},
		actors.User,
		userPositionNft,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}
}

func TestFeeHelpers(t *testing.T) {
	t.Run("get base fee params with Linear Fee Scheduler", func(t *testing.T) {
		const (
			maxBaseFee = 4_000 // 40%
			minBaseFee = 100   // 1%
		)
		baseFeeParams, err := helpers.GetBaseFeeParams(
			maxBaseFee,
			minBaseFee,
			types.Linear,
			120,
			60,
		)
		if err != nil {
			t.Fatalf("\nerr from GetBaseFeeParams:\n %s", err.Error())
		}
		cliffFeeNumerator := helpers.BpsToFeeNumerator(maxBaseFee)
		baseFeeNumerator := helpers.GetBaseFeeNumerator(
			types.Linear,
			cliffFeeNumerator,
			big.NewInt(120),
			baseFeeParams.ReductionFactor,
		)
		minBaseFeeNumerator := helpers.BpsToFeeNumerator(minBaseFee)

		assert.Equal(t, minBaseFeeNumerator, baseFeeNumerator)
	})
	t.Run("get base fee params with Exponential Fee Scheduler", func(t *testing.T) {
		const (
			maxBaseFee = 4_000 // 40%
			minBaseFee = 100   // 1%
		)
		baseFeeParams, err := helpers.GetBaseFeeParams(
			maxBaseFee,
			minBaseFee,
			types.Exponential,
			120,
			60,
		)
		if err != nil {
			t.Fatalf("\nerr from GetBaseFeeParams:\n %s", err.Error())
		}
		cliffFeeNumerator := helpers.BpsToFeeNumerator(maxBaseFee)
		baseFeeNumerator := helpers.GetBaseFeeNumerator(
			types.Exponential,
			cliffFeeNumerator,
			big.NewInt(120),
			baseFeeParams.ReductionFactor,
		)
		minBaseFeeNumerator := helpers.BpsToFeeNumerator(minBaseFee)
		diff := math.Abs(float64(minBaseFeeNumerator.Uint64()) - float64(baseFeeNumerator.Uint64()))
		percentDifference := diff / float64(minBaseFeeNumerator.Uint64()) * 100

		// less than 1%.
		assert.True(t, percentDifference < 1)
	})

	t.Run("get dynamic fee params", func(t *testing.T) {
		const baseFeeBps = 400 // 4%
		dynamicFeeParams, err := helpers.GetDynamicFeeParams(baseFeeBps, 0)
		if err != nil {
			t.Fatalf("\nerr from GetDynamicFeeParams:\n %s", err.Error())
		}

		maxDynamicFeeNumerator := helpers.GetDynamicFeeNumerator(
			new(big.Int).SetUint64(dynamicFeeParams.MaxVolatilityAccumulator),
			new(big.Int).SetUint64(uint64(dynamicFeeParams.BinStep)),
			new(big.Int).SetUint64(dynamicFeeParams.VariableFeeControl),
		).Uint64()
		expectDynamicFeeNumberator := new(big.Int).Div(
			new(big.Int).Mul(helpers.BpsToFeeNumerator(baseFeeBps), big.NewInt(20)),
			big.NewInt(100),
		).Uint64()

		diff := expectDynamicFeeNumberator - maxDynamicFeeNumerator
		percentDifference := float64(diff/expectDynamicFeeNumberator) * 100

		// less than 1%. Approximate by rounding
		assert.True(t, percentDifference < 0.1)
	})

	t.Run("get dynamic fee params with price change = 10%", func(t *testing.T) {
		const baseFeeBps = 400 // 4%
		dynamicFeeParams, err := helpers.GetDynamicFeeParams(baseFeeBps, 1_000)
		if err != nil {
			t.Fatalf("\nerr from GetDynamicFeeParams:\n %s", err.Error())
		}

		maxDynamicFeeNumerator := helpers.GetDynamicFeeNumerator(
			new(big.Int).SetUint64(dynamicFeeParams.MaxVolatilityAccumulator),
			new(big.Int).SetUint64(uint64(dynamicFeeParams.BinStep)),
			new(big.Int).SetUint64(dynamicFeeParams.VariableFeeControl),
		).Uint64()
		expectDynamicFeeNumberator := new(big.Int).Div(
			new(big.Int).Mul(helpers.BpsToFeeNumerator(baseFeeBps), big.NewInt(20)),
			big.NewInt(100),
		).Uint64()

		diff := expectDynamicFeeNumberator - maxDynamicFeeNumerator
		percentDifference := float64(diff/expectDynamicFeeNumberator) * 100

		// less than 0.1%. Approximate by rounding
		assert.True(t, percentDifference < 0.1)
	})
}

func TestLockPosition(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)

	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  0,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("CreateCustomPool was successful ✅")

	poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
	if err != nil {
		t.Fatalf("err from GetPool: %s", err.Error())
	}

	positionState, err := testUtils.GetPosition(conn, createCustomPoolResult.Position)
	if err != nil {
		t.Fatalf("err from GetPosition: %s", err.Error())
	}

	// add liquidity
	liquidityDelta := dammv2gosdk.GetDepositQuote(types.GetDepositQuoteParams{
		InAmount:     new(big.Int).SetUint64(1_000 * 1_000_000),
		IsTokenA:     true,
		SqrtPrice:    poolState.SqrtPrice.BigInt(),
		MinSqrtPrice: poolState.SqrtMinPrice.BigInt(),
		MaxSqrtPrice: poolState.SqrtMaxPrice.BigInt(),
	})

	addLiquidityParams := types.AddLiquidityParams{
		Owner:                 actors.PoolCreator.PublicKey(),
		Pool:                  createCustomPoolResult.Pool,
		Position:              createCustomPoolResult.Position,
		PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
		LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
		MaxAmountTokenA:       1_000 * 1_000_000,
		MaxAmountTokenB:       1_000 * 1_000_000,
		TokenAAmountThreshold: math.MaxUint,
		TokenBAmountThreshold: math.MaxUint,
		TokenAMint:            poolState.TokenAMint,
		TokenBMint:            poolState.TokenBMint,
		TokenAVault:           poolState.TokenAVault,
		TokenBVault:           poolState.TokenBVault,
		TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
		TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
	}

	addLiquidityIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParams)
	if err != nil {
		t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		addLiquidityIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("AddLiquidity was successful ✅")

	// lock all liquidity
	const numberOfPeriod = 10
	liquidityToLock, periodFrequency := new(big.Int).Div(
		positionState.UnlockedLiquidity.BigInt(),
		big.NewInt(2),
	), big.NewInt(1)

	cliffUnlockLiquidity := new(big.Int).Div(liquidityToLock, big.NewInt(2))
	liquidityPerPeriod := new(big.Int).Div(
		new(big.Int).Sub(liquidityToLock, cliffUnlockLiquidity),
		big.NewInt(numberOfPeriod),
	)

	loss := new(big.Int).Sub(
		liquidityToLock,
		new(big.Int).Add(
			cliffUnlockLiquidity,
			new(big.Int).Mul(liquidityPerPeriod, big.NewInt(numberOfPeriod)),
		),
	)

	cliffUnlockLiquidity = new(big.Int).Add(cliffUnlockLiquidity, loss)

	vestingAccount := solana.NewWallet().PrivateKey
	lockPositionParams := types.LockPositionParams{
		Owner:                actors.PoolCreator.PublicKey(),
		Payer:                actors.PoolCreator.PublicKey(),
		VestingAccount:       vestingAccount.PublicKey(),
		Position:             createCustomPoolResult.Position,
		PositionNftAccount:   dammv2gosdk.DerivePositionNftAccount(positionState.NftMint),
		Pool:                 createCustomPoolResult.Pool,
		LiquidityPerPeriod:   helpers.MustBigIntToUint128(liquidityPerPeriod),
		CliffPoint:           nil,
		PeriodFrequency:      periodFrequency.Uint64(),
		CliffUnlockLiquidity: helpers.MustBigIntToUint128(cliffUnlockLiquidity),
		NumberOfPeriod:       numberOfPeriod,
	}

	lockPositionIx, err := ammInstance.LockPosition(lockPositionParams)
	if err != nil {
		t.Fatalf("\nerr from LockPosition:\n %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		[]solana.Instruction{lockPositionIx},
		actors.PoolCreator,
		vestingAccount,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}
}

func TestMergePosition(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)

	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  0,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("CreateCustomPool was successful ✅")

	poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
	if err != nil {
		t.Fatalf("err from GetPool: %s", err.Error())
	}

	// add liquidity
	liquidityDelta := dammv2gosdk.GetDepositQuote(types.GetDepositQuoteParams{
		InAmount:     new(big.Int).SetUint64(1_000 * 1_000_000),
		IsTokenA:     true,
		SqrtPrice:    poolState.SqrtPrice.BigInt(),
		MinSqrtPrice: poolState.SqrtMinPrice.BigInt(),
		MaxSqrtPrice: poolState.SqrtMaxPrice.BigInt(),
	})

	addLiquidityParams := types.AddLiquidityParams{
		Owner:                 actors.PoolCreator.PublicKey(),
		Pool:                  createCustomPoolResult.Pool,
		Position:              createCustomPoolResult.Position,
		PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
		LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
		MaxAmountTokenA:       1_000 * 1_000_000,
		MaxAmountTokenB:       1_000 * 1_000_000,
		TokenAAmountThreshold: math.MaxUint,
		TokenBAmountThreshold: math.MaxUint,
		TokenAMint:            poolState.TokenAMint,
		TokenBMint:            poolState.TokenBMint,
		TokenAVault:           poolState.TokenAVault,
		TokenBVault:           poolState.TokenBVault,
		TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
		TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
	}

	addLiquidityIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParams)
	if err != nil {
		t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		addLiquidityIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("AddLiquidity was successful for position 1 ✅")

	// create position 2
	secondPositionNft := solana.NewWallet().PrivateKey
	createPosition2Result, err := ammInstance.CreatePosition(types.CreatePositionParams{
		Owner:       actors.PoolCreator.PublicKey(),
		Payer:       actors.PoolCreator.PublicKey(),
		Pool:        createCustomPoolResult.Pool,
		PositionNft: secondPositionNft.PublicKey(),
	})
	if err != nil {
		t.Fatalf("err from ammInstance.CreatePosition: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		[]solana.Instruction{createPosition2Result.Ix},
		actors.PoolCreator,
		secondPositionNft,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	// add liquidity position 2
	addLiquidityParamsPosition2 := types.AddLiquidityParams{
		Owner:                 actors.PoolCreator.PublicKey(),
		Pool:                  createCustomPoolResult.Pool,
		Position:              dammv2gosdk.DerivePositionAddress(secondPositionNft.PublicKey()),
		PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(secondPositionNft.PublicKey()),
		LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
		MaxAmountTokenA:       1_000 * 1_000_000,
		MaxAmountTokenB:       1_000 * 1_000_000,
		TokenAAmountThreshold: math.MaxUint,
		TokenBAmountThreshold: math.MaxUint,
		TokenAMint:            poolState.TokenAMint,
		TokenBMint:            poolState.TokenBMint,
		TokenAVault:           poolState.TokenAVault,
		TokenBVault:           poolState.TokenBVault,
		TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
		TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
	}

	addLiquidityInSecondPositionIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParamsPosition2)
	if err != nil {
		t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		addLiquidityInSecondPositionIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("AddLiquidity was successful for position 2 ✅")

	// merge two position
	secondPositionState, err := testUtils.GetPosition(conn, createCustomPoolResult.Position)
	if err != nil {
		t.Fatalf("err from GetPosition: %s", err.Error())
	}

	mergeIxns, err := ammInstance.MergePosition(
		context.Background(),
		types.MergePositionParams{
			Owner:                                actors.PoolCreator.PublicKey(),
			PositionA:                            dammv2gosdk.DerivePositionAddress(secondPositionNft.PublicKey()),
			PositionB:                            createCustomPoolResult.Position,
			PoolState:                            poolState,
			PositionBNftAccount:                  dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
			PositionANftAccount:                  dammv2gosdk.DerivePositionNftAccount(secondPositionNft.PublicKey()),
			PositionBState:                       secondPositionState,
			TokenAAmountAddLiquidityThreshold:    math.MaxUint64,
			TokenBAmountAddLiquidityThreshold:    math.MaxUint64,
			TokenAAmountRemoveLiquidityThreshold: 0,
			TokenBAmountRemoveLiquidityThreshold: 0,
			PositionBVestings:                    nil,
			CurrentPoint:                         0,
		},
	)

	if err != nil {
		t.Fatalf("err from MergePosition: %s", err.Error())
	}
	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		mergeIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}
}

func TestPermanentLockPosition(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	var (
		rootKeypair             = solana.NewWallet().PrivateKey
		positionNFT             = solana.NewWallet().PrivateKey
		customizeablePoolParams types.InitializeCustomizeablePoolParams
	)

	actors, err := testUtils.SetupTestContext(
		conn,
		wsClient,
		rootKeypair,
		false,
		nil,
	)

	if err != nil {
		t.Fatalf("err from SetupTestContext: %s", err.Error())
	}

	{
		tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

		if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
			t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
		}

		poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
			types.PreparePoolCreationParams{
				TokenAAmount: tokenAAmount,
				TokenBAmount: tokenBAmount,
				MinSqrtPrice: testUtils.MinSqrtPrice,
				MaxSqrtPrice: testUtils.MaxSqrtPrice,
			},
		)
		if err != nil {
			t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
		}

		customizeablePoolParams = types.InitializeCustomizeablePoolParams{
			Payer:          actors.Payer.PublicKey(),
			Creator:        actors.PoolCreator.PublicKey(),
			PositionNFT:    positionNFT.PublicKey(),
			TokenAMint:     actors.TokenAMint.PublicKey(),
			TokenBMint:     actors.TokenBMint.PublicKey(),
			TokenAAmount:   tokenAAmount.Uint64(),
			TokenBAmount:   tokenBAmount.Uint64(),
			SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
			SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
			LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
			InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				ProtocolFeePercent: 20,
				PartnerFeePercent:  0,
				ReferralFeePercent: 20,
				DynamicFee:         nil,
			},
			ActivationType:  1, // 0 slot, 1 timestap
			CollectFeeMode:  0,
			ActivationPoint: nil,
			TokenAProgram:   solana.TokenProgramID,
			TokenBProgram:   solana.TokenProgramID,
		}

	}

	ammInstance := dammv2gosdk.NewCpAMM(conn)
	createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
	if err != nil {
		t.Fatalf("err from CreateCustomPool: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		createCustomPoolResult.Ixns,
		actors.Payer,
		positionNFT,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("CreateCustomPool was successful ✅")

	poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
	if err != nil {
		t.Fatalf("err from GetPool: %s", err.Error())
	}

	positionState, err := testUtils.GetPosition(conn, createCustomPoolResult.Position)
	if err != nil {
		t.Fatalf("err from GetPosition: %s", err.Error())
	}

	// add liquidity
	liquidityDelta := dammv2gosdk.GetDepositQuote(types.GetDepositQuoteParams{
		InAmount:     new(big.Int).SetUint64(1_000 * 1_000_000),
		IsTokenA:     true,
		SqrtPrice:    poolState.SqrtPrice.BigInt(),
		MinSqrtPrice: poolState.SqrtMinPrice.BigInt(),
		MaxSqrtPrice: poolState.SqrtMaxPrice.BigInt(),
	})

	addLiquidityParams := types.AddLiquidityParams{
		Owner:                 actors.PoolCreator.PublicKey(),
		Pool:                  createCustomPoolResult.Pool,
		Position:              createCustomPoolResult.Position,
		PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
		LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
		MaxAmountTokenA:       1_000 * 1_000_000,
		MaxAmountTokenB:       1_000 * 1_000_000,
		TokenAAmountThreshold: math.MaxUint,
		TokenBAmountThreshold: math.MaxUint,
		TokenAMint:            poolState.TokenAMint,
		TokenBMint:            poolState.TokenBMint,
		TokenAVault:           poolState.TokenAVault,
		TokenBVault:           poolState.TokenBVault,
		TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
		TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
	}

	addLiquidityIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParams)
	if err != nil {
		t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		addLiquidityIxns,
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("AddLiquidity was successful ✅")

	// permanant lock position
	lockPositionIx, err := ammInstance.PermanentLockPosition(
		types.PermanentLockParams{
			Owner:              actors.PoolCreator.PublicKey(),
			Position:           createCustomPoolResult.Position,
			PositionNftAccount: dammv2gosdk.DerivePositionNftAccount(positionState.NftMint),
			Pool:               createCustomPoolResult.Pool,
			UnlockedLiquidity:  positionState.UnlockedLiquidity,
		},
	)
	if err != nil {
		t.Fatalf("err from ammInstance.PermanentLockPosition: %s", err.Error())
	}

	if _, err = testUtils.SendAndConfirmTxn(
		conn,
		wsClient,
		[]solana.Instruction{lockPositionIx},
		actors.PoolCreator,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}
}

func TestRemoveLiquidity(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	t.Run("remove liquidity with SPL-Token", func(t *testing.T) {
		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)
		actors, err := testUtils.SetupTestContext(
			conn,
			wsClient,
			rootKeypair,
			false,
			nil,
		)

		if err != nil {
			t.Fatalf("err from SetupTestContext: %s", err.Error())
		}

		{
			tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
			tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

			if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
				t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
			}

			poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
				types.PreparePoolCreationParams{
					TokenAAmount: tokenAAmount,
					TokenBAmount: tokenBAmount,
					MinSqrtPrice: testUtils.MinSqrtPrice,
					MaxSqrtPrice: testUtils.MaxSqrtPrice,
				},
			)
			if err != nil {
				t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
			}

			customizeablePoolParams = types.InitializeCustomizeablePoolParams{
				Payer:          actors.Payer.PublicKey(),
				Creator:        actors.PoolCreator.PublicKey(),
				PositionNFT:    positionNFT.PublicKey(),
				TokenAMint:     actors.TokenAMint.PublicKey(),
				TokenBMint:     actors.TokenBMint.PublicKey(),
				TokenAAmount:   tokenAAmount.Uint64(),
				TokenBAmount:   tokenBAmount.Uint64(),
				SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
				SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
				LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
				InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					ProtocolFeePercent: 20,
					PartnerFeePercent:  0,
					ReferralFeePercent: 20,
					DynamicFee:         nil,
				},
				ActivationType:  1, // 0 slot, 1 timestap
				CollectFeeMode:  0,
				ActivationPoint: nil,
				TokenAProgram:   solana.TokenProgramID,
				TokenBProgram:   solana.TokenProgramID,
			}

		}

		ammInstance := dammv2gosdk.NewCpAMM(conn)
		createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
		if err != nil {
			t.Fatalf("err from CreateCustomPool: %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			createCustomPoolResult.Ixns,
			actors.Payer,
			positionNFT,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("CreateCustomPool was successful ✅")

		poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
		if err != nil {
			t.Fatalf("err from GetPool: %s", err.Error())
		}

		// add liquidity
		liquidityDelta := dammv2gosdk.GetDepositQuote(types.GetDepositQuoteParams{
			InAmount:     new(big.Int).SetUint64(1_000 * 1_000_000),
			IsTokenA:     true,
			SqrtPrice:    poolState.SqrtPrice.BigInt(),
			MinSqrtPrice: poolState.SqrtMinPrice.BigInt(),
			MaxSqrtPrice: poolState.SqrtMaxPrice.BigInt(),
		})

		addLiquidityParams := types.AddLiquidityParams{
			Owner:                 actors.PoolCreator.PublicKey(),
			Pool:                  createCustomPoolResult.Pool,
			Position:              createCustomPoolResult.Position,
			PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
			LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
			MaxAmountTokenA:       1_000 * 1_000_000,
			MaxAmountTokenB:       1_000 * 1_000_000,
			TokenAAmountThreshold: math.MaxUint,
			TokenBAmountThreshold: math.MaxUint,
			TokenAMint:            poolState.TokenAMint,
			TokenBMint:            poolState.TokenBMint,
			TokenAVault:           poolState.TokenAVault,
			TokenBVault:           poolState.TokenBVault,
			TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
			TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
		}

		addLiquidityIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParams)
		if err != nil {
			t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			addLiquidityIxns,
			actors.PoolCreator,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("AddLiquidity was successful ✅")

		// remove liquidiy
		removeLiquidityIxns, err := ammInstance.RemoveLiquidity(
			context.Background(),
			types.RemoveLiquidityParams{
				Owner:                 actors.PoolCreator.PublicKey(),
				Pool:                  createCustomPoolResult.Pool,
				Position:              createCustomPoolResult.Position,
				PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
				LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
				TokenAAmountThreshold: math.MaxUint,
				TokenBAmountThreshold: math.MaxUint,
				TokenAMint:            poolState.TokenAMint,
				TokenBMint:            poolState.TokenBMint,
				TokenAVault:           poolState.TokenAVault,
				TokenBVault:           poolState.TokenBVault,
				TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
				Vestings:              nil,
				CurrentPoint:          0,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.RemoveLiquidity:\n %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			removeLiquidityIxns,
			actors.PoolCreator,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("remove liquidity with Token 2022", func(t *testing.T) {
	})
}

func TestSwap(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	t.Run("swap with SPL-Token", func(t *testing.T) {
		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)
		actors, err := testUtils.SetupTestContext(
			conn,
			wsClient,
			rootKeypair,
			false,
			nil,
		)

		if err != nil {
			t.Fatalf("err from SetupTestContext: %s", err.Error())
		}

		{
			tokenAAmount := new(big.Int).SetUint64(1_000 * 1_000_000)
			tokenBAmount := new(big.Int).SetUint64(1_000 * 1_000_000)

			if !tokenAAmount.IsUint64() || !tokenBAmount.IsInt64() {
				t.Fatal("tokenAAmount || tokenBAmount cannot fit into uint64")
			}

			poolCreationParams, err := dammv2gosdk.PreparePoolCreationParams(
				types.PreparePoolCreationParams{
					TokenAAmount: tokenAAmount,
					TokenBAmount: tokenBAmount,
					MinSqrtPrice: testUtils.MinSqrtPrice,
					MaxSqrtPrice: testUtils.MaxSqrtPrice,
				},
			)
			if err != nil {
				t.Fatalf("err from PreparePoolCreationParams: %s", err.Error())
			}

			customizeablePoolParams = types.InitializeCustomizeablePoolParams{
				Payer:          actors.Payer.PublicKey(),
				Creator:        actors.PoolCreator.PublicKey(),
				PositionNFT:    positionNFT.PublicKey(),
				TokenAMint:     actors.TokenAMint.PublicKey(),
				TokenBMint:     actors.TokenBMint.PublicKey(),
				TokenAAmount:   tokenAAmount.Uint64(),
				TokenBAmount:   tokenBAmount.Uint64(),
				SqrtMinPrice:   helpers.MustBigIntToUint128(testUtils.MinSqrtPrice),
				SqrtMaxPrice:   helpers.MustBigIntToUint128(testUtils.MaxSqrtPrice),
				LiquidityDelta: helpers.MustBigIntToUint128(poolCreationParams.LiquidityDelta),
				InitSqrtPrice:  helpers.MustBigIntToUint128(poolCreationParams.InitSqrtPrice),
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					ProtocolFeePercent: 20,
					PartnerFeePercent:  0,
					ReferralFeePercent: 20,
					DynamicFee:         nil,
				},
				ActivationType:  1, // 0 slot, 1 timestap
				CollectFeeMode:  0,
				ActivationPoint: nil,
				TokenAProgram:   solana.TokenProgramID,
				TokenBProgram:   solana.TokenProgramID,
			}

		}

		ammInstance := dammv2gosdk.NewCpAMM(conn)
		createCustomPoolResult, err := ammInstance.CreateCustomPool(context.Background(), customizeablePoolParams)
		if err != nil {
			t.Fatalf("err from CreateCustomPool: %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			createCustomPoolResult.Ixns,
			actors.Payer,
			positionNFT,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("CreateCustomPool was successful ✅")

		poolState, err := testUtils.GetPool(conn, createCustomPoolResult.Pool)
		if err != nil {
			t.Fatalf("err from GetPool: %s", err.Error())
		}

		// add liquidity
		liquidityDelta := dammv2gosdk.GetDepositQuote(types.GetDepositQuoteParams{
			InAmount:     new(big.Int).SetUint64(1_000 * 1_000_000),
			IsTokenA:     true,
			SqrtPrice:    poolState.SqrtPrice.BigInt(),
			MinSqrtPrice: poolState.SqrtMinPrice.BigInt(),
			MaxSqrtPrice: poolState.SqrtMaxPrice.BigInt(),
		})

		addLiquidityParams := types.AddLiquidityParams{
			Owner:                 actors.PoolCreator.PublicKey(),
			Pool:                  createCustomPoolResult.Pool,
			Position:              createCustomPoolResult.Position,
			PositionNftAccount:    dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
			LiquidityDelta:        helpers.MustBigIntToUint128(liquidityDelta.LiquidityDelta),
			MaxAmountTokenA:       1_000 * 1_000_000,
			MaxAmountTokenB:       1_000 * 1_000_000,
			TokenAAmountThreshold: math.MaxUint,
			TokenBAmountThreshold: math.MaxUint,
			TokenAMint:            poolState.TokenAMint,
			TokenBMint:            poolState.TokenBMint,
			TokenAVault:           poolState.TokenAVault,
			TokenBVault:           poolState.TokenBVault,
			TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
			TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
		}

		addLiquidityIxns, err := ammInstance.AddLiquidity(context.Background(), addLiquidityParams)
		if err != nil {
			t.Fatalf("err from ammInstance.AddLiquidity: %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			addLiquidityIxns,
			actors.PoolCreator,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("AddLiquidity was successful ✅")

		// swap
		swapIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:            actors.Payer.PublicKey(),
				Pool:             createCustomPoolResult.Pool,
				InputTokenMint:   poolState.TokenAMint,
				OutputTokenMint:  poolState.TokenBMint,
				AmountIn:         100 * 1_000_000,
				MinimumAmountOut: 0,
				TokenAMint:       poolState.TokenAMint,
				TokenBMint:       poolState.TokenBMint,
				TokenAVault:      poolState.TokenAVault,
				TokenBVault:      poolState.TokenBVault,
				TokenAProgram:    helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:    helpers.GetTokenProgram(poolState.TokenBFlag),
				// ReferralTokenAccount: ,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap:\n %s", err.Error())
		}

		if _, err = testUtils.SendAndConfirmTxn(
			conn,
			wsClient,
			swapIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("swap with Token 2022", func(t *testing.T) {
	})
}
