package query

import (
	"net/url"
	"testing"
)

var testSchema = Schema{
	"name": {Column: "name", Sortable: true, Filterable: true, Partial: true},
	"age":  {Column: "age", Sortable: true, Filterable: true},
	"note": {Column: "note", Filterable: true, Partial: true}, // not sortable
}

func TestParseSort(t *testing.T) {
	tests := []struct {
		name string
		sort string
		want []Sort
	}{
		{"single ascending", "name", []Sort{{Column: "name", Desc: false}}},
		{"single descending", "-name", []Sort{{Column: "name", Desc: true}}},
		{"multiple fields priority order", "-age,name", []Sort{
			{Column: "age", Desc: true},
			{Column: "name", Desc: false},
		}},
		{"unknown field is dropped", "unknown", nil},
		{"non-sortable field is dropped", "note", nil},
		{"empty sort", "", nil},
		{"whitespace around field", " name ", []Sort{{Column: "name", Desc: false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{"sort": {tt.sort}}
			got := Parse(values, testSchema).Sorts
			if !equalSorts(got, tt.want) {
				t.Errorf("Sorts = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  *Filter
	}{
		{"filterable partial field", "filter[name]", "bob", &Filter{Column: "name", Value: "bob", Partial: true}},
		{"filterable exact field", "filter[age]", "30", &Filter{Column: "age", Value: "30", Partial: false}},
		{"unknown field dropped", "filter[unknown]", "x", nil},
		{"malformed key dropped", "filterage", "30", nil},
		{"empty value dropped", "filter[name]", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{tt.key: {tt.value}}
			got := Parse(values, testSchema).Filters
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("Filters = %+v, want empty", got)
				}
				return
			}
			if len(got) != 1 || got[0] != *tt.want {
				t.Errorf("Filters = %+v, want [%+v]", got, *tt.want)
			}
		})
	}
}

func TestParseCombinedSortAndFilter(t *testing.T) {
	values := url.Values{
		"sort":         {"-age"},
		"filter[name]": {"ann"},
	}
	opts := Parse(values, testSchema)

	if len(opts.Sorts) != 1 || opts.Sorts[0] != (Sort{Column: "age", Desc: true}) {
		t.Errorf("Sorts = %+v", opts.Sorts)
	}
	if len(opts.Filters) != 1 || opts.Filters[0] != (Filter{Column: "name", Value: "ann", Partial: true}) {
		t.Errorf("Filters = %+v", opts.Filters)
	}
}

func equalSorts(a, b []Sort) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
