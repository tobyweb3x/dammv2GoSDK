package dammv2gosdk

import "github.com/gagliardetto/solana-go"

func DerivePoolAuthority() solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("pool_authority"),
		},
		CpAMMProgramId,
	)
	return pda
}

func DeriveTokenBadgeAddress(tokenMint solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("pool_authority"),
			tokenMint.Bytes(),
		},
		CpAMMProgramId,
	)
	return pda
}

func DerivePositionAddress(positionNft solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("position"),
			positionNft.Bytes(),
		},
		CpAMMProgramId,
	)
	return pda
}

func DerivePositionNftAccount(positionNftMint solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("position_nft_account"),
			positionNftMint.Bytes(),
		},
		CpAMMProgramId,
	)
	return pda
}

func DeriveTokenVaultAddress(tokenMint, pool solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("token_vault"),
			tokenMint.Bytes(),
			pool.Bytes(),
		},
		CpAMMProgramId,
	)
	return pda
}
