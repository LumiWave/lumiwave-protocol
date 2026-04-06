package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
)

// InitGenesis initializes the tokenfactory module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.CreateModuleAccount(ctx)

	if genState.Params.DenomCreationFee == nil {
		genState.Params.DenomCreationFee = sdk.NewCoins()
	}
	k.SetParams(ctx, genState.Params)

	for _, genDenom := range genState.GetFactoryDenoms() {
		creator, subdenom, err := types.DeconstructDenom(genDenom.GetDenom())
		if err != nil {
			panic(err)
		}
		// reject genesis denoms whose subdenom collides with an existing native denom
		if k.bankKeeper.HasSupply(ctx, subdenom) {
			panic(fmt.Errorf("genesis denom %s has subdenom %s that collides with an existing native denom supply", genDenom.GetDenom(), subdenom))
		}
		err = k.createDenomAfterValidation(ctx, creator, genDenom.GetDenom())
		if err != nil {
			panic(err)
		}
		err = k.setAuthorityMetadata(ctx, genDenom.GetDenom(), genDenom.GetAuthorityMetadata())
		if err != nil {
			panic(err)
		}
		// restore BeforeSendHook address if it was set
		if genDenom.GetBeforeSendHook() != "" {
			store := k.GetDenomPrefixStore(ctx, genDenom.GetDenom())
			store.Set([]byte(types.BeforeSendHookAddressPrefixKey), []byte(genDenom.GetBeforeSendHook()))
		}
	}
}

// ExportGenesis returns the tokenfactory module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genDenoms := []types.GenesisDenom{}
	iterator := k.GetAllDenomsIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Value())

		authorityMetadata, err := k.GetAuthorityMetadata(ctx, denom)
		if err != nil {
			// prevent panic on nil/corrupt AuthorityMetadata during export
			authorityMetadata = types.DenomAuthorityMetadata{}
		}

		beforeSendHook := k.GetBeforeSendHook(ctx, denom)

		genDenoms = append(genDenoms, types.GenesisDenom{
			Denom:             denom,
			AuthorityMetadata: authorityMetadata,
			BeforeSendHook:    beforeSendHook,
		})
	}

	return &types.GenesisState{
		FactoryDenoms: genDenoms,
		Params:        k.GetParams(ctx),
	}
}
