package keeper

import (
	"github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set params.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	// Existing chains may add tokenfactory via upgrade, where params keys are
	// not initialized yet. Fall back to module defaults instead of panicking.
	params = types.DefaultParams()
	k.paramSpace.GetIfExists(ctx, types.KeyDenomCreationFee, &params.DenomCreationFee)
	k.paramSpace.GetIfExists(ctx, types.KeyDenomCreationGasConsume, &params.DenomCreationGasConsume)
	return params
}

// SetParams sets the total set of params.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	if params.DenomCreationFee == nil {
		params.DenomCreationFee = sdk.NewCoins()
	}
	k.paramSpace.SetParamSet(ctx, &params)
}

// SetParam sets a specific tokenfactory module's parameter with the provided parameter.
func (k Keeper) SetParam(ctx sdk.Context, key []byte, value interface{}) {
	k.paramSpace.Set(ctx, key, value)
}
