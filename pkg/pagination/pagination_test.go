package pagination

import "testing"

func TestNewParams(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		limit     int
		wantPage  int
		wantLimit int
	}{
		{"valid values pass through", 2, 20, 2, 20},
		{"zero page defaults to 1", 0, 20, 1, 20},
		{"negative page defaults to 1", -5, 20, 1, 20},
		{"zero limit defaults to 10", 2, 0, 2, 10},
		{"negative limit defaults to 10", 2, -5, 2, 10},
		{"limit above max is clamped", 1, 1000, 1, 100},
		{"limit at max passes through", 1, 100, 1, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewParams(tt.page, tt.limit)
			if got.Page != tt.wantPage || got.Limit != tt.wantLimit {
				t.Errorf("NewParams(%d, %d) = %+v, want {Page:%d Limit:%d}",
					tt.page, tt.limit, got, tt.wantPage, tt.wantLimit)
			}
		})
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name string
		p    Params
		want int
	}{
		{"first page", Params{Page: 1, Limit: 10}, 0},
		{"second page", Params{Page: 2, Limit: 10}, 10},
		{"third page different limit", Params{Page: 3, Limit: 25}, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Offset(); got != tt.want {
				t.Errorf("Offset() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewMeta(t *testing.T) {
	tests := []struct {
		name      string
		p         Params
		total     int64
		wantPages int
	}{
		{"exact multiple", Params{Page: 1, Limit: 10}, 30, 3},
		{"remainder rounds up", Params{Page: 1, Limit: 10}, 31, 4},
		{"zero total", Params{Page: 1, Limit: 10}, 0, 0},
		{"zero limit avoids divide by zero", Params{Page: 1, Limit: 0}, 10, 0},
		{"fewer items than limit", Params{Page: 1, Limit: 10}, 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMeta(tt.p, tt.total)
			if got.TotalPages != tt.wantPages {
				t.Errorf("TotalPages = %d, want %d", got.TotalPages, tt.wantPages)
			}
			if got.Total != tt.total {
				t.Errorf("Total = %d, want %d", got.Total, tt.total)
			}
			if got.Page != tt.p.Page || got.Limit != tt.p.Limit {
				t.Errorf("Meta Page/Limit = %d/%d, want %d/%d", got.Page, got.Limit, tt.p.Page, tt.p.Limit)
			}
		})
	}
}
