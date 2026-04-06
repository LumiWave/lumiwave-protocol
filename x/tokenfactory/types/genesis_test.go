package types_test

import (
	"testing"

	"github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
)

func TestGenesisDenonBeforeSendHookMarshalRoundtrip(t *testing.T) {
	hookAddr := "lumi14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sqyfc2m"

	original := types.GenesisDenom{
		Denom:             "factory/lumi1creator/test",
		AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
		BeforeSendHook:    hookAddr,
	}

	// Marshal
	bz, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded types.GenesisDenom
	if err := decoded.Unmarshal(bz); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Denom != original.Denom {
		t.Errorf("Denom mismatch: got %q, want %q", decoded.Denom, original.Denom)
	}
	if decoded.AuthorityMetadata.Admin != original.AuthorityMetadata.Admin {
		t.Errorf("Admin mismatch: got %q, want %q", decoded.AuthorityMetadata.Admin, original.AuthorityMetadata.Admin)
	}
	if decoded.BeforeSendHook != hookAddr {
		t.Errorf("BeforeSendHook mismatch: got %q, want %q", decoded.BeforeSendHook, hookAddr)
	}
}

func TestGenesisDenonBeforeSendHookEmpty(t *testing.T) {
	original := types.GenesisDenom{
		Denom:             "factory/lumi1creator/test",
		AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
		BeforeSendHook:    "",
	}

	bz, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded types.GenesisDenom
	if err := decoded.Unmarshal(bz); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.BeforeSendHook != "" {
		t.Errorf("BeforeSendHook should be empty, got %q", decoded.BeforeSendHook)
	}
}

func TestGenesisDenonEqual(t *testing.T) {
	a := types.GenesisDenom{
		Denom:             "factory/lumi1creator/test",
		AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
		BeforeSendHook:    "lumi1hook",
	}
	b := types.GenesisDenom{
		Denom:             "factory/lumi1creator/test",
		AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
		BeforeSendHook:    "lumi1hook",
	}
	c := types.GenesisDenom{
		Denom:             "factory/lumi1creator/test",
		AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
		BeforeSendHook:    "lumi1different",
	}

	if !a.Equal(b) {
		t.Error("Equal GenesisDenoms should be equal")
	}
	if a.Equal(c) {
		t.Error("GenesisDenoms with different BeforeSendHook should not be equal")
	}
}

func TestGenesisStateMarshalWithHooks(t *testing.T) {
	hookAddr := "lumi14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sqyfc2m"

	gs := types.GenesisState{
		Params: types.DefaultParams(),
		FactoryDenoms: []types.GenesisDenom{
			{
				Denom:             "factory/lumi1creator/tokenA",
				AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
				BeforeSendHook:    hookAddr,
			},
			{
				Denom:             "factory/lumi1creator/tokenB",
				AuthorityMetadata: types.DenomAuthorityMetadata{Admin: "lumi1creator"},
				BeforeSendHook:    "",
			},
		},
	}

	bz, err := gs.Marshal()
	if err != nil {
		t.Fatalf("GenesisState Marshal failed: %v", err)
	}

	var decoded types.GenesisState
	if err := decoded.Unmarshal(bz); err != nil {
		t.Fatalf("GenesisState Unmarshal failed: %v", err)
	}

	if len(decoded.FactoryDenoms) != 2 {
		t.Fatalf("Expected 2 denoms, got %d", len(decoded.FactoryDenoms))
	}

	if decoded.FactoryDenoms[0].BeforeSendHook != hookAddr {
		t.Errorf("tokenA hook mismatch: got %q, want %q", decoded.FactoryDenoms[0].BeforeSendHook, hookAddr)
	}
	if decoded.FactoryDenoms[1].BeforeSendHook != "" {
		t.Errorf("tokenB hook should be empty, got %q", decoded.FactoryDenoms[1].BeforeSendHook)
	}
}
