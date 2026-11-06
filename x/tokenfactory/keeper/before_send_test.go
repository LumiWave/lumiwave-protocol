package keeper

import (
	"encoding/json"
	"errors"
	"testing"

	errorsmod "cosmossdk.io/errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"
)

func TestCheckBeforeSendProbe(t *testing.T) {
	const hook = "block_before_send"

	tests := []struct {
		name   string
		err    error
		reject bool
	}{
		{name: "handler succeeded"},
		{
			name: "contract refused the probe by policy",
			err:  errorsmod.Wrap(wasmtypes.ErrExecuteFailed, "Generic error: sender not allowed"),
		},
		{
			name:   "no such contract",
			err:    errorsmod.Wrap(wasmtypes.ErrNoSuchContractFn("lumi1abc"), "lookup"),
			reject: true,
		},
		{
			name:   "contract has no sudo export",
			err:    errorsmod.Wrap(wasmtypes.ErrVMError, "Could not get export: Missing export sudo"),
			reject: true,
		},
		{
			name:   "other VM error",
			err:    errorsmod.Wrap(wasmtypes.ErrVMError, "Aborted: panicked at contract"),
			reject: true,
		},
		{
			name:   "message variant not handled",
			err:    errorsmod.Wrap(wasmtypes.ErrExecuteFailed, "Error parsing into type hook::SudoMsg: unknown variant `block_before_send`"),
			reject: true,
		},
		{
			name:   "message fields not handled",
			err:    errorsmod.Wrap(wasmtypes.ErrExecuteFailed, "Error parsing into type hook::SudoMsg: missing field `coin`"),
			reject: true,
		},
		{
			name:   "unrelated error type",
			err:    errors.New("store failure"),
			reject: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkBeforeSendProbe(tc.err, hook)
			if tc.reject {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBeforeSendProbeMsgsAreWellFormed(t *testing.T) {
	const (
		addr  = "lumi1contract"
		denom = "factory/lumi1creator/token"
	)

	trackMsg, blockMsg, err := beforeSendProbeMsgs(addr, denom)
	require.NoError(t, err)

	for key, raw := range map[string][]byte{"track_before_send": trackMsg, "block_before_send": blockMsg} {
		var decoded map[string]struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Amount struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"amount"`
		}
		require.NoError(t, json.Unmarshal(raw, &decoded))
		require.Len(t, decoded, 1)

		msg, ok := decoded[key]
		require.True(t, ok, key)
		require.Equal(t, addr, msg.From)
		require.Equal(t, addr, msg.To)
		require.Equal(t, denom, msg.Amount.Denom)
		require.Equal(t, "0", msg.Amount.Amount, "a Coin cannot parse an empty amount")
	}
}
