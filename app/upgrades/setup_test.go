package upgrades

import (
	"context"
	"errors"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	tokenfactorytypes "github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/stretchr/testify/require"
	googlegrpc "google.golang.org/grpc"
)

func TestSetup_RegistersHandlersAndRunsMigrations(t *testing.T) {
	upgradeKeeper := &fakeUpgradeKeeper{
		readPlan: upgradetypes.Plan{Name: nameV009, Height: 9},
	}
	moduleManager := &fakeModuleManager{
		resultVM: module.VersionMap{"foo": 2},
	}
	tokenFactoryKeeper := &fakeTokenFactoryKeeper{}

	setStoreLoaderCalls := 0
	err := Setup(Dependencies{
		UpgradeKeeper:      upgradeKeeper,
		ModuleManager:      moduleManager,
		Configurator:       fakeConfigurator{},
		TokenFactoryKeeper: tokenFactoryKeeper,
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{
				existing: map[string]bool{
					string(tokenfactorytypes.KeyDenomCreationFee):        true,
					string(tokenfactorytypes.KeyDenomCreationGasConsume): true,
				},
			}
		},
		SetStoreLoader: func(baseapp.StoreLoader) {
			setStoreLoaderCalls++
		},
	})
	require.NoError(t, err)

	require.Len(t, upgradeKeeper.handlers, 3)
	require.Contains(t, upgradeKeeper.handlers, nameV1UpgradeLumiwaveProtocol)
	require.Contains(t, upgradeKeeper.handlers, nameV009)
	require.Contains(t, upgradeKeeper.handlers, nameV0010)
	require.Equal(t, 1, setStoreLoaderCalls)

	fromVM := module.VersionMap{"foo": 1}
	handler := upgradeKeeper.handlers[nameV009]
	gotVM, err := handler(sdk.WrapSDKContext(sdk.Context{}), upgradetypes.Plan{Name: nameV009}, fromVM)
	require.NoError(t, err)
	require.Equal(t, module.VersionMap{"foo": 2}, gotVM)
	require.Equal(t, 1, moduleManager.calls)
	require.Equal(t, fromVM, moduleManager.lastFromVM)
	require.Equal(t, 0, tokenFactoryKeeper.createModuleAccountCalls)
	require.Equal(t, 0, tokenFactoryKeeper.setParamsCalls)
}

func TestSetup_ReadUpgradeInfoFromDiskError(t *testing.T) {
	expectedErr := errors.New("read upgrade info failed")
	err := Setup(Dependencies{
		UpgradeKeeper: &fakeUpgradeKeeper{
			readErr: expectedErr,
		},
		ModuleManager:      &fakeModuleManager{},
		Configurator:       fakeConfigurator{},
		TokenFactoryKeeper: &fakeTokenFactoryKeeper{},
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{}
		},
		SetStoreLoader: func(baseapp.StoreLoader) {},
	})
	require.ErrorIs(t, err, expectedErr)
}

func TestSetup_DoesNotSetStoreLoaderForUnknownUpgrade(t *testing.T) {
	setStoreLoaderCalls := 0
	err := Setup(Dependencies{
		UpgradeKeeper: &fakeUpgradeKeeper{
			readPlan: upgradetypes.Plan{Name: "unknown-upgrade", Height: 9},
		},
		ModuleManager:      &fakeModuleManager{},
		Configurator:       fakeConfigurator{},
		TokenFactoryKeeper: &fakeTokenFactoryKeeper{},
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{}
		},
		SetStoreLoader: func(baseapp.StoreLoader) {
			setStoreLoaderCalls++
		},
	})
	require.NoError(t, err)
	require.Equal(t, 0, setStoreLoaderCalls)
}

func TestSetup_DoesNotSetStoreLoaderWhenSkipHeight(t *testing.T) {
	setStoreLoaderCalls := 0
	err := Setup(Dependencies{
		UpgradeKeeper: &fakeUpgradeKeeper{
			readPlan: upgradetypes.Plan{Name: nameV009, Height: 10},
			skipHeights: map[int64]bool{
				10: true,
			},
		},
		ModuleManager:      &fakeModuleManager{},
		Configurator:       fakeConfigurator{},
		TokenFactoryKeeper: &fakeTokenFactoryKeeper{},
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{}
		},
		SetStoreLoader: func(baseapp.StoreLoader) {
			setStoreLoaderCalls++
		},
	})
	require.NoError(t, err)
	require.Equal(t, 0, setStoreLoaderCalls)
}

func TestSetup_HandlerReturnsMigrationError(t *testing.T) {
	migrationErr := errors.New("migration failed")
	upgradeKeeper := &fakeUpgradeKeeper{
		readPlan: upgradetypes.Plan{Name: "unknown-upgrade", Height: 1},
	}
	moduleManager := &fakeModuleManager{
		runErr: migrationErr,
	}

	err := Setup(Dependencies{
		UpgradeKeeper:      upgradeKeeper,
		ModuleManager:      moduleManager,
		Configurator:       fakeConfigurator{},
		TokenFactoryKeeper: &fakeTokenFactoryKeeper{},
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{}
		},
		SetStoreLoader: func(baseapp.StoreLoader) {},
	})
	require.NoError(t, err)

	handler := upgradeKeeper.handlers[nameV009]
	_, err = handler(sdk.WrapSDKContext(sdk.Context{}), upgradetypes.Plan{Name: nameV009}, module.VersionMap{})
	require.ErrorIs(t, err, migrationErr)
}

func TestSetup_ValidateRequiredDependencies(t *testing.T) {
	baseDeps := Dependencies{
		UpgradeKeeper:      &fakeUpgradeKeeper{},
		ModuleManager:      &fakeModuleManager{},
		Configurator:       fakeConfigurator{},
		TokenFactoryKeeper: &fakeTokenFactoryKeeper{},
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{}
		},
		SetStoreLoader: func(baseapp.StoreLoader) {},
	}

	tests := []struct {
		name string
		deps Dependencies
		msg  string
	}{
		{
			name: "missing UpgradeKeeper",
			deps: func() Dependencies {
				d := baseDeps
				d.UpgradeKeeper = nil
				return d
			}(),
			msg: "upgrades: UpgradeKeeper is nil",
		},
		{
			name: "missing ModuleManager",
			deps: func() Dependencies {
				d := baseDeps
				d.ModuleManager = nil
				return d
			}(),
			msg: "upgrades: ModuleManager is nil",
		},
		{
			name: "missing Configurator",
			deps: func() Dependencies {
				d := baseDeps
				d.Configurator = nil
				return d
			}(),
			msg: "upgrades: Configurator is nil",
		},
		{
			name: "missing TokenFactoryKeeper",
			deps: func() Dependencies {
				d := baseDeps
				d.TokenFactoryKeeper = nil
				return d
			}(),
			msg: "upgrades: TokenFactoryKeeper is nil",
		},
		{
			name: "missing GetSubspace",
			deps: func() Dependencies {
				d := baseDeps
				d.GetSubspace = nil
				return d
			}(),
			msg: "upgrades: GetSubspace is nil",
		},
		{
			name: "missing SetStoreLoader",
			deps: func() Dependencies {
				d := baseDeps
				d.SetStoreLoader = nil
				return d
			}(),
			msg: "upgrades: SetStoreLoader is nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Setup(tc.deps)
			require.EqualError(t, err, tc.msg)
		})
	}
}

func TestV0010_PostMigrate_InitializesTokenFactoryParamsWhenMissing(t *testing.T) {
	tokenFactoryKeeper := &fakeTokenFactoryKeeper{}
	def := v0010(Dependencies{
		TokenFactoryKeeper: tokenFactoryKeeper,
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{
				existing: map[string]bool{},
			}
		},
	})

	require.NotNil(t, def.postMigrate)
	err := def.postMigrate(sdk.Context{})
	require.NoError(t, err)
	require.Equal(t, 1, tokenFactoryKeeper.createModuleAccountCalls)
	require.Equal(t, 1, tokenFactoryKeeper.setParamsCalls)
	require.Equal(t, tokenfactorytypes.DefaultParams(), tokenFactoryKeeper.lastParams)
}

func TestV0010_PostMigrate_SkipsParamsInitWhenAlreadySet(t *testing.T) {
	tokenFactoryKeeper := &fakeTokenFactoryKeeper{}
	def := v0010(Dependencies{
		TokenFactoryKeeper: tokenFactoryKeeper,
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{
				existing: map[string]bool{
					string(tokenfactorytypes.KeyDenomCreationFee):        true,
					string(tokenfactorytypes.KeyDenomCreationGasConsume): true,
				},
			}
		},
	})

	require.NotNil(t, def.postMigrate)
	err := def.postMigrate(sdk.Context{})
	require.NoError(t, err)
	require.Equal(t, 1, tokenFactoryKeeper.createModuleAccountCalls)
	require.Equal(t, 0, tokenFactoryKeeper.setParamsCalls)
}

func TestV0010_DefinesTokenFactoryStoreUpgrade(t *testing.T) {
	def := v0010(Dependencies{
		TokenFactoryKeeper: &fakeTokenFactoryKeeper{},
		GetSubspace: func(string) ParamSubspace {
			return fakeSubspace{}
		},
	})

	require.NotNil(t, def.storeUpgrades)
	require.Equal(t, []string{tokenfactorytypes.StoreKey}, def.storeUpgrades.Added)
}

type fakeUpgradeKeeper struct {
	handlers    map[string]upgradetypes.UpgradeHandler
	readPlan    upgradetypes.Plan
	readErr     error
	skipHeights map[int64]bool
}

func (f *fakeUpgradeKeeper) SetUpgradeHandler(upgradeName string, upgradeHandler upgradetypes.UpgradeHandler) {
	if f.handlers == nil {
		f.handlers = make(map[string]upgradetypes.UpgradeHandler)
	}
	f.handlers[upgradeName] = upgradeHandler
}

func (f *fakeUpgradeKeeper) ReadUpgradeInfoFromDisk() (upgradetypes.Plan, error) {
	if f.readErr != nil {
		return upgradetypes.Plan{}, f.readErr
	}
	return f.readPlan, nil
}

func (f *fakeUpgradeKeeper) IsSkipHeight(height int64) bool {
	return f.skipHeights[height]
}

type fakeModuleManager struct {
	calls      int
	lastFromVM module.VersionMap
	runErr     error
	resultVM   module.VersionMap
}

func (f *fakeModuleManager) RunMigrations(_ context.Context, _ module.Configurator, fromVM module.VersionMap) (module.VersionMap, error) {
	f.calls++
	f.lastFromVM = fromVM
	if f.runErr != nil {
		return nil, f.runErr
	}
	if f.resultVM != nil {
		return f.resultVM, nil
	}
	return fromVM, nil
}

type fakeTokenFactoryKeeper struct {
	createModuleAccountCalls int
	setParamsCalls           int
	lastParams               tokenfactorytypes.Params
}

func (f *fakeTokenFactoryKeeper) CreateModuleAccount(sdk.Context) {
	f.createModuleAccountCalls++
}

func (f *fakeTokenFactoryKeeper) SetParams(_ sdk.Context, params tokenfactorytypes.Params) {
	f.setParamsCalls++
	f.lastParams = params
}

type fakeSubspace struct {
	existing map[string]bool
}

func (f fakeSubspace) Has(_ sdk.Context, key []byte) bool {
	return f.existing[string(key)]
}

type fakeConfigurator struct{}

func (fakeConfigurator) RegisterService(*googlegrpc.ServiceDesc, interface{}) {}

func (fakeConfigurator) Error() error { return nil }

func (fakeConfigurator) MsgServer() gogogrpc.Server { return fakeGRPCServer{} }

func (fakeConfigurator) QueryServer() gogogrpc.Server { return fakeGRPCServer{} }

func (fakeConfigurator) RegisterMigration(string, uint64, module.MigrationHandler) error { return nil }

type fakeGRPCServer struct{}

func (fakeGRPCServer) RegisterService(*googlegrpc.ServiceDesc, interface{}) {}
