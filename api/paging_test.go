package api

import (
	"math"
	"testing"
)

func TestComputePageBoundsRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name                string
		count, page, items  int
		wantFrom, wantTo    int
		wantPage, wantPages int
	}{
		{name: "first page", count: 25, page: 0, items: 10, wantFrom: 0, wantTo: 10, wantPage: 0, wantPages: 3},
		{name: "last page", count: 25, page: 2, items: 10, wantFrom: 20, wantTo: 25, wantPage: 2, wantPages: 3},
		{name: "huge page", count: 25, page: math.MaxInt, items: 10, wantFrom: 20, wantTo: 25, wantPage: 2, wantPages: 3},
		{name: "negative page", count: 25, page: -8, items: 10, wantFrom: 0, wantTo: 10, wantPage: 0, wantPages: 3},
		{name: "invalid item count", count: 25, page: 0, items: 0, wantFrom: 0, wantTo: 0, wantPage: 0, wantPages: 0},
		{name: "empty", count: 0, page: math.MaxInt, items: 10, wantFrom: 0, wantTo: 0, wantPage: 0, wantPages: 0},
		{name: "max count", count: math.MaxInt, page: math.MaxInt, items: 10, wantFrom: math.MaxInt - 7, wantTo: math.MaxInt, wantPage: math.MaxInt / 10, wantPages: math.MaxInt/10 + 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			from, to, page, pages := computePageBounds(test.count, test.page, test.items)
			if from != test.wantFrom || to != test.wantTo || page != test.wantPage || pages != test.wantPages {
				t.Fatalf("computePageBounds() = (%d, %d, %d, %d), want (%d, %d, %d, %d)", from, to, page, pages, test.wantFrom, test.wantTo, test.wantPage, test.wantPages)
			}
		})
	}
}
