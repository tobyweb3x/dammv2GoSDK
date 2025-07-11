package dammv2gosdk

import (
	"encoding/binary"
	"slices"

	"github.com/gagliardetto/solana-go"
)

func GetFirstkey(key1, key2 solana.PublicKey) []byte {
	if slices.Compare(key1.Bytes(), key2.Bytes()) == 1 {
		return key1.Bytes()
	}
	return key2.Bytes()
}

func GetSecondkey(key1, key2 solana.PublicKey) []byte {
	if slices.Compare(key1.Bytes(), key2.Bytes()) == 1 {
		return key2.Bytes()
	}
	return key1.Bytes()
}

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

func DerivePoolAddress(
	config, tokenAMint, tokenBMint solana.PublicKey,
) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("pool"),
			config.Bytes(),
			GetFirstkey(tokenAMint, tokenBMint),
			GetSecondkey(tokenAMint, tokenBMint),
		},
		CpAMMProgramId,
	)
	return pda
}

func DeriveCustomizablePoolAddress(
	tokenAMint, tokenBMint solana.PublicKey,
) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("cpool"),
			GetFirstkey(tokenAMint, tokenBMint),
			GetSecondkey(tokenAMint, tokenBMint),
		},
		CpAMMProgramId,
	)
	return pda
}
func DeriveConfigAddress(index uint64) solana.PublicKey {
	space := make([]byte, 8)
	binary.LittleEndian.PutUint64(space, index)
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("config"),
			space,
		},
		CpAMMProgramId,
	)
	return pda
}
