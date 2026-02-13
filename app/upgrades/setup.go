package upgrades

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tokenfactorytypes "github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

type UpgradeKeeper interface {
	SetUpgradeHandler(upgradeName string, upgradeHandler upgradetypes.UpgradeHandler)
	ReadUpgradeInfoFromDisk() (upgradetypes.Plan, error)
	IsSkipHeight(height int64) bool
}

type ModuleManager interface {
	RunMigrations(ctx context.Context, configurator module.Configurator, fromVM module.VersionMap) (module.VersionMap, error)
}

type TokenFactoryKeeper interface {
	CreateModuleAccount(ctx sdk.Context)
	SetParams(ctx sdk.Context, params tokenfactorytypes.Params)
}

type ParamSubspace interface {
	Has(ctx sdk.Context, key []byte) bool
}

type Dependencies struct {
	UpgradeKeeper      UpgradeKeeper
	ModuleManager      ModuleManager
	Configurator       module.Configurator
	TokenFactoryKeeper TokenFactoryKeeper
	GetSubspace        func(moduleName string) ParamSubspace
	SetStoreLoader     func(baseapp.StoreLoader)
}

type definition struct {
	name          string
	postMigrate   func(sdk.Context) error
	storeUpgrades *storetypes.StoreUpgrades
}

func Setup(deps Dependencies) error {
	if err := deps.validate(); err != nil {
		return err
	}

	definitions := []definition{
		v1UpgradeLumiwaveProtocol(),
		v009(),
		v0010(deps),
	}

	for _, def := range definitions {
		definition := def
		deps.UpgradeKeeper.SetUpgradeHandler(
			definition.name,
			func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
				sdkCtx := sdk.UnwrapSDKContext(ctx)
				newVM, err := deps.ModuleManager.RunMigrations(ctx, deps.Configurator, fromVM)
				if err != nil {
					return nil, err
				}

				if definition.postMigrate != nil {
					if err := definition.postMigrate(sdkCtx); err != nil {
						return nil, err
					}
				}

				return newVM, nil
			},
		)
	}

	upgradeInfo, err := deps.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return err
	}

	for _, def := range definitions {
		if upgradeInfo.Name != def.name {
			continue
		}
		if deps.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
			return nil
		}

		if def.storeUpgrades == nil {
			def.storeUpgrades = &storetypes.StoreUpgrades{}
		}
		deps.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, def.storeUpgrades))
		return nil
	}

	return nil
}

func (deps Dependencies) validate() error {
	if deps.UpgradeKeeper == nil {
		return fmt.Errorf("upgrades: UpgradeKeeper is nil")
	}
	if deps.ModuleManager == nil {
		return fmt.Errorf("upgrades: ModuleManager is nil")
	}
	if deps.Configurator == nil {
		return fmt.Errorf("upgrades: Configurator is nil")
	}
	if deps.TokenFactoryKeeper == nil {
		return fmt.Errorf("upgrades: TokenFactoryKeeper is nil")
	}
	if deps.GetSubspace == nil {
		return fmt.Errorf("upgrades: GetSubspace is nil")
	}
	if deps.SetStoreLoader == nil {
		return fmt.Errorf("upgrades: SetStoreLoader is nil")
	}
	return nil
}
