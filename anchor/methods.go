package anchor

import (
	cp_amm "dammv2GoSDK/generated/cpAmm"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type PgMethodI interface {
	PgAccountI
	Build() *cp_amm.Instruction
	ValidateAndBuild() (*cp_amm.Instruction, error)
}

type PgMethods[T PgMethodI] struct {
	programID            solana.PublicKey
	accountDiscriminator [8]byte
	conn                 *rpc.Client
	account              func() T
}

