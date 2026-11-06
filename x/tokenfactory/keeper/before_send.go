package keeper

import (
	"context"
	"encoding/json"
	"strings"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
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

		cwAddr, err := sdk.AccAddressFromBech32(cosmwasmAddress)
		if err != nil {
			return err
		}

		if err := k.validateBeforeSendHookContract(ctx, cwAddr, denom); err != nil {
			return err
		}
	}

	_, err = sdk.AccAddressFromBech32(cosmwasmAddress)
	if err != nil {
		return err
	}

	store.Set([]byte(types.BeforeSendHookAddressPrefixKey), []byte(cosmwasmAddress))

	return nil
}

// validateBeforeSendHookContract calls the contract with well-formed hook messages in
// discarded cache contexts, to confirm it can receive both of them.
func (k Keeper) validateBeforeSendHookContract(ctx sdk.Context, cwAddr sdk.AccAddress, denom string) error {
	trackMsg, blockMsg, err := beforeSendProbeMsgs(cwAddr.String(), denom)
	if err != nil {
		return err
	}

	cacheCtx, _ := ctx.CacheContext()
	_, err = k.contractKeeper.Sudo(cacheCtx, cwAddr, trackMsg)
	if err := checkBeforeSendProbe(err, "track_before_send"); err != nil {
		return err
	}

	cacheCtx2, _ := ctx.CacheContext()
	_, err = k.contractKeeper.Sudo(cacheCtx2, cwAddr, blockMsg)
	return checkBeforeSendProbe(err, "block_before_send")
}

// beforeSendProbeMsgs builds the validation messages. The fields are well formed because
// contracts usually parse amount as a Coin, which rejects an empty amount.
func beforeSendProbeMsgs(contractAddr, denom string) (trackMsg, blockMsg []byte, err error) {
	amount := wasmvmtypes.Coin{Denom: denom, Amount: "0"}

	trackMsg, err = json.Marshal(types.TrackBeforeSendSudoMsg{
		TrackBeforeSend: types.TrackBeforeSendMsg{From: contractAddr, To: contractAddr, Amount: amount},
	})
	if err != nil {
		return nil, nil, err
	}

	blockMsg, err = json.Marshal(types.BlockBeforeSendSudoMsg{
		BlockBeforeSend: types.BlockBeforeSendMsg{From: contractAddr, To: contractAddr, Amount: amount},
	})
	if err != nil {
		return nil, nil, err
	}

	return trackMsg, blockMsg, nil
}

// checkBeforeSendProbe decides whether a validation call shows the contract can handle the
// hook message. An error returned by the contract's own handler is accepted, since it may
// refuse the probe by policy. A parse failure on a well-formed message or a VM error is not:
// a contract that cannot handle block_before_send would fail every transfer of the denom.
func checkBeforeSendProbe(err error, hook string) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "no such contract") {
		return err
	}

	if wasmtypes.ErrExecuteFailed.Is(err) &&
		!strings.Contains(err.Error(), "Error parsing into type") &&
		!strings.Contains(err.Error(), "unknown variant") {
		return nil
	}

	return errorsmod.Wrapf(err, "contract does not support %s", hook)
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
