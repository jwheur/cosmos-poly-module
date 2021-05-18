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

package exported

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/polynetwork/cosmos-poly-module/lockproxypip1/internal/types"
)

// UnlockKeeper is the exported interface for keepers that can unlock tokens
type UnlockKeeper interface {
	Unlock(ctx sdk.Context, fromChainID uint64, fromContractAddr sdk.AccAddress, toContractAddr []byte, argsBs []byte) error
	ContainToContractAddr(ctx sdk.Context, toContractAddr []byte, fromChainID uint64) bool
}

// LockProxyKeeper is the exported interface for the LockProxyKeeper
type LockProxyKeeper interface {
	UnlockKeeper

	SetParams(ctx sdk.Context, params types.Params)
	GetParams(ctx sdk.Context) (params types.Params)

	GetVersion(ctx sdk.Context) (version uint64)

	EnsureLockProxyExist(ctx sdk.Context, creator sdk.AccAddress) bool
	CreateLockProxy(ctx sdk.Context, creator sdk.AccAddress) error
	CreateCoinAndDelegateToProxy(ctx sdk.Context, creator sdk.AccAddress, coin sdk.Coin, lockproxyHash []byte,
		nativeChainID uint64, nativeLockProxyHash []byte, nativeAssetHash []byte) error
	Lock(ctx sdk.Context, lockProxyHash []byte, fromAddress sdk.AccAddress, sourceAssetDenom string,
		toChainID uint64, toChainProxyHash []byte, toChainAssetHash []byte, toAddressBs []byte,
		value sdk.Int, deductFeeInLock bool, feeAmount sdk.Int, feeAddress []byte) error
	SyncRegisteredAsset(ctx sdk.Context, syncer sdk.AccAddress, nativeChainID uint64, denom string, nativeAssetHash, lockProxyHash, nativeLockProxyHash []byte) error
}
