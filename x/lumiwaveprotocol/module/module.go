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
	"strconv"
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

func calculateYearFromHeight(currentHeight, blocksPerYear int64) int {
	if currentHeight <= 0 || blocksPerYear <= 0 {
		return 1
	}

	return int((currentHeight-1)/blocksPerYear) + 1
}

func inflationMaxForYear(year int) math.LegacyDec {
	if maxRateStr, ok := InflationSchedule[year]; ok {
		return math.LegacyMustNewDecFromStr(maxRateStr)
	}

	if year > 6 {
		baseRate := math.LegacyMustNewDecFromStr(InflationSchedule[6])
		halfLifeCycles := (year - 5) / 2

		// cap at 62 to prevent int64 overflow
		if halfLifeCycles > 62 {
			halfLifeCycles = 62
		}
		denominator := int64(1)
		for i := 0; i < halfLifeCycles; i++ {
			denominator *= 2
		}

		return baseRate.Quo(math.LegacyNewDec(denominator))
	}

	return math.LegacyZeroDec()
}

func calculateYearAndTargetInflation(currentHeight int64, blocksPerYear uint64) (int, math.LegacyDec, error) {
	if blocksPerYear == 0 {
		return 0, math.LegacyZeroDec(), fmt.Errorf("blocks_per_year is zero")
	}
	// prevent overflow when casting uint64 to int64
	if blocksPerYear > uint64(1<<63-1) {
		return 0, math.LegacyZeroDec(), fmt.Errorf("blocks_per_year exceeds max int64")
	}

	year := calculateYearFromHeight(currentHeight, int64(blocksPerYear))
	return year, inflationMaxForYear(year), nil
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
	currentHeight := sdkCtx.BlockHeight()

	params, err := am.mintKeeper.Params.Get(sdkCtx)
	if err != nil {
		return fmt.Errorf("failed to get mint params (height=%d): %w", currentHeight, err)
	}

	// 1. Calculate the current year from the active mint params.
	year, targetMax, err := calculateYearAndTargetInflation(currentHeight, params.BlocksPerYear)
	if err != nil {
		return fmt.Errorf("invalid mint params: %w (height=%d)", err, currentHeight)
	}

	// 3. Update the Inflation Max parameter and enforce the cap (Clipping logic)
	if !params.InflationMax.Equal(targetMax) {
		params.InflationMax = targetMax
		params.InflationMin = math.LegacyZeroDec()

		// Persist the updated parameters to the store
		if err := am.mintKeeper.Params.Set(sdkCtx, params); err != nil {
			return fmt.Errorf("failed to set mint params (height=%d, year=%d, target_max=%s): %w", currentHeight, year, targetMax.String(), err)
		}

		// Reset the current inflation rate if it exceeds the new target maximum
		minter, err := am.mintKeeper.Minter.Get(sdkCtx)
		if err != nil {
			return fmt.Errorf("failed to get mint minter (height=%d, year=%d): %w", currentHeight, year, err)
		}
		inflationClipped := minter.Inflation.GT(targetMax)
		if inflationClipped {
			minter.Inflation = targetMax
			if err := am.mintKeeper.Minter.Set(sdkCtx, minter); err != nil {
				return fmt.Errorf("failed to set mint minter (height=%d, year=%d, target_max=%s): %w", currentHeight, year, targetMax.String(), err)
			}
		}

		// emit SDK event for indexers and block explorers
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"inflation_schedule_transition",
				sdk.NewAttribute("year", strconv.Itoa(year)),
				sdk.NewAttribute("height", strconv.FormatInt(currentHeight, 10)),
				sdk.NewAttribute("new_inflation_max", targetMax.String()),
				sdk.NewAttribute("inflation_clipped", strconv.FormatBool(inflationClipped)),
			),
		)

		sdkCtx.Logger().Info("Applied Year Transition and Halving",
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
