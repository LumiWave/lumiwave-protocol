package lumiwaveprotocol

import (
	"testing"

	"cosmossdk.io/math"
)

func TestCalculateYearFromHeightBoundaries(t *testing.T) {
	t.Parallel()

	const blocksPerYear int64 = 6307200

	testCases := []struct {
		name        string
		height      int64
		expectedYear int
	}{
		{name: "height 1 is year 1", height: 1, expectedYear: 1},
		{name: "last block of year 1", height: blocksPerYear, expectedYear: 1},
		{name: "first block of year 2", height: blocksPerYear + 1, expectedYear: 2},
		{name: "last block of year 2", height: blocksPerYear * 2, expectedYear: 2},
		{name: "first block of year 3", height: blocksPerYear*2 + 1, expectedYear: 3},
		{name: "non-positive height falls back to year 1", height: 0, expectedYear: 1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			year := calculateYearFromHeight(tc.height, blocksPerYear)
			if year != tc.expectedYear {
				t.Fatalf("height %d: expected year %d, got %d", tc.height, tc.expectedYear, year)
			}
		})
	}
}

func TestInflationMaxForYearBoundaries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		year     int
		expected string
	}{
		{name: "year 1 schedule rate", year: 1, expected: "0.3441"},
		{name: "year 6 schedule boundary", year: 6, expected: "0.0441"},
		{name: "year 7 first halving cycle", year: 7, expected: "0.02205"},
		{name: "year 8 same cycle as year 7", year: 8, expected: "0.02205"},
		{name: "year 9 second halving cycle", year: 9, expected: "0.011025"},
		{name: "year 10 same cycle as year 9", year: 10, expected: "0.011025"},
		{name: "invalid year fallback", year: 0, expected: "0.0000"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := inflationMaxForYear(tc.year)
			want := math.LegacyMustNewDecFromStr(tc.expected)
			if !got.Equal(want) {
				t.Fatalf("year %d: expected %s, got %s", tc.year, want.String(), got.String())
			}
		})
	}
}

func TestCalculateYearAndTargetInflation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		height        int64
		blocksPerYear uint64
		expectedYear  int
		expectedMax   string
		wantErr       bool
	}{
		{
			name:          "uses dynamic blocks_per_year for year boundary",
			height:        13,
			blocksPerYear: 12,
			expectedYear:  2,
			expectedMax:   "0.2560",
		},
		{
			name:          "year 7 uses halved rate when blocks_per_year is 12",
			height:        73, // (6 * 12) + 1
			blocksPerYear: 12,
			expectedYear:  7,
			expectedMax:   "0.02205",
		},
		{
			name:          "zero blocks_per_year returns error",
			height:        1,
			blocksPerYear: 0,
			wantErr:       true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			year, targetMax, err := calculateYearAndTargetInflation(tc.height, tc.blocksPerYear)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if year != tc.expectedYear {
				t.Fatalf("expected year %d, got %d", tc.expectedYear, year)
			}

			wantMax := math.LegacyMustNewDecFromStr(tc.expectedMax)
			if !targetMax.Equal(wantMax) {
				t.Fatalf("expected target max %s, got %s", wantMax.String(), targetMax.String())
			}
		})
	}
}
