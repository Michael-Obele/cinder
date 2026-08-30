package search

import (
	"testing"
)

func TestSearchCacheKeyDeterministic(t *testing.T) {
	a := SearchOptions{Query: "golang", Limit: 10, IncludeDomains: []string{"go.dev"}}
	b := SearchOptions{Query: "golang", Limit: 10, IncludeDomains: []string{"go.dev"}}
	c := SearchOptions{Query: "golang", Limit: 10, IncludeDomains: []string{"github.com"}}

	if searchCacheKey(a) != searchCacheKey(b) {
		t.Error("identical options must produce identical cache keys")
	}
	if searchCacheKey(a) == searchCacheKey(c) {
		t.Error("different filters must produce different cache keys")
	}
}
