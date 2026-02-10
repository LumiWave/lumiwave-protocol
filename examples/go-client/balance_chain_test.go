package main

import "testing"

func TestRunBalanceCommandValidation(t *testing.T) {
	t.Parallel()

	if err := runBalanceCommand(nil); err == nil {
		t.Fatal("expected error for missing balance subcommand")
	}
	if err := runBalanceCommand([]string{"unknown"}); err == nil {
		t.Fatal("expected error for unknown balance subcommand")
	}
	if err := runBalanceAll(nil); err == nil {
		t.Fatal("expected error for missing address")
	}
	if err := runBalanceByDenom(nil); err == nil {
		t.Fatal("expected error for missing address")
	}
}

func TestRunChainCommandValidation(t *testing.T) {
	t.Parallel()

	if err := runChainCommand(nil); err == nil {
		t.Fatal("expected error for missing chain subcommand")
	}
	if err := runChainCommand([]string{"unknown"}); err == nil {
		t.Fatal("expected error for unknown chain subcommand")
	}
	if err := runChainAccount(nil); err == nil {
		t.Fatal("expected error for missing account address")
	}
}
