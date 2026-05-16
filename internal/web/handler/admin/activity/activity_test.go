package activity

import (
	"testing"
)

func TestBuildPageLinks(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		want    []pageLink
	}{
		{
			name:    "single page",
			current: 1,
			total:   1,
			want: []pageLink{
				{Number: 1, Active: true},
			},
		},
		{
			name:    "two pages, current first",
			current: 1,
			total:   2,
			want: []pageLink{
				{Number: 1, Active: true},
				{Number: 2},
			},
		},
		{
			name:    "small total fits without ellipsis",
			current: 4,
			total:   7,
			want: []pageLink{
				{Number: 1},
				{Number: 2},
				{Number: 3},
				{Number: 4, Active: true},
				{Number: 5},
				{Number: 6},
				{Number: 7},
			},
		},
		{
			name:    "current at start, trailing ellipsis only",
			current: 1,
			total:   20,
			want: []pageLink{
				{Number: 1, Active: true},
				{Number: 2},
				{Number: 3},
				{Ellipsis: true},
				{Number: 20},
			},
		},
		{
			name:    "current near start, no leading ellipsis",
			current: 2,
			total:   20,
			want: []pageLink{
				{Number: 1},
				{Number: 2, Active: true},
				{Number: 3},
				{Number: 4},
				{Ellipsis: true},
				{Number: 20},
			},
		},
		{
			name:    "current in middle, both ellipses",
			current: 10,
			total:   20,
			want: []pageLink{
				{Number: 1},
				{Ellipsis: true},
				{Number: 8},
				{Number: 9},
				{Number: 10, Active: true},
				{Number: 11},
				{Number: 12},
				{Ellipsis: true},
				{Number: 20},
			},
		},
		{
			name:    "current near end, no trailing ellipsis",
			current: 19,
			total:   20,
			want: []pageLink{
				{Number: 1},
				{Ellipsis: true},
				{Number: 17},
				{Number: 18},
				{Number: 19, Active: true},
				{Number: 20},
			},
		},
		{
			name:    "current at last page",
			current: 20,
			total:   20,
			want: []pageLink{
				{Number: 1},
				{Ellipsis: true},
				{Number: 18},
				{Number: 19},
				{Number: 20, Active: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPageLinks(tt.current, tt.total)
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d want %d; got=%+v", len(got), len(tt.want), got)
			}

			for i, g := range got {
				w := tt.want[i]
				if g != w {
					t.Errorf("links[%d] = %+v, want %+v", i, g, w)
				}
			}
		})
	}
}

func TestBuildPageLinks_ExactlyOneActive(t *testing.T) {
	for total := 1; total <= 30; total++ {
		for current := 1; current <= total; current++ {
			links := buildPageLinks(current, total)
			activeCount := 0
			activeNum := 0

			for _, l := range links {
				if l.Active {
					activeCount++
					activeNum = l.Number
				}
			}

			if activeCount != 1 {
				t.Errorf("total=%d current=%d: activeCount=%d, want 1 (links=%+v)", total, current, activeCount, links)
			}

			if activeNum != current {
				t.Errorf("total=%d current=%d: active page=%d, want %d", total, current, activeNum, current)
			}
		}
	}
}

func TestBuildPageLinks_AlwaysContainsFirstAndLast(t *testing.T) {
	for total := 1; total <= 50; total++ {
		for current := 1; current <= total; current++ {
			links := buildPageLinks(current, total)

			firstSeen, lastSeen := false, false

			for _, l := range links {
				if l.Ellipsis {
					continue
				}

				if l.Number == 1 {
					firstSeen = true
				}

				if l.Number == total {
					lastSeen = true
				}
			}

			if !firstSeen {
				t.Errorf("total=%d current=%d: page 1 missing", total, current)
			}

			if !lastSeen {
				t.Errorf("total=%d current=%d: page %d missing", total, current, total)
			}
		}
	}
}

func TestBuildPageLinks_NoAdjacentEllipses(t *testing.T) {
	for total := 1; total <= 50; total++ {
		for current := 1; current <= total; current++ {
			links := buildPageLinks(current, total)

			for i := 1; i < len(links); i++ {
				if links[i-1].Ellipsis && links[i].Ellipsis {
					t.Errorf("total=%d current=%d: adjacent ellipses at %d", total, current, i)
				}
			}
		}
	}
}
