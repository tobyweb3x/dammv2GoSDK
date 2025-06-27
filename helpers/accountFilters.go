package helpers

import (
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

var (
	PositionByPoolFilter = func(pool solana.PublicKey) rpc.RPCFilter {
		return rpc.RPCFilter{
			Memcmp: &rpc.RPCFilterMemcmp{
				Bytes:  pool.Bytes(),
				Offset: 8,
			},
		}
	}

	VestingByPositionFilter = func(position solana.PublicKey) rpc.RPCFilter {
		return rpc.RPCFilter{
			Memcmp: &rpc.RPCFilterMemcmp{
				Bytes:  position.Bytes(),
				Offset: 8,
			},
		}
	}
)
