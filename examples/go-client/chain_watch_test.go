package main

import (
	"context"
	"testing"
	"time"
)

func TestWatchLatestBlocks_Once(t *testing.T) {
	fetchCalls := 0
	callbackCalls := 0

	err := watchLatestBlocks(
		context.Background(),
		time.Millisecond,
		true,
		func(context.Context) (*BlockSnapshot, error) {
			fetchCalls++
			return &BlockSnapshot{Height: 10}, nil
		},
		func(*BlockSnapshot) error {
			callbackCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("watchLatestBlocks failed: %v", err)
	}
	if fetchCalls == 0 {
		t.Fatal("fetch should be called at least once")
	}
	if callbackCalls != 1 {
		t.Fatalf("callback should be called once, got %d", callbackCalls)
	}
}

func TestWatchLatestBlocks_OnlyIncreasingHeights(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heights := []int64{7, 7, 8, 8, 9}
	index := 0
	called := make([]int64, 0, 3)

	err := watchLatestBlocks(
		ctx,
		time.Millisecond,
		false,
		func(context.Context) (*BlockSnapshot, error) {
			if index >= len(heights) {
				cancel()
				return &BlockSnapshot{Height: heights[len(heights)-1]}, nil
			}
			h := heights[index]
			index++
			return &BlockSnapshot{Height: h}, nil
		},
		func(snapshot *BlockSnapshot) error {
			called = append(called, snapshot.Height)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("watchLatestBlocks failed: %v", err)
	}

	if len(called) != 3 {
		t.Fatalf("expected 3 callback calls, got %d", len(called))
	}
	if called[0] != 7 || called[1] != 8 || called[2] != 9 {
		t.Fatalf("unexpected callback heights: %+v", called)
	}
}

func TestWatchLatestBlocks_InvalidInterval(t *testing.T) {
	err := watchLatestBlocks(
		context.Background(),
		0,
		false,
		func(context.Context) (*BlockSnapshot, error) { return &BlockSnapshot{Height: 1}, nil },
		func(*BlockSnapshot) error { return nil },
	)
	if err == nil {
		t.Fatal("expected error for non-positive interval")
	}
}
