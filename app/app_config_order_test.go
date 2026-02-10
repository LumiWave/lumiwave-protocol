package app

import (
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	lumiwaveprotocolmoduletypes "github.com/LumiWave/lumiwave-protocol/x/lumiwaveprotocol/types"
)

type testAppOptions map[string]interface{}

func (o testAppOptions) Get(key string) interface{} {
	return o[key]
}

func TestBeginBlockerOrder_LumiwaveprotocolBeforeMint(t *testing.T) {
	db := dbm.NewMemDB()
	appOpts := testAppOptions{
		flags.FlagHome: DefaultNodeHome,
	}

	app := New(log.NewNopLogger(), db, nil, true, appOpts, baseapp.SetChainID("order-test-chain"))
	t.Cleanup(func() {
		require.NoError(t, app.Close())
	})

	beginOrder := app.ModuleManager.OrderBeginBlockers
	require.NotEmpty(t, beginOrder)

	lumiIdx := indexOf(beginOrder, lumiwaveprotocolmoduletypes.ModuleName)
	mintIdx := indexOf(beginOrder, minttypes.ModuleName)

	require.NotEqual(t, -1, lumiIdx, "lumiwaveprotocol must exist in begin blocker order")
	require.NotEqual(t, -1, mintIdx, "mint must exist in begin blocker order")
	require.Less(t, lumiIdx, mintIdx, "lumiwaveprotocol must run before mint in begin blocker order")
}

func indexOf(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}

	return -1
}
