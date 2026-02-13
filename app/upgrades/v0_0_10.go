package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	tokenfactorytypes "github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func v0010(deps Dependencies) definition {
	return definition{
		name: nameV0010,
		postMigrate: func(ctx sdk.Context) error {
			// tokenfactory was introduced in v0.0.10; existing chains upgrading from
			// earlier versions do not have tokenfactory params initialized.
			deps.TokenFactoryKeeper.CreateModuleAccount(ctx)

			subspace := deps.GetSubspace(tokenfactorytypes.ModuleName)
			if !subspace.Has(ctx, tokenfactorytypes.KeyDenomCreationFee) ||
				!subspace.Has(ctx, tokenfactorytypes.KeyDenomCreationGasConsume) {
				deps.TokenFactoryKeeper.SetParams(ctx, tokenfactorytypes.DefaultParams())
			}

			return nil
		},
		storeUpgrades: &storetypes.StoreUpgrades{
			Added: []string{tokenfactorytypes.StoreKey},
		},
	}
}
