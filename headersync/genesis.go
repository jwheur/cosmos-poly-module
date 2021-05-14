package headersync

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/polynetwork/cosmos-poly-module/headersync/internal/types"
	polycommon "github.com/polynetwork/poly/common"
)

// InitGenesis new ccm genesis
func InitGenesis(ctx sdk.Context, keeper Keeper, data GenesisState) {
	for _, v := range data.ConsensusPeers {
		keeper.SetConsensusPeers(ctx, v)
	}

	for k, v := range data.CheckpointHashes {
		hash, err := polycommon.Uint256ParseFromBytes(v)
		if err != nil {
			panic(err)
		}
		keeper.SetKeyHeaderHash(ctx, k, hash)
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) GenesisState {
	// iterate ConsensusPeers
	peers := make(map[uint64]ConsensusPeers)
	iter := keeper.StoreIterator(ctx, ConsensusPeerPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, v := iter.Key(), iter.Value()
		chainId := binary.LittleEndian.Uint64(k[1:])
		p := new(types.ConsensusPeers)
		if err := p.Deserialization(polycommon.NewZeroCopySource(v)); err != nil {
			panic(err)
		}
		peers[chainId] = *p
	}

	// iterate CheckpointHashes
	hashes := make(map[uint64][]byte)
	iter1 := keeper.StoreIterator(ctx, KeyHeaderHashPrefix)
	defer iter1.Close()
	for ; iter1.Valid(); iter1.Next() {
		k, v := iter1.Key(), iter1.Value()
		chainId := binary.LittleEndian.Uint64(k[1:])
		hashes[chainId] = v
	}

	return NewGenesisState(peers, hashes)
}
