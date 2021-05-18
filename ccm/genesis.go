/*
 * Copyright (C) 2020 The poly network Authors
 * This file is part of The poly network library.
 *
 * The  poly network  is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The  poly network  is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 * You should have received a copy of the GNU Lesser General Public License
 * along with The poly network .  If not, see <http://www.gnu.org/licenses/>.
 */

package ccm

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// InitGenesis new ccm genesis
func InitGenesis(ctx sdk.Context, keeper Keeper, data GenesisState) []abci.ValidatorUpdate {
	keeper.SetParams(ctx, data.Params)

	keeper.SetCrossChainId(ctx, data.CreatedTxCount)

	store := keeper.Store(ctx)

	// set details
	for k, v := range data.CreatedTxDetails {
		txParamHash, err := hex.DecodeString(k)
		if err != nil {
			panic(err)
		}
		store.Set(GetCrossChainTxKey(txParamHash), v)
	}

	// set tx ids
	for k, v := range data.ReceivedTxIDs {
		key, err := hex.DecodeString(k)
		if err != nil {
			panic(err)
		}

		fromChainId := binary.LittleEndian.Uint64(key[0:8])
		toChainId := key[8:]

		if bytes.Compare(toChainId, v) != 0 {
			panic("Invalid toChainId in init genesis!")
		}

		keeper.PutDoneTx(ctx, fromChainId, toChainId)
	}

	// set denom creators
	for k, v := range data.DenomCreators {
		addr, err := sdk.AccAddressFromBech32(v)
		if err != nil {
			panic(err)
		}
		keeper.SetDenomCreator(ctx, k, addr)
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) GenesisState {
	params := keeper.GetParams(ctx)

	txCount, err := keeper.GetCrossChainId(ctx)
	if err != nil {
		panic(err)
	}

	// iterate CreatedTxDetails
	details := make(map[string][]byte, txCount.Int64())
	iter := keeper.StoreIterator(ctx, CrossChainTxDetailPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, v := iter.Key(), iter.Value()
		details[fmt.Sprintf("%x", k[1:])] = v
	}

	// iterate ReceivedTxIDs
	txIDs := make(map[string][]byte)
	iter1 := keeper.StoreIterator(ctx, CrossChainDoneTxPrefix)
	defer iter1.Close()
	for ; iter1.Valid(); iter1.Next() {
		k, v := iter1.Key(), iter1.Value()
		txIDs[fmt.Sprintf("%x", k[1:])] = v
	}

	// iterate DenomCreators
	denomCreators := make(map[string]string)
	iter2 := keeper.StoreIterator(ctx, DenomToCreatorPrefix)
	defer iter2.Close()
	for ; iter2.Valid(); iter2.Next() {
		k, v := iter2.Key(), iter2.Value()
		// extract denom
		denom := string(k[1:])
		// convert to accAddress
		addr := sdk.AccAddress(v)
		denomCreators[denom] = addr.String()
	}

	return NewGenesisState(params, txCount, details, txIDs, denomCreators)
}
