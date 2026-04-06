package keeper

import (
	"context"
	"encoding/json"
	"strings"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k Keeper) setBeforeSendHook(ctx sdk.Context, denom string, cosmwasmAddress string) error {
	// verify that denom is an x/tokenfactory denom
	_, _, err := types.DeconstructDenom(denom)
	if err != nil {
		return err
	}

	store := k.GetDenomPrefixStore(ctx, denom)

	// delete the store for denom prefix store when cosmwasm address is nil
	if cosmwasmAddress == "" {
		store.Delete([]byte(types.BeforeSendHookAddressPrefixKey))
		return nil
	} else {
		if k.contractKeeper == nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "contract keeper not configured")
		}

		// validate the contract exists and supports the full before-send hook interface
		cacheCtx, _ := ctx.CacheContext()

		cwAddr, err := sdk.AccAddressFromBech32(cosmwasmAddress)
		if err != nil {
			return err
		}

		// verify TrackBeforeSend support
		trackMsg := types.TrackBeforeSendSudoMsg{
			TrackBeforeSend: types.TrackBeforeSendMsg{},
		}
		trackMsgBz, err := json.Marshal(trackMsg)
		if err != nil {
			return err
		}
		_, err = k.contractKeeper.Sudo(cacheCtx, cwAddr, trackMsgBz)
		if err != nil {
			if strings.Contains(err.Error(), "no such contract") {
				return err
			}
			return errorsmod.Wrapf(err, "contract does not support track_before_send")
		}

		// verify BlockBeforeSend support
		blockMsg := types.BlockBeforeSendSudoMsg{
			BlockBeforeSend: types.BlockBeforeSendMsg{},
		}
		blockMsgBz, err := json.Marshal(blockMsg)
		if err != nil {
			return err
		}
		cacheCtx2, _ := ctx.CacheContext()
		_, err = k.contractKeeper.Sudo(cacheCtx2, cwAddr, blockMsgBz)
		if err != nil {
			if strings.Contains(err.Error(), "no such contract") {
				return err
			}
			return errorsmod.Wrapf(err, "contract does not support block_before_send")
		}
	}

	_, err = sdk.AccAddressFromBech32(cosmwasmAddress)
	if err != nil {
		return err
	}

	store.Set([]byte(types.BeforeSendHookAddressPrefixKey), []byte(cosmwasmAddress))

	return nil
}

func (k Keeper) GetBeforeSendHook(ctx sdk.Context, denom string) string {
	store := k.GetDenomPrefixStore(ctx, denom)

	bz := store.Get([]byte(types.BeforeSendHookAddressPrefixKey))
	if bz == nil {
		return ""
	}

	return string(bz)
}

func (k Keeper) GetAllBeforeSendHooks(ctx sdk.Context) ([]string, []string) {
	denomsList := []string{}
	beforeSendHooksList := []string{}

	iterator := k.GetAllDenomsIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Value())

		beforeSendHook := k.GetBeforeSendHook(ctx, denom)
		if beforeSendHook != "" {
			denomsList = append(denomsList, denom)
			beforeSendHooksList = append(beforeSendHooksList, beforeSendHook)
		}
	}
	return denomsList, beforeSendHooksList
}

// Hooks wrapper struct for bank keeper
type Hooks struct {
	k Keeper
}

var _ types.BankHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// TrackBeforeSend calls the before send listener contract suppresses any errors
func (h Hooks) TrackBeforeSend(ctx context.Context, from, to sdk.AccAddress, amount sdk.Coins) {
	_ = h.k.callBeforeSendListener(ctx, from, to, amount, false)
}

// TrackBeforeSend calls the before send listener contract returns any errors
func (h Hooks) BlockBeforeSend(ctx context.Context, from, to sdk.AccAddress, amount sdk.Coins) error {
	return h.k.callBeforeSendListener(ctx, from, to, amount, true)
}

// callBeforeSendListener iterates over each coin and sends corresponding sudo msg to the contract address stored in state.
// If blockBeforeSend is true, sudoMsg wraps BlockBeforeSendMsg, otherwise sudoMsg wraps TrackBeforeSendMsg.
// Note that we gas meter trackBeforeSend to prevent infinite contract calls.
// CONTRACT: this should not be called in beginBlock or endBlock since out of gas will cause this method to panic.
func (k Keeper) callBeforeSendListener(context context.Context, from, to sdk.AccAddress, amount sdk.Coins, blockBeforeSend bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errorsmod.Wrapf(types.ErrBeforeSendHookOutOfGas, "%v", r)
		}
	}()

	ctx := sdk.UnwrapSDKContext(context)
	for _, coin := range amount {
		cosmwasmAddress := k.GetBeforeSendHook(ctx, coin.Denom)
		if cosmwasmAddress != "" {
			cwAddr, err := sdk.AccAddressFromBech32(cosmwasmAddress)
			if err != nil {
				return err
			}

			var msgBz []byte

			// get msgBz, either BlockBeforeSend or TrackBeforeSend
			// Note that for trackBeforeSend, we need to gas meter computations to prevent infinite loop
			// specifically because module to module sends are not gas metered.
			// We don't need to do this for blockBeforeSend since blockBeforeSend is not called during module to module sends.
			if blockBeforeSend {
				msg := types.BlockBeforeSendSudoMsg{
					BlockBeforeSend: types.BlockBeforeSendMsg{
						From:   from.String(),
						To:     to.String(),
						Amount: cwCoinFromSDKCoin(coin),
					},
				}
				msgBz, err = json.Marshal(msg)
			} else {
				msg := types.TrackBeforeSendSudoMsg{
					TrackBeforeSend: types.TrackBeforeSendMsg{
						From:   from.String(),
						To:     to.String(),
						Amount: cwCoinFromSDKCoin(coin),
					},
				}
				msgBz, err = json.Marshal(msg)
			}
			if err != nil {
				return err
			}
			em := sdk.NewEventManager()

			// Check remaining gas in parent context and use the lesser of the fixed limit and remaining gas
			gasLimit := min(ctx.GasMeter().GasRemaining(), types.BeforeSendHookGasLimit)

			childCtx := ctx.WithGasMeter(storetypes.NewGasMeter(gasLimit))

			// Execute the contract call with proper gas tracking and panic recovery

			func() {
				defer func() {
					// Always consume gas from child context to parent, even if contract panics
					ctx.GasMeter().ConsumeGas(childCtx.GasMeter().GasConsumed(), "track before send gas")
					if r := recover(); r != nil {
						err = errorsmod.Wrapf(types.ErrBeforeSendHookOutOfGas, "%v", r)
					}
				}()
				_, err = k.contractKeeper.Sudo(childCtx.WithEventManager(em), cwAddr, msgBz)
			}()

			if err != nil {
				if strings.Contains(err.Error(), "no such contract") {
					return nil
				}
				if k.IsModuleAcc(ctx, from) {
					return nil
				}

				return errorsmod.Wrapf(err, "failed to call before send hook for denom %s", coin.Denom)
			}
		}
	}
	return nil
}

func cwCoinFromSDKCoin(coin sdk.Coin) wasmvmtypes.Coin {
	return wasmvmtypes.Coin{
		Denom:  coin.Denom,
		Amount: coin.Amount.String(),
	}
}
