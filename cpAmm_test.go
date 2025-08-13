package dammv2gosdk_test

import (
	"context"
	dammv2gosdk "dammv2GoSDK"
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"dammv2GoSDK/helpers"
	"dammv2GoSDK/types"
	"fmt"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/gagliardetto/solana-go"
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

	t.Run("Add liquidity with SPL-Token", func(t *testing.T) {
		rootKeypair := solana.NewWallet().PrivateKey
		actors, err := testUtils.SetupTestContext(
			t,
			conn,
			wsClient,
			rootKeypair,
			false,
			nil,
		)
		if err != nil {
			t.Fatalf("err from SetupTestContext: %s", err.Error())
		}

		tokenAAmount := big.NewInt(1_000 * 1_000_000)
		tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
			SqrtMinPrice:   testUtils.MinSqrtPrice,
			SqrtMaxPrice:   testUtils.MaxSqrtPrice,
			LiquidityDelta: poolCreationParams.LiquidityDelta,
			InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, //1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				DynamicFee: nil,
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

		txnSig, err := testUtils.ExecuteTransaction(
			conn,
			wsClient,
			createCustomPoolResult.Ixns,
			actors.Payer,
			positionNFT,
		)
		if err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		assert.NotNil(t, txnSig)
	})

	t.Run("Add liquidity with Token 2022", func(t *testing.T) {

	})
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
		t,
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
		tokenAAmount := new(big.Int).SetUint64(1_000_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(100 * 1_000_000)

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
			SqrtMinPrice:   testUtils.MinSqrtPrice,
			SqrtMaxPrice:   testUtils.MaxSqrtPrice,
			LiquidityDelta: poolCreationParams.LiquidityDelta,
			InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 500_000_000, // 50%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				DynamicFee: nil,
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

	if _, err = testUtils.ExecuteTransaction(
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
	createAtaIx := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
		actors.Payer.PublicKey(),
		toAccount,
		actors.Payer.PublicKey(),
		poolState.TokenBMint,
		helpers.GetTokenProgram(poolState.TokenBFlag),
		solana.PublicKey{},
	)

	swapAtoBIxns, err := ammInstance.Swap(
		context.Background(),
		types.SwapParams{
			Payer:                actors.Payer.PublicKey(),
			Pool:                 createCustomPoolResult.Pool,
			InputTokenMint:       poolState.TokenAMint,
			OutputTokenMint:      poolState.TokenBMint,
			AmountIn:             100_000 * 1_000_000,
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
	if _, err = testUtils.ExecuteTransaction(
		conn,
		wsClient,
		// swapAtoBIxns,
		newIxns,
		actors.Payer,
	); err != nil {
		testUtils.PrettyPrintTxnErrorLog(t, err)
		t.FailNow()
	}

	t.Log("Swap was successful ✅")

	t.Run("claim position fee to owner", func(t *testing.T) {
		// claim position fee
		claimFeeIxns, err := ammInstance.ClaimPositionFee(
			context.Background(),
			types.ClaimPositionFeeParams{
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			claimFeeIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("claim position fee to receiver", func(t *testing.T) {
		// claim position fee
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			ixns,
			actors.Payer,
			rootKeypair,
			tempWSolAccountKP,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})
}

func TestClaimFee2(t *testing.T) {
	t.Skip("TestClaimFee2 skippped")
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
		t,
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
		tokenAAmount := new(big.Int).SetUint64(1_000_000 * 1_000_000)
		tokenBAmount := new(big.Int).SetUint64(100 * 1_000_000)

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
			SqrtMinPrice:   testUtils.MinSqrtPrice,
			SqrtMaxPrice:   testUtils.MaxSqrtPrice,
			LiquidityDelta: poolCreationParams.LiquidityDelta,
			InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 500_000_000, // 50%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				DynamicFee: nil,
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

	if _, err = testUtils.ExecuteTransaction(
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

	fmt.Printf("\n%+v\n\n", poolState)

	t.Run("claim position fee to receiver", func(t *testing.T) {
		// swap A -> B
		toAccountB, err := helpers.GetAssociatedTokenAddressSync(
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
			toAccountB,
			actors.Payer.PublicKey(),
			poolState.TokenBMint,
			helpers.GetTokenProgram(poolState.TokenBFlag),
			solana.PublicKey{},
		)

		swapAtoBIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenAMint,
				OutputTokenMint:      poolState.TokenBMint,
				AmountIn:             1_000_000 * 1_000_000,
				MinimumAmountOut:     0,
				TokenAMint:           poolState.TokenAMint,
				TokenBMint:           poolState.TokenBMint,
				TokenAVault:          poolState.TokenAVault,
				TokenBVault:          poolState.TokenBVault,
				TokenAProgram:        helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:        helpers.GetTokenProgram(poolState.TokenBFlag),
				ReferralTokenAccount: toAccountB,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap: %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			append([]solana.Instruction{createAtaIxforMintB}, swapAtoBIxns...),
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("Swap A -> B was successful ✅")

		// swap B -> A
		toAccountA, err := helpers.GetAssociatedTokenAddressSync(
			poolState.TokenAMint,
			actors.Payer.PublicKey(),
			false,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)
		if err != nil {
			t.Fatalf("err from helpers.GetAssociatedTokenAddressSync: %s", err.Error())
		}

		createAtaIxforMintA := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
			actors.Payer.PublicKey(),
			toAccountA,
			actors.Payer.PublicKey(),
			poolState.TokenAMint,
			helpers.GetTokenProgram(poolState.TokenAFlag),
			solana.PublicKey{},
		)

		// fundIx := system.NewTransferInstruction(
		// 	10*1_000_000, // 10 tokens in micro-lamports (adjust if decimals are different)
		// 	actors.Payer.PublicKey(),
		// 	wsolATA,
		// )

		// // Sync WSOL balance
		// syncIx := token2022.NewSyncNativeInstruction(
		// 	wsolATA,
		// 	helpers.GetTokenProgram(poolState.TokenBFlag),
		// 	nil,
		// )

		swapBtoAIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenBMint,
				OutputTokenMint:      poolState.TokenAMint,
				AmountIn:             10 * 1_000_000,
				MinimumAmountOut:     0,
				TokenAMint:           poolState.TokenAMint,
				TokenBMint:           poolState.TokenBMint,
				TokenAVault:          poolState.TokenAVault,
				TokenBVault:          poolState.TokenBVault,
				TokenAProgram:        helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:        helpers.GetTokenProgram(poolState.TokenBFlag),
				ReferralTokenAccount: toAccountA,
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.Swap: %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			append([]solana.Instruction{createAtaIxforMintA}, swapBtoAIxns...),
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("Swap B -> A was successful ✅")

		recipientKP := solana.NewWallet().PrivateKey

		// system.NewTransferInstruction(
		// 	1*solana.LAMPORTS_PER_SOL,
		// 	rootKeypair.PublicKey(),
		// )

		// claim position fee
		claimFeeIxns, err := ammInstance.ClaimPositionFee2(
			context.Background(),
			types.ClaimPositionFeeParams2{
				Receiver:           recipientKP.PublicKey(),
				FeePayer:           actors.Payer.PublicKey(),
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			claimFeeIxns,
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
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

	t.Run("remove all liquidity and close position with SPL-Token", func(t *testing.T) {

		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)

		actors, err := testUtils.SetupTestContext(
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
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
			InAmount:     big.NewInt(1_000 * 1_000_000),
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
			MaxAmountTokenA:       10 * 1_000_000,
			MaxAmountTokenB:       10 * 1_000_000,
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

		if _, err = testUtils.ExecuteTransaction(
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
			CurrentPoint:       big.NewInt(0),
		}

		// remove all liquidity
		removeLiquidityParams.TokenAAmountThreshold = 0
		removeLiquidityParams.TokenBAmountThreshold = 0

		removeAllLiquidityIxns, err := ammInstance.RemoveALLLiquidity(context.Background(), removeLiquidityParams)
		if err != nil {
			t.Fatalf("err from ammInstance.RemoveALLLiquidity: %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			[]solana.Instruction{closePositionIx},
			actors.PoolCreator,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("remove all liquidity and close position with Token 2022", func(t *testing.T) {
	})
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

	t.Run("createCustomizablePool with SPL-Token", func(t *testing.T) {
		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)

		actors, err := testUtils.SetupTestContext(
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err := testUtils.ExecuteTransaction(
			conn,
			wsClient,
			createCustomPoolResult.Ixns,
			actors.Payer,
			positionNFT,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("createCustomizablePool with Token-22", func(t *testing.T) {
	})
}

func TestCreateCustomizablePoolWithConfig(t *testing.T) {
	t.Skip("TestCreateCustomizablePoolWithConfig skipped")
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	t.Run("initialize customizeablePoolWithConfig with spl token", func(t *testing.T) {

		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)

		actors, err := testUtils.SetupTestContext(
			t,
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
			actors.Payer,
			1,
			actors.PoolCreator.PublicKey(),
		)
		if err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		t.Log("CreateDynamicConfig was successful ✅")

		{
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		txnSig, err := testUtils.ExecuteTransaction(
			conn,
			wsClient,
			createCustomPoolWithConfigResult.Ixns,
			actors.Payer,
			positionNFT,
		)
		if err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}

		assert.NotNil(t, txnSig)
	})
	t.Run("initialize customizeablePoolWithConfig with Token-22", func(t *testing.T) {
	})
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
	t.Run("create position with SPL-Token", func(t *testing.T) {
		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)

		actors, err := testUtils.SetupTestContext(
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			[]solana.Instruction{createPositionIx.Ix},
			actors.User,
			userPositionNft,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("create position with Token-22", func(t *testing.T) {
	})
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
			types.FeeSchedulerModeLinear,
			120,
			60,
		)
		if err != nil {
			t.Fatalf("\nerr from GetBaseFeeParams:\n %s", err.Error())
		}
		cliffFeeNumerator := helpers.BpsToFeeNumerator(maxBaseFee)
		baseFeeNumerator := helpers.GetBaseFeeNumerator(
			types.FeeSchedulerModeLinear,
			cliffFeeNumerator,
			big.NewInt(120),
			new(big.Int).SetUint64(baseFeeParams.ReductionFactor),
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
			types.FeeSchedulerModeExponential,
			120,
			60,
		)
		if err != nil {
			t.Fatalf("\nerr from GetBaseFeeParams:\n %s", err.Error())
		}
		cliffFeeNumerator := helpers.BpsToFeeNumerator(maxBaseFee)
		baseFeeNumerator := helpers.GetBaseFeeNumerator(
			types.FeeSchedulerModeExponential,
			cliffFeeNumerator,
			big.NewInt(120),
			new(big.Int).SetUint64(baseFeeParams.ReductionFactor),
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
		)
		expectDynamicFeeNumberator := new(big.Int).Div(
			new(big.Int).Mul(helpers.BpsToFeeNumerator(baseFeeBps), big.NewInt(20)),
			big.NewInt(100),
		)

		diff := expectDynamicFeeNumberator.Uint64() - maxDynamicFeeNumerator.Uint64()
		percentDifference := float64(diff/expectDynamicFeeNumberator.Uint64()) * 100

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
		)
		expectDynamicFeeNumberator := new(big.Int).Div(
			new(big.Int).Mul(helpers.BpsToFeeNumerator(baseFeeBps), big.NewInt(20)),
			big.NewInt(100),
		)

		diff := expectDynamicFeeNumberator.Uint64() - maxDynamicFeeNumerator.Uint64()
		percentDifference := float64(diff/expectDynamicFeeNumberator.Uint64()) * 100

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

	t.Run("lock Position with SPL-Token", func(t *testing.T) {

		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)

		actors, err := testUtils.SetupTestContext(
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
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
			InAmount:     big.NewInt(1_000 * 1_000_000),
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

		if _, err = testUtils.ExecuteTransaction(
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			[]solana.Instruction{lockPositionIx},
			actors.PoolCreator,
			vestingAccount,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("lock position with Token 2022", func(t *testing.T) {

	})

}
func TestMergePosition(t *testing.T) {
	t.Skip("TestMergePosition skipped")
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
		t,
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
		tokenAAmount := big.NewInt(1_000 * 1_000_000)
		tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
			SqrtMinPrice:   testUtils.MinSqrtPrice,
			SqrtMaxPrice:   testUtils.MaxSqrtPrice,
			LiquidityDelta: poolCreationParams.LiquidityDelta,
			InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
			PoolFees: cp_amm.PoolFeeParameters{
				BaseFee: cp_amm.BaseFeeParameters{
					CliffFeeNumerator: 1_000_000, // 1%
					NumberOfPeriod:    10,
					PeriodFrequency:   10,
					ReductionFactor:   2,
					FeeSchedulerMode:  0, // linear
				},
				DynamicFee: nil,
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

	if _, err = testUtils.ExecuteTransaction(
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
		InAmount:     big.NewInt(1_000 * 1_000_000),
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

	if _, err = testUtils.ExecuteTransaction(
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

	if _, err = testUtils.ExecuteTransaction(
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
		Position:              dammv2gosdk.DerivePositionAddress(secondPositionNft.PublicKey()),
		Pool:                  createCustomPoolResult.Pool,
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

	if _, err = testUtils.ExecuteTransaction(
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
			PositionB:                            createCustomPoolResult.Position,
			PositionA:                            dammv2gosdk.DerivePositionAddress(secondPositionNft.PublicKey()),
			PoolState:                            poolState,
			PositionBNftAccount:                  dammv2gosdk.DerivePositionNftAccount(positionNFT.PublicKey()),
			PositionANftAccount:                  dammv2gosdk.DerivePositionNftAccount(secondPositionNft.PublicKey()),
			PositionBState:                       secondPositionState,
			TokenAAmountAddLiquidityThreshold:    math.MaxUint64,
			TokenBAmountAddLiquidityThreshold:    math.MaxUint64,
			TokenAAmountRemoveLiquidityThreshold: 0,
			TokenBAmountRemoveLiquidityThreshold: 0,
			PositionBVestings:                    []types.Vesting{},
			CurrentPoint:                         big.NewInt(0),
		},
	)

	if err != nil {
		t.Fatalf("err from MergePosition: %s", err.Error())
	}
	if _, err = testUtils.ExecuteTransaction(
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

	t.Run("permanant Lock Position with SPL-Token", func(t *testing.T) {
		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			positionNFT             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)

		actors, err := testUtils.SetupTestContext(
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
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
			InAmount:     big.NewInt(1_000 * 1_000_000),
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

		if _, err = testUtils.ExecuteTransaction(
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			[]solana.Instruction{lockPositionIx},
			actors.PoolCreator,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("permanant Lock Position with Token-22", func(t *testing.T) {
	})
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
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
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
			InAmount:     big.NewInt(1_000 * 1_000_000),
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

		if _, err = testUtils.ExecuteTransaction(
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
				TokenAAmountThreshold: 0,
				TokenBAmountThreshold: 0,
				TokenAMint:            poolState.TokenAMint,
				TokenBMint:            poolState.TokenBMint,
				TokenAVault:           poolState.TokenAVault,
				TokenBVault:           poolState.TokenBVault,
				TokenAProgram:         helpers.GetTokenProgram(poolState.TokenAFlag),
				TokenBProgram:         helpers.GetTokenProgram(poolState.TokenBFlag),
				Vestings:              []types.Vesting{},
				CurrentPoint:          big.NewInt(0),
			},
		)
		if err != nil {
			t.Fatalf("err from ammInstance.RemoveLiquidity:\n %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
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

func TestSplitPosition(t *testing.T) {
	conn := rpc.New(surfPoolRPCClient)
	wsClient, err := ws.Connect(context.Background(), surfPoolWSlient)
	if err != nil {
		t.Fatalf("err creating ws client: %s", err.Error())
	}

	t.Cleanup(func() {
		conn.Close()
		wsClient.Close()
	})

	t.Run("should successfully split position between poolCreator and user", func(t *testing.T) {
		var (
			rootKeypair             = solana.NewWallet().PrivateKey
			customizeablePoolParams types.InitializeCustomizeablePoolParams
		)
		actors, err := testUtils.SetupTestContext(
			t,
			conn,
			wsClient,
			rootKeypair,
			false,
			nil,
		)
		if err != nil {
			t.Fatalf("err from SetupTestContext: %s", err.Error())
		}

		// 1. Create pool with first position (owned by poolCreator)
		firstPositionNft := solana.NewWallet().PrivateKey
		{
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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

			baseFee, err := helpers.GetBaseFeeParams(
				100, // 1% max fee in bps
				100, // 1% min fee in bps
				types.FeeSchedulerModeLinear,
				0, // numberOfPeriod
				0, // totalDuration
			)
			if err != nil {
				t.Fatalf("err from GetBaseFeeParams: %s", err.Error())
			}

			customizeablePoolParams = types.InitializeCustomizeablePoolParams{
				Payer:          actors.Payer.PublicKey(),
				Creator:        actors.PoolCreator.PublicKey(),
				PositionNFT:    firstPositionNft.PublicKey(),
				TokenAMint:     actors.TokenAMint.PublicKey(),
				TokenBMint:     actors.TokenBMint.PublicKey(),
				TokenAAmount:   tokenAAmount.Uint64(),
				TokenBAmount:   tokenBAmount.Uint64(),
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee:    baseFee,
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			createCustomPoolResult.Ixns,
			actors.Payer,
			firstPositionNft,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
		t.Log("CreateCustomPool was successful ✅")

		// 2. Create second position (owned by user)
		secondPositionNft := solana.NewWallet().PrivateKey
		createPositionParams := types.CreatePositionParams{
			Owner:       actors.User.PublicKey(),
			Payer:       actors.User.PublicKey(),
			Pool:        createCustomPoolResult.Pool,
			PositionNft: secondPositionNft.PublicKey(),
		}

		createPositionIx, err := ammInstance.CreatePosition(createPositionParams)
		if err != nil {
			t.Fatalf("err from CreatePosition:\n %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			[]solana.Instruction{createPositionIx.Ix},
			actors.User,
			secondPositionNft,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
		t.Log("CreatePosition was successful ✅")

		// 3. Execute split position
		splitPositionParams := types.SplitPositionParams{
			FirstPositionOwner:  actors.PoolCreator.PublicKey(),
			SecondPositionOwner: actors.User.PublicKey(),
			Pool:                createCustomPoolResult.Pool,
			FirstPosition:       createCustomPoolResult.Position,
			FirstPositionNftAccount: dammv2gosdk.DerivePositionNftAccount(
				firstPositionNft.PublicKey(),
			),
			SecondPosition: dammv2gosdk.DerivePositionAddress(
				secondPositionNft.PublicKey(),
			),
			SecondPositionNftAccount: dammv2gosdk.DerivePositionNftAccount(
				secondPositionNft.PublicKey(),
			),
			UnlockedLiquidityPercentage: 50,
			FeeAPercentage:              50,
			FeeBPercentage:              50,
			Reward0Percentage:           50,
			Reward1Percentage:           50,
		}

		splitPositionIx, err := ammInstance.SplitPosition(splitPositionParams)
		if err != nil {
			t.Fatalf("err from CreatePosition:\n %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			[]solana.Instruction{splitPositionIx},
			actors.PoolCreator,
			actors.User,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
		t.Log("SplitPosition was successful ✅")

		afterFirstPositionState, err := testUtils.GetPosition(
			conn,
			createCustomPoolResult.Position,
		)
		if err != nil {
			t.Fatalf("err from GetPosition:\n %s", err.Error())
		}

		afterSecondPositionState, err := testUtils.GetPosition(
			conn,
			dammv2gosdk.DerivePositionAddress(
				secondPositionNft.PublicKey(),
			),
		)
		if err != nil {
			t.Fatalf("err from GetPosition:\n %s", err.Error())
		}

		assert.True(t, afterFirstPositionState.UnlockedLiquidity.BigInt().Cmp(
			afterSecondPositionState.UnlockedLiquidity.BigInt(),
		) == 0)

		assert.True(t, afterFirstPositionState.PermanentLockedLiquidity.BigInt().Cmp(
			afterSecondPositionState.PermanentLockedLiquidity.BigInt(),
		) == 0)

		assert.True(t, afterFirstPositionState.FeeAPending == afterSecondPositionState.FeeAPending)

		assert.True(t, afterFirstPositionState.FeeBPending == afterSecondPositionState.FeeBPending)
	})

	t.Run("Split position with Token 2022", func(t *testing.T) {
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
			t,
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
			tokenAAmount := big.NewInt(1_000 * 1_000_000)
			tokenBAmount := big.NewInt(1_000 * 1_000_000)

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
				SqrtMinPrice:   testUtils.MinSqrtPrice,
				SqrtMaxPrice:   testUtils.MaxSqrtPrice,
				LiquidityDelta: poolCreationParams.LiquidityDelta,
				InitSqrtPrice:  poolCreationParams.InitSqrtPrice,
				PoolFees: cp_amm.PoolFeeParameters{
					BaseFee: cp_amm.BaseFeeParameters{
						CliffFeeNumerator: 1_000_000, // 1%
						NumberOfPeriod:    10,
						PeriodFrequency:   10,
						ReductionFactor:   2,
						FeeSchedulerMode:  0, // linear
					},
					DynamicFee: nil,
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

		if _, err = testUtils.ExecuteTransaction(
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
			InAmount:     big.NewInt(1_000 * 1_000_000),
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

		if _, err = testUtils.ExecuteTransaction(
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
		createAtaIx := helpers.CreateAssociatedTokenAccountIdempotentInstruction(
			actors.Payer.PublicKey(),
			toAccount,
			actors.Payer.PublicKey(),
			poolState.TokenBMint,
			helpers.GetTokenProgram(poolState.TokenBFlag),
			solana.PublicKey{},
		)

		swapIxns, err := ammInstance.Swap(
			context.Background(),
			types.SwapParams{
				Payer:                actors.Payer.PublicKey(),
				Pool:                 createCustomPoolResult.Pool,
				InputTokenMint:       poolState.TokenAMint,
				OutputTokenMint:      poolState.TokenBMint,
				AmountIn:             100 * 1_000_000,
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
			t.Fatalf("err from ammInstance.Swap:\n %s", err.Error())
		}

		if _, err = testUtils.ExecuteTransaction(
			conn,
			wsClient,
			append([]solana.Instruction{createAtaIx}, swapIxns...),
			actors.Payer,
		); err != nil {
			testUtils.PrettyPrintTxnErrorLog(t, err)
			t.FailNow()
		}
	})

	t.Run("swap with Token 2022", func(t *testing.T) {
	})
}
