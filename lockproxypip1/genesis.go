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

package lockproxypip1

import (
	"bytes"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/polynetwork/cosmos-poly-module/lockproxypip1/internal/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// InitGenesis init genesis for lockproxypip1 module
func InitGenesis(ctx sdk.Context, keeper Keeper, data GenesisState) []abci.ValidatorUpdate {
	// check if the module account exists
	moduleAcc := keeper.GetModuleAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("initGenesis error: %s module account has not been set", types.ModuleName))
	}

	store := keeper.Store(ctx)

	keeper.SetParams(ctx, data.Params)

	keeper.SetNonce(ctx, data.Nonce)

	for k, v := range data.Operators {
		operator, err := sdk.AccAddressFromBech32(k)
		if err != nil {
			panic(err)
		}

		if bytes.Compare(operator.Bytes(), v) != 0 {
			panic("Invalid operator bytes in init genesis!")
		}

		store.Set(GetOperatorToLockProxyKey(operator), v)
	}

	// set chain ids directly
	for k, v := range data.ChainIDs {
		key, err := hex.DecodeString(k)
		if err != nil {
			panic(err)
		}
		store.Set(key, v)
	}

	// set registries directly
	for k, v := range data.Registries {
		key, err := hex.DecodeString(k)
		if err != nil {
			panic(err)
		}
		store.Set(key, v)
	}

	// set balances directly
	for k, v := range data.Balances {
		key, err := hex.DecodeString(k)
		if err != nil {
			panic(err)
		}
		keeper.StoreBalance(ctx, key, v)
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) GenesisState {
	params := keeper.GetParams(ctx)

	nonce := keeper.GetNonce(ctx)

	// iterate Operators
	operators := make(map[string][]byte)
	iter := keeper.StoreIterator(ctx, OperatorToLockProxyKey)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, v := iter.Key(), iter.Value()
		addr := sdk.AccAddress(k[1:])
		operators[addr.String()] = v
	}

	// iterate ChainIDs
	chainIDs := make(map[string][]byte)
	iter1 := keeper.StoreIterator(ctx, BindChainIdPrefix)
	defer iter1.Close()
	for ; iter1.Valid(); iter1.Next() {
		k, v := iter1.Key(), iter1.Value()
		chainIDs[fmt.Sprintf("%x", k)] = v
	}

	// iterate Registries
	registries := make(map[string][]byte)
	iter2 := keeper.StoreIterator(ctx, RegistryPrefix)
	defer iter2.Close()
	for ; iter2.Valid(); iter2.Next() {
		k, v := iter2.Key(), iter2.Value()
		registries[fmt.Sprintf("%x", k)] = v
	}

	// iterate Balances
	balances := make(map[string]sdk.Int)
	iter3 := keeper.StoreIterator(ctx, BalancePrefix)
	defer iter3.Close()
	for ; iter3.Valid(); iter3.Next() {
		k, _ := iter3.Key(), iter3.Value()
		amt := keeper.GetBalance(ctx, k)
		balances[fmt.Sprintf("%x", k)] = amt
	}

	return NewGenesisState(params, nonce, chainIDs, registries, operators, balances)
}
