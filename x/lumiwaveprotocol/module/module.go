package lumiwaveprotocol

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/math"
	"github.com/LumiWave/lumiwave-protocol/x/lumiwaveprotocol/keeper"
	"github.com/LumiWave/lumiwave-protocol/x/lumiwaveprotocol/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

var (
	_ module.AppModuleBasic = (*AppModule)(nil)
	_ module.AppModule      = (*AppModule)(nil)
	_ module.HasGenesis     = (*AppModule)(nil)

	_ appmodule.AppModule       = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker = (*AppModule)(nil)
	_ appmodule.HasEndBlocker   = (*AppModule)(nil)
)

var InflationSchedule = map[int]string{
	1: "0.3441",
	2: "0.2560",
	3: "0.1038",
	4: "0.0940",
	5: "0.0461",
	6: "0.0441",
}

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	cdc        codec.Codec
	keeper     keeper.Keeper
	authKeeper types.AuthKeeper
	bankKeeper types.BankKeeper
	mintKeeper mintkeeper.Keeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	authKeeper types.AuthKeeper,
	bankKeeper types.BankKeeper,
	mintKeeper mintkeeper.Keeper,
) AppModule {
	return AppModule{
		cdc:        cdc,
		keeper:     keeper,
		authKeeper: authKeeper,
		bankKeeper: bankKeeper,
		mintKeeper: mintKeeper,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the name of the module as a string.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec
func (AppModule) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(clientCtx.CmdContext, mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (AppModule) RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registrar)
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(registrar grpc.ServiceRegistrar) error {
	types.RegisterMsgServer(registrar, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(registrar, keeper.NewQueryServerImpl(am.keeper))

	return nil
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for testing.
func (am AppModule) DefaultGenesis(codec.JSONCodec) json.RawMessage {
	return am.cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (am AppModule) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := am.cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return genState.Validate()
}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	if err := am.cdc.UnmarshalJSON(gs, &genState); err != nil {
		panic(fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err))
	}

	if err := am.keeper.InitGenesis(ctx, genState); err != nil {
		panic(fmt.Errorf("failed to initialize %s genesis state: %w", types.ModuleName, err))
	}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	genState, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to export %s genesis state: %w", types.ModuleName, err))
	}

	bz, err := am.cdc.MarshalJSON(genState)
	if err != nil {
		panic(fmt.Errorf("failed to marshal %s genesis state: %w", types.ModuleName, err))
	}

	return bz
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block.
// The begin block implementation is optional.
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Calculate the current year based on block height (Assuming 5s block time)
	// For testing: Use 12, For production: Use 6307200
	const BlocksPerYear = 6307200
	currentHeight := sdkCtx.BlockHeight()

	// Year calculation: (currentHeight - 1) / BlocksPerYear + 1
	// e.g., Blocks 1-12 correspond to Year 1 when BlocksPerYear is 12
	year := int((currentHeight-1)/int64(BlocksPerYear)) + 1

	var targetMax math.LegacyDec

	// 2. Determine the Inflation Max rate for the current year
	if maxRateStr, ok := InflationSchedule[year]; ok {
		// Years 1-6: Use the predefined rates from the schedule table
		targetMax = math.LegacyMustNewDecFromStr(maxRateStr)
	} else if year > 6 {
		// Post-Year 6: Apply halving logic based on the Year 6 rate (0.0441)
		baseRate := math.LegacyMustNewDecFromStr(InflationSchedule[6])

		// Half-life cycle occurs every 2 years (e.g., 1 cycle for Years 7-8, 2 for Years 9-10)
		halfLifeCycles := (year - 5) / 2

		// Calculate 2^n for the denominator to determine the halved rate (1 / 2^n)
		denominator := int64(1)
		for i := 0; i < halfLifeCycles; i++ {
			denominator *= 2
		}

		// Apply halving: targetMax = baseRate / 2^n
		targetMax = baseRate.Quo(math.LegacyNewDec(denominator))
	} else {
		// Fallback for unexpected year values
		targetMax = math.LegacyZeroDec()
	}

	params, err := am.mintKeeper.Params.Get(sdkCtx)
	if err != nil {
		return nil
	}

	// 3. Update the Inflation Max parameter and enforce the cap (Clipping logic)
	if !params.InflationMax.Equal(targetMax) {
		params.InflationMax = targetMax
		params.InflationMin = math.LegacyZeroDec()

		// Persist the updated parameters to the store
		am.mintKeeper.Params.Set(sdkCtx, params)

		// Reset the current inflation rate if it exceeds the new target maximum
		minter, _ := am.mintKeeper.Minter.Get(sdkCtx)
		if minter.Inflation.GT(targetMax) {
			minter.Inflation = targetMax
			am.mintKeeper.Minter.Set(sdkCtx, minter)
		}

		// Log the transition for auditing and monitoring
		sdkCtx.Logger().Info("✅ Applied Year Transition and Halving",
			"Year", year,
			"NewMax", targetMax.String(),
		)
	}

	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(_ context.Context) error {
	return nil
}
