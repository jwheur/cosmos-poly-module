package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/polynetwork/cosmos-poly-module/lockproxypip1/internal/types"
)

// SetHooks set lockproxypip1 hooks
func (k *Keeper) SetHooks(hooks types.LockProxyHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set lockproxypip1 hooks twice")
	}
	k.hooks = hooks
	return k
}

// AfterProxyUnlock - call hook if registered
func (k Keeper) AfterProxyUnlock(ctx sdk.Context, from, to sdk.AccAddress, coin sdk.Coins) {
	if k.hooks != nil {
		k.hooks.AfterProxyUnlock(ctx, from, to, coin)
	}
}
