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

package keeper

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/supply/exported"
	selfexported "github.com/polynetwork/cosmos-poly-module/lockproxypip1/exported"
	"github.com/polynetwork/cosmos-poly-module/lockproxypip1/internal/types"
	polycommon "github.com/polynetwork/poly/common"
)

// Keeper of the mint store
type Keeper struct {
	cdc          *codec.Codec
	storeKey     sdk.StoreKey
	authKeeper   types.AccountKeeper
	bankKeeper   types.BankKeeper
	supplyKeeper types.SupplyKeeper
	ccmKeeper    types.CrossChainManager
	paramSpace   params.Subspace
	hooks        types.LockProxyHooks
	selfexported.LockProxyKeeper
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc *codec.Codec, key sdk.StoreKey, ak types.AccountKeeper, bk types.BankKeeper,
	supplyKeeper types.SupplyKeeper, ccmKeeper types.CrossChainManager,
	paramSpace params.Subspace) Keeper {

	// ensure mint module account is set
	if addr := supplyKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("the %s module account has not been set", types.ModuleName))
	}

	return Keeper{
		cdc:          cdc,
		storeKey:     key,
		authKeeper:   ak,
		bankKeeper:   bk,
		supplyKeeper: supplyKeeper,
		ccmKeeper:    ccmKeeper,
		paramSpace:   paramSpace.WithKeyTable(types.ParamKeyTable()),
	}
}

// GetParams returns the total set of lockproxpip1 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of lockproxpip1 parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

// Store fetches the main kv store
func (k Keeper) Store(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// StoreIterator returns the iterator for the store
func (k Keeper) StoreIterator(ctx sdk.Context, prefix []byte) sdk.Iterator {
	store := ctx.KVStore(k.storeKey)
	return sdk.KVStorePrefixIterator(store, prefix)
}

// GetModuleAccount gets the module account for this module.
func (k Keeper) GetModuleAccount(ctx sdk.Context) exported.ModuleAccountI {
	return k.supplyKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// EnsureAccountExist returns an err if the give accAddress is not created yet.
func (k Keeper) EnsureAccountExist(ctx sdk.Context, addr sdk.AccAddress) error {
	acct := k.authKeeper.GetAccount(ctx, addr)
	if acct == nil {
		return types.ErrAccountNotExist(fmt.Sprintf("account %s does not exist", addr.String()))
	}
	return nil
}

func (k Keeper) ContainToContractAddr(ctx sdk.Context, toContractAddr []byte, fromChainId uint64) bool {
	return ctx.KVStore(k.storeKey).Get((GetBindChainIdKey(toContractAddr, fromChainId))) != nil
}

func (k Keeper) CreateLockProxy(ctx sdk.Context, creator sdk.AccAddress) error {
	if k.EnsureLockProxyExist(ctx, creator) {
		return types.ErrCreateLockProxy(fmt.Sprintf("creator:%s already created lockproxy contract with hash:%x", creator.String(), creator.Bytes()))
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(GetOperatorToLockProxyKey(creator), creator.Bytes())
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateLockProxy,
			sdk.NewAttribute(types.AttributeKeyCreator, creator.String()),
			sdk.NewAttribute(types.AttributeKeyProxyHash, hex.EncodeToString(creator.Bytes())),
		),
	})
	ctx.Logger().With("module", fmt.Sprintf("creator:%s initialized a lockproxy contract with hash: %x", creator.String(), creator.Bytes()))
	return nil
}

func (k Keeper) EnsureLockProxyExist(ctx sdk.Context, creator sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return bytes.Equal(store.Get(GetOperatorToLockProxyKey(creator)), creator)
}

func (k Keeper) GetLockProxyByOperator(ctx sdk.Context, operator sdk.AccAddress) []byte {
	store := ctx.KVStore(k.storeKey)
	proxyBytes := store.Get(GetOperatorToLockProxyKey(operator))
	if len(proxyBytes) == 0 || !bytes.Equal(operator.Bytes(), proxyBytes) {
		return nil
	}
	return proxyBytes
}

func (k Keeper) updateRegistry(ctx sdk.Context, lockProxyHash []byte, assetHash []byte,
	nativeChainID uint64, nativeLockProxyHash []byte, nativeAssetHash []byte) error {
	if k.AssetIsRegistered(ctx, lockProxyHash, assetHash, nativeChainID, nativeLockProxyHash, nativeAssetHash) {
		return types.ErrRegistryAlreadyExists(fmt.Sprintf("asset already registered %x, %d, %x, %x", assetHash, nativeChainID, nativeLockProxyHash, nativeAssetHash))
	}

	store := ctx.KVStore(k.storeKey)
	registryKey := GetRegistryKey(lockProxyHash, assetHash, nativeChainID, nativeLockProxyHash, nativeAssetHash)
	store.Set(registryKey, []byte("1"))

	// GetBindChainIdKey is used in ContainToContractAddr to check when to return true
	// this will allow the module to be called by the ccm keeper to handle the appropriate cross-chain txns
	bindChainIDKey := GetBindChainIdKey(lockProxyHash, nativeChainID)
	if store.Get(bindChainIDKey) == nil {
		store.Set(bindChainIDKey, []byte("1"))
	}

	return nil
}

func (k Keeper) GetBalance(ctx sdk.Context, balanceKey []byte) sdk.Int {
	store := ctx.KVStore(k.storeKey)
	currentAmount := sdk.ZeroInt()
	currentAmountBz := store.Get(balanceKey)
	if currentAmountBz != nil {
		err := k.cdc.UnmarshalBinaryLengthPrefixed(currentAmountBz, &currentAmount)
		if err != nil {
			panic(err)
		}
	}

	return currentAmount
}

func (k Keeper) StoreBalance(ctx sdk.Context, balanceKey []byte, newAmount sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	newAmountBz, err := k.cdc.MarshalBinaryLengthPrefixed(newAmount)
	if err != nil {
		panic(err)
	}
	store.Set(balanceKey, newAmountBz)
}

// IncreaseBalance increases the balance locked in this module associated to the
// native lockProxy, asset, and creator tuple, for the given asset.
//
// Deprecated: this method is deprecated after version 0 and is a no-op
func (k Keeper) IncreaseBalance(ctx sdk.Context, lockProxyHash []byte, assetHash []byte,
	nativeChainID uint64, nativeLockProxyHash []byte, nativeAssetHash []byte, amount sdk.Int) {
	if k.GetVersion(ctx) > 0 {
		return
	}
	balanceKey := GetBalanceKey(lockProxyHash, assetHash, nativeChainID, nativeLockProxyHash, nativeAssetHash)
	currentAmount := k.GetBalance(ctx, balanceKey)
	newAmount := currentAmount.Add(amount)
	k.StoreBalance(ctx, balanceKey, newAmount)
}

// DecreaseBalance decreases the balance locked in this module associated to the
// native lockProxy, asset, and creator tuple, for the given asset.
//
// Deprecated: this method is deprecated after version 0 and is a no-op
func (k Keeper) DecreaseBalance(ctx sdk.Context, lockProxyHash []byte, assetHash []byte,
	nativeChainID uint64, nativeLockProxyHash []byte, nativeAssetHash []byte, amount sdk.Int) error {
	if k.GetVersion(ctx) > 0 {
		return nil
	}
	balanceKey := GetBalanceKey(lockProxyHash, assetHash, nativeChainID, nativeLockProxyHash, nativeAssetHash)
	currentAmount := k.GetBalance(ctx, balanceKey)
	newAmount := currentAmount.Sub(amount)
	if newAmount.LT(sdk.ZeroInt()) {
		return types.ErrBalance(fmt.Sprintf("insufficient balance, current balance: %s, decrement balance: %s", currentAmount.String(), amount.String()))
	}
	k.StoreBalance(ctx, balanceKey, newAmount)
	return nil
}

// AssetIsRegistered returns whether the given assetID, chainID, denom, denom creator tuple has been registered.
func (k Keeper) AssetIsRegistered(ctx sdk.Context, lockProxyHash []byte, assetHash []byte,
	nativeChainID uint64, nativeLockProxyHash []byte, nativeAssetHash []byte) bool {
	store := ctx.KVStore(k.storeKey)
	key := GetRegistryKey(lockProxyHash, assetHash, nativeChainID, nativeLockProxyHash, nativeAssetHash)
	registryBytes := store.Get(key)
	return len(registryBytes) != 0
}

// RegisterAsset registers an asset.
//
// Deprecated: this method is deprecated and always returns an error.
func (k Keeper) RegisterAsset(ctx sdk.Context, fromChainID uint64, fromContractAddr []byte, toContractAddr []byte, argsBs []byte) error {
	return types.ErrRegisterAsset("asset registration disallowed")
}

// CreateCoinAndDelegateToProxy creates a new coin for a given creator and registers it to the given lock contract and asset on the native chain.
func (k Keeper) CreateCoinAndDelegateToProxy(ctx sdk.Context, creator sdk.AccAddress, coin sdk.Coin, lockproxyHash []byte,
	nativeChainID uint64, nativeLockProxyHash []byte, nativeAssetHash []byte) error {
	if len(k.ccmKeeper.GetDenomCreator(ctx, coin.Denom)) != 0 {
		return types.ErrCreateCoinAndDelegateToProxy(fmt.Sprintf("denom:%s already exists", coin.Denom))
	}
	if exist := k.EnsureLockProxyExist(ctx, lockproxyHash); !exist {
		return types.ErrCreateCoinAndDelegateToProxy(fmt.Sprintf("lockproxy with hash: %s not created", lockproxyHash))
	}

	k.ccmKeeper.SetDenomCreator(ctx, coin.Denom, creator)

	if err := k.updateRegistry(ctx, lockproxyHash, []byte(coin.Denom), nativeChainID, nativeLockProxyHash, nativeAssetHash); err != nil {
		return err
	}

	if k.GetVersion(ctx) == 0 {
		// only mint coins here in legacy version
		if err := k.supplyKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
			return types.ErrCreateCoinAndDelegateToProxy(fmt.Sprintf("supplyKeeper.MintCoins Error: %s", err.Error()))
		}
	} else if !coin.Amount.IsZero() {
		// version > 0 should create coins with 0 amt
		return types.ErrCreateCoinAndDelegateToProxy(fmt.Sprintf("coin amount should be zero %d", coin.Amount))
	}

	k.IncreaseBalance(ctx, lockproxyHash, []byte(coin.Denom), nativeChainID, nativeLockProxyHash, nativeAssetHash, coin.Amount)

	args := types.RegisterAssetTxArgs{
		AssetHash:       []byte(coin.Denom),
		NativeAssetHash: nativeAssetHash,
	}
	sink := polycommon.NewZeroCopySink(nil)
	if err := args.Serialization(sink); err != nil {
		return types.ErrCreateCoinAndDelegateToProxy(fmt.Sprintf("TxArgs Serialization Error:%v", err))
	}
	if err := k.ccmKeeper.CreateCrossChainTx(ctx, creator, nativeChainID, lockproxyHash, nativeLockProxyHash, "registerAsset", sink.Bytes()); err != nil {
		return types.ErrCreateCoinAndDelegateToProxy(
			fmt.Sprintf("ccmKeeper.CreateCrossChainTx Error: toChainId: %d, fromContractHash: %x, toChainProxyHash: %x, args: %x",
				nativeChainID, lockproxyHash, nativeLockProxyHash, args))
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateAndDelegateCoinToProxy,
			sdk.NewAttribute(types.AttributeKeySourceAssetDenom, coin.Denom),
			sdk.NewAttribute(types.AttributeKeyCreator, creator.String()),
			sdk.NewAttribute(types.AttributeKeyAmount, coin.Amount.String()),
		),
	})
	return nil
}

func (k Keeper) GetNonce(ctx sdk.Context) sdk.Int {
	store := ctx.KVStore(k.storeKey)

	nonce := sdk.ZeroInt()
	nonceBz := store.Get(NonceKey)
	if nonceBz != nil {
		err := k.cdc.UnmarshalBinaryLengthPrefixed(nonceBz, &nonce)
		if err != nil {
			panic(err)
		}
	}

	return nonce
}

func (k Keeper) SetNonce(ctx sdk.Context, x sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	newNonceBz, err := k.cdc.MarshalBinaryLengthPrefixed(x)
	if err != nil {
		panic(err)
	}
	store.Set(NonceKey, newNonceBz)
}

func (k Keeper) getNextNonce(ctx sdk.Context) sdk.Int {
	nonce := k.GetNonce(ctx)
	newNonce := nonce.Add(sdk.NewInt(1))
	k.SetNonce(ctx, newNonce)

	return newNonce
}

// Lock sends tokens to this module, releasing it on the toChain.
// On version > 0, the tokens are burnt to give the correct global supply.
func (k Keeper) Lock(ctx sdk.Context, lockProxyHash []byte, fromAddress sdk.AccAddress, sourceAssetDenom string,
	toChainID uint64, toChainProxyHash []byte, toChainAssetHash []byte, toAddressBs []byte,
	value sdk.Int, deductFeeInLock bool, feeAmount sdk.Int, feeAddress []byte) error {
	if exist := k.EnsureLockProxyExist(ctx, lockProxyHash); !exist {
		return types.ErrLock(fmt.Sprintf("lockproxy with hash: %s not created", lockProxyHash))
	}

	nonce := k.getNextNonce(ctx)
	args := types.TxArgs{
		FromAddress:   fromAddress,
		FromAssetHash: []byte(sourceAssetDenom),
		ToAssetHash:   toChainAssetHash,
		ToAddress:     toAddressBs,
		Amount:        value.BigInt(),
		FeeAmount:     feeAmount.BigInt(),
		FeeAddress:    feeAddress,
		Nonce:         nonce.BigInt(),
	}

	afterFeeAmount := value
	feeAddressAcc := sdk.AccAddress(args.FeeAddress)
	if deductFeeInLock && feeAmount.GT(sdk.ZeroInt()) {
		if feeAddressAcc.Empty() {
			return types.ErrLock("FeeAmount is present but FeeAddress is empty")
		}

		if feeAmount.GT(value) {
			return types.ErrLock(fmt.Sprintf("feeAmount %s is greater than value %s", feeAmount.String(), value.String()))
		}

		afterFeeAmount = value.Sub(feeAmount)
		feeCoins := sdk.NewCoins(sdk.NewCoin(sourceAssetDenom, feeAmount))
		err := k.bankKeeper.SendCoins(ctx, fromAddress, feeAddress, feeCoins)
		if err != nil {
			return types.ErrLock(fmt.Sprintf("bankKeeper.SendCoins Error: from: %s, amount: %s", fromAddress.String(), feeCoins.String()))
		}

		args.Amount = afterFeeAmount.BigInt()
		args.FeeAmount = big.NewInt(0)
	}

	// send coin of sourceAssetDenom from fromAddress to module account address
	amountCoins := sdk.NewCoins(sdk.NewCoin(sourceAssetDenom, afterFeeAmount))
	if err := k.supplyKeeper.SendCoinsFromAccountToModule(ctx, fromAddress, types.ModuleName, amountCoins); err != nil {
		return types.ErrLock(fmt.Sprintf("supplyKeeper.SendCoinsFromAccountToModule Error: from: %s, moduleAccount: %s of moduleName: %s, amount: %s", fromAddress.String(), k.supplyKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress(), types.ModuleName, amountCoins.String()))
	}

	// burn the module account coins unless legacy version
	if k.GetVersion(ctx) > 0 {
		if err := k.supplyKeeper.BurnCoins(ctx, types.ModuleName, amountCoins); err != nil {
			return types.ErrLock(fmt.Sprintf("supplyKeeper.BurnCoins Error: %s", err.Error()))
		}
	}

	sink := polycommon.NewZeroCopySink(nil)
	if err := args.Serialization(sink, 32); err != nil {
		return types.ErrLock(fmt.Sprintf("TxArgs Serialization Error:%v", err))
	}
	fromContractHash := lockProxyHash
	if err := k.ccmKeeper.CreateCrossChainTx(ctx, fromAddress, toChainID, fromContractHash, toChainProxyHash, "unlock", sink.Bytes()); err != nil {
		return types.ErrLock(fmt.Sprintf("ccmKeeper.CreateCrossChainTx Error: toChainId: %d, fromContractHash: %x, toChainProxyHash: %x, args: %x",
			toChainID, fromContractHash, toChainProxyHash, args))
	}
	if amountCoins.AmountOf(sourceAssetDenom).IsNegative() {
		return types.ErrLock(fmt.Sprintf("the coin being crossed has negative amount value, coin:%s", amountCoins.String()))
	}

	if !k.AssetIsRegistered(ctx, lockProxyHash, []byte(sourceAssetDenom), toChainID, toChainProxyHash, toChainAssetHash) {
		return types.ErrLock(fmt.Sprintf("missing asset registry: lockProxyHash: %s, denom: %s, toChainId: %d, toChainProxyHash: %s, toChainAssetHash: %s",
			string(lockProxyHash), sourceAssetDenom, toChainID, hex.EncodeToString(toChainProxyHash), hex.EncodeToString(toChainAssetHash)))
	}

	k.IncreaseBalance(ctx, lockProxyHash, []byte(sourceAssetDenom), toChainID, toChainProxyHash, toChainAssetHash, afterFeeAmount)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeLock,
			sdk.NewAttribute(types.AttributeKeyFromContractHash, hex.EncodeToString([]byte(sourceAssetDenom))),
			sdk.NewAttribute(types.AttributeKeyToChainId, strconv.FormatUint(toChainID, 10)),
			sdk.NewAttribute(types.AttributeKeyToChainProxyHash, hex.EncodeToString(toChainProxyHash)),
			sdk.NewAttribute(types.AttributeKeyToChainAssetHash, hex.EncodeToString(toChainAssetHash)),
			sdk.NewAttribute(types.AttributeKeyFromAddress, fromAddress.String()),
			sdk.NewAttribute(types.AttributeKeyToAddress, hex.EncodeToString(toAddressBs)),
			sdk.NewAttribute(types.AttributeKeyAmount, value.String()),
			sdk.NewAttribute(types.AttributeKeyLockProxy, hex.EncodeToString(fromContractHash)),
			sdk.NewAttribute(types.AttributeKeyFeeAmount, feeAmount.String()),
			sdk.NewAttribute(types.AttributeKeyFeeAddress, feeAddressAcc.String()),
			sdk.NewAttribute(types.AttributeKeyNonce, nonce.String()),
		),
	})

	return nil
}

// Unlock sends tokens from this module to the target account.
// On version > 0, the tokens are minted before releasing, to give the correct global supply.
func (k Keeper) Unlock(ctx sdk.Context, fromChainID uint64, fromContractAddr sdk.AccAddress, toContractAddr []byte, argsBs []byte) error {
	args := new(types.TxArgs)
	if err := args.Deserialization(polycommon.NewZeroCopySource(argsBs), 32); err != nil {
		return types.ErrUnLock(fmt.Sprintf("unlock, Deserialization args error:%s", err))
	}
	fromAssetHash := args.FromAssetHash
	toAssetHash := args.ToAssetHash
	toAddress := args.ToAddress
	amount := sdk.NewIntFromBigInt(args.Amount)
	feeAmount := sdk.NewIntFromBigInt(args.FeeAmount)
	nonce := sdk.NewIntFromBigInt(args.Nonce)

	if !k.AssetIsRegistered(ctx, toContractAddr, toAssetHash, fromChainID, fromContractAddr, fromAssetHash) {
		return types.ErrUnLock(fmt.Sprintf("missing asset registry: toContractAddr: %s, toAssetHash: %s, fromChainId: %d, fromContractAddr: %s, fromAssetHash: %s",
			string(toContractAddr), toAssetHash, fromChainID, hex.EncodeToString(fromContractAddr), hex.EncodeToString(fromAssetHash)))
	}

	// to asset hash should be the hex format string of source asset denom name, NOT Module account address
	toAssetDenom := string(toAssetHash)

	toAcctAddress := make(sdk.AccAddress, len(toAddress))
	copy(toAcctAddress, toAddress)

	fromAcctAddress := sdk.AccAddress(args.FromAddress)
	if fromAcctAddress.Empty() {
		return types.ErrUnLock("FromAddress is empty")
	}

	// mint coin of toAssetDenom unless legacy version
	if k.GetVersion(ctx) > 0 {
		mintCoins := sdk.NewCoins(sdk.NewCoin(toAssetDenom, amount))
		if err := k.supplyKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
			return types.ErrUnLock(fmt.Sprintf("supplyKeeper.MintCoins Error: %s", err.Error()))
		}
	}

	afterFeeAmount := amount
	feeAddressAcc := sdk.AccAddress(args.FeeAddress)
	if feeAmount.GT(sdk.ZeroInt()) {
		if feeAmount.GT(amount) {
			return types.ErrUnLock(fmt.Sprintf("feeAmount %s is greater than amount %s", feeAmount.String(), amount.String()))
		}

		if feeAddressAcc.Empty() {
			return types.ErrUnLock("FeeAmount is present but FeeAddress is empty")
		}

		afterFeeAmount = afterFeeAmount.Sub(feeAmount)
		feeCoins := sdk.NewCoins(sdk.NewCoin(toAssetDenom, feeAmount))
		if err := k.supplyKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, feeAddressAcc, feeCoins); err != nil {
			return types.ErrUnLock(fmt.Sprintf("supplyKeeper.SendCoinsFromModuleToAccount, Error: send coins:%s from Module account:%s to receiver account:%s error", feeCoins.String(), k.GetModuleAccount(ctx).GetAddress().String(), feeAddressAcc.String()))
		}
	}
	amountCoins := sdk.NewCoins(sdk.NewCoin(toAssetDenom, afterFeeAmount))
	if err := k.supplyKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAcctAddress, amountCoins); err != nil {
		return types.ErrUnLock(fmt.Sprintf("supplyKeeper.SendCoinsFromModuleToAccount, Error: send coins:%s from Module account:%s to receiver account:%s error", amountCoins.String(), k.GetModuleAccount(ctx).GetAddress().String(), toAcctAddress.String()))
	}

	err := k.DecreaseBalance(ctx, toContractAddr, toAssetHash, fromChainID, fromContractAddr, fromAssetHash, amount)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUnlock,
			sdk.NewAttribute(types.AttributeKeyToChainAssetHash, hex.EncodeToString([]byte(toAssetDenom))),
			sdk.NewAttribute(types.AttributeKeyToAddress, toAcctAddress.String()),
			sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
			sdk.NewAttribute(types.AttributeKeyFromAddress, fromAcctAddress.String()),
			sdk.NewAttribute(types.AttributeKeySourceAssetHash, hex.EncodeToString(fromAssetHash)),
			sdk.NewAttribute(types.AttributeKeyFeeAmount, feeAmount.String()),
			sdk.NewAttribute(types.AttributeKeyFeeAddress, feeAddressAcc.String()),
			sdk.NewAttribute(types.AttributeKeyNonce, nonce.String()),
		),
	})

	k.AfterProxyUnlock(ctx, fromAcctAddress, toAcctAddress, amountCoins)

	return nil
}

// GetVersion gets the runtime version of the lockproxypip1
func (k Keeper) GetVersion(ctx sdk.Context) (version uint64) {
	k.paramSpace.GetIfExists(ctx, types.Version, &version)
	return
}
