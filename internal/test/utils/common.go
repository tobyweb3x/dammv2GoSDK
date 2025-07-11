package testUtils

import (
	"context"
	dammv2gosdk "dammv2GoSDK"
	cp_amm "dammv2GoSDK/generated/cpAmm"
	"errors"
	"fmt"
	"math/big"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	ata "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

type TestActors struct {
	Admin       solana.PrivateKey
	Payer       solana.PrivateKey
	PoolCreator solana.PrivateKey
	TokenAMint  solana.PrivateKey
	TokenBMint  solana.PrivateKey
	RewardMint  solana.PrivateKey
	Funder      solana.PrivateKey
	User        solana.PrivateKey
	Operator    solana.PrivateKey
	Partner     solana.PrivateKey
}

func SetupTestContext(
	conn *rpc.Client,
	wsClient *ws.Client,
	rootKeypair solana.PrivateKey,
	token2022 bool,
	extensions []uint8,
) (*TestActors, error) {

	actors := newTestActors()

	// fund rootKeyPair
	{
		if _, err := conn.RequestAirdrop(
			context.Background(),
			rootKeypair.PublicKey(),
			100_000*solana.LAMPORTS_PER_SOL,
			rpc.CommitmentFinalized,
		); err != nil {
			return nil, fmt.Errorf("error: RequestAirdrop - %s", err.Error())
		}

		fmt.Println("got airdrop")
	}

	// fund actors
	{
		pubkeys := []solana.PublicKey{
			actors.Admin.PublicKey(),
			actors.Payer.PublicKey(),
			actors.PoolCreator.PublicKey(),
			actors.User.PublicKey(),
			actors.Funder.PublicKey(),
			actors.Operator.PublicKey(),
			actors.Partner.PublicKey(),
		}

		ixns := make([]solana.Instruction, 0, len(pubkeys))

		for _, pubkey := range pubkeys {
			ix := system.NewTransferInstruction(
				1_000*solana.LAMPORTS_PER_SOL,
				rootKeypair.PublicKey(),
				pubkey,
			).Build()
			ixns = append(ixns, ix)
		}

		if _, err := SendAndConfirmTxn(
			conn,
			wsClient,
			ixns,
			rootKeypair,
		); err != nil {
			return nil, err
		}

		fmt.Println("actors funded")
	}

	// create token
	{
		if token2022 {

		} else {
			mintAccouts := []solana.PublicKey{
				actors.TokenAMint.PublicKey(),
				actors.TokenBMint.PublicKey(),
				actors.RewardMint.PublicKey(),
			}
			ixns := make([]solana.Instruction, 0, len(mintAccouts)*2)

			lamports, err := conn.GetMinimumBalanceForRentExemption(
				context.Background(),
				token.MINT_SIZE,
				rpc.CommitmentConfirmed,
			)
			if err != nil {
				return nil, fmt.Errorf("err from GetMinimumBalanceForRentExemption: %s", err.Error())
			}

			for _, mint := range mintAccouts {
				createIx := system.NewCreateAccountInstruction(
					lamports,
					token.MINT_SIZE,
					solana.TokenProgramID,
					rootKeypair.PublicKey(),
					mint,
				).Build()
				initIx := token.NewInitializeMint2Instruction(
					Decimals,
					rootKeypair.PublicKey(),
					solana.PublicKey{},
					mint,
				).Build()

				ixns = append(ixns, createIx, initIx)
			}

			if _, err := SendAndConfirmTxn(
				conn,
				wsClient,
				ixns,
				rootKeypair,
				actors.TokenAMint,
				actors.TokenBMint,
				actors.RewardMint,
			); err != nil {
				return nil, err
			}
		}

		fmt.Println("tokens created")
	}

	// mint token
	{
		mintAccouts := []solana.PublicKey{
			actors.TokenAMint.PublicKey(),
			actors.TokenBMint.PublicKey(),
		}
		recipients := []solana.PublicKey{
			actors.Payer.PublicKey(),
			actors.User.PublicKey(),
			actors.Partner.PublicKey(),
			actors.PoolCreator.PublicKey(),
		}

		rawAmount := new(big.Int).Mul(
			big.NewInt(100_000_000),
			new(big.Int).Exp(big.NewInt(10), big.NewInt(Decimals), nil),
		)
		if !rawAmount.IsUint64() {
			return nil, errors.New("amount to be minted cannot fit into uint64")
		}

		amount := rawAmount.Uint64()

		ixns := make([]solana.Instruction, 0, len(mintAccouts)*4*2+2)
		for _, mint := range mintAccouts {
			for _, wallet := range recipients {
				ixns = append(ixns, MinTo(
					amount,
					wallet,
					mint,
					rootKeypair.PublicKey(),
					rootKeypair.PublicKey(),
				)...)
			}
		}

		ixns = append(ixns, MinTo(
			amount,
			actors.Funder.PublicKey(),
			actors.RewardMint.PublicKey(),
			rootKeypair.PublicKey(),
			rootKeypair.PublicKey(),
		)...)

		ixns = append(ixns, MinTo(
			amount,
			actors.User.PublicKey(),
			actors.RewardMint.PublicKey(),
			rootKeypair.PublicKey(),
			rootKeypair.PublicKey(),
		)...)

		if _, err := SendAndConfirmTxn(
			conn,
			wsClient,
			ixns,
			rootKeypair,
		); err != nil {
			return nil, err
		}

		fmt.Println("tokens minted")
	}

	return actors, nil
}

func newTestActors() *TestActors {
	return &TestActors{
		Admin:       solana.NewWallet().PrivateKey,
		Payer:       solana.NewWallet().PrivateKey,
		PoolCreator: solana.NewWallet().PrivateKey,
		User:        solana.NewWallet().PrivateKey,
		Funder:      solana.NewWallet().PrivateKey,
		Operator:    solana.NewWallet().PrivateKey,
		Partner:     solana.NewWallet().PrivateKey,
		TokenAMint:  solana.NewWallet().PrivateKey,
		TokenBMint:  solana.NewWallet().PrivateKey,
		RewardMint:  solana.NewWallet().PrivateKey,
	}
}

func GetPool(
	conn *rpc.Client,
	pool solana.PublicKey,
) (*cp_amm.PoolAccount, error) {
	var account cp_amm.PoolAccount
	acc, err := conn.GetAccountInfo(context.TODO(), pool)
	if err != nil {
		return nil, err
	}

	if err = account.UnmarshalWithDecoder(
		bin.NewBorshDecoder(acc.GetBinary()),
	); err != nil {
		return nil, err
	}

	return &account, nil
}

func SendAndConfirmTxn(
	conn *rpc.Client,
	wsClient *ws.Client,
	ixns []solana.Instruction,
	payer solana.PrivateKey,
	signers ...solana.PrivateKey,
) (solana.Signature, error) {

	signerMap := make(map[solana.PublicKey]*solana.PrivateKey, 1+len(signers))
	signerMap[payer.PublicKey()] = &payer

	for _, signer := range signers {
		s := signer // avoid loop variable capture
		signerMap[s.PublicKey()] = &s
	}

	blockHash, err := conn.GetLatestBlockhash(
		context.Background(),
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("error retrieving blockHash: %s", err.Error())
	}

	txn, err := solana.NewTransaction(
		ixns,
		blockHash.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("error building txn: %s", err.Error())
	}

	if _, err := txn.Sign(func(pubkey solana.PublicKey) *solana.PrivateKey {
		return signerMap[pubkey]
	}); err != nil {
		return solana.Signature{}, fmt.Errorf("unable to sign transaction: %w", err)
	}

	txnSize, _ := txn.MarshalBinary()
	if size := len(txnSize); size > 1232 {
		return solana.Signature{}, fmt.Errorf("transaction size %d exceeds the limit", size)
	}

	txnSig, err := confirm.SendAndConfirmTransaction(
		context.Background(),
		conn,
		wsClient,
		txn,
	)

	if err != nil {
		return solana.Signature{}, fmt.Errorf("error from sent txn: %s", err.Error())
	}

	return txnSig, nil
}

func MinTo(
	amount uint64,
	wallet, mint, mintAuth, payer solana.PublicKey,
) []solana.Instruction {

	createAtaIx := ata.NewCreateInstruction(
		payer,
		wallet,
		mint,
	).Build()

	ataAddr, _, _ := solana.FindAssociatedTokenAddress(
		wallet,
		mint,
	)

	mintToIx := token.NewMintToInstruction(
		amount,
		mint,
		ataAddr,
		mintAuth,
		nil,
	).Build()

	return []solana.Instruction{
		createAtaIx,
		mintToIx,
	}
}

func CreateDynamicConfig(
	conn *rpc.Client,
	wsClient *ws.Client,
	admin solana.PrivateKey,
	index uint64,
	poolCreatorAuthority solana.PublicKey,
) (solana.PublicKey, error) {
	config := dammv2gosdk.DeriveConfigAddress(index)
	createDynamicConfigPtr := cp_amm.NewCreateDynamicConfigInstruction(
		index,
		cp_amm.DynamicConfigParameters{
			PoolCreatorAuthority: poolCreatorAuthority,
		},
		config,
		admin.PublicKey(),
		solana.SystemProgramID,
		solana.PublicKey{},
		dammv2gosdk.CpAMMProgramId,
	)
	eventAuthPDA, _, err := createDynamicConfigPtr.FindEventAuthorityAddress()
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("err deriving eventAuthPDA: %w", err)
	}
	// configAdddressPDA, _, err := createDynamicConfigPtr.FindConfigAddress()
	// if err != nil {
	// 	return solana.PublicKey{}, fmt.Errorf("err deriving configAdddressPDA: %w", err)
	// }

	ix, err := createDynamicConfigPtr.SetEventAuthorityAccount(eventAuthPDA).
		ValidateAndBuild()
	if err != nil {
		return solana.PublicKey{}, err
	}

	if _, err := SendAndConfirmTxn(
		conn,
		wsClient,
		[]solana.Instruction{ix},
		admin,
	); err != nil {
		return solana.PublicKey{}, err
	}

	return config, nil
}

func GetPosition(conn *rpc.Client, position solana.PublicKey) (*cp_amm.PositionAccount, error) {
	out, err := conn.GetAccountInfo(context.Background(), position)
	if err != nil {
		return nil, err
	}
	if out == nil || out.Value == nil {
		return nil, fmt.Errorf("account not found for position: %s", position.String())
	}

	var p cp_amm.PositionAccount
	decoder := bin.NewBinDecoder(out.Value.Data.GetBinary())
	if err := p.UnmarshalWithDecoder(decoder); err != nil {
		return nil, fmt.Errorf("failed to decode position account: %w", err)
	}

	return &p, nil
}
