package scraper

import (
	"go.uber.org/fx"

	"github.com/jose/ratiodash/internal/domain"
)

// Registry maps scraper keys to their TrackerScraper implementations.
// Populated at startup via FX value groups — see Module below.
type Registry struct {
	scrapers map[string]domain.TrackerScraper
}

type registryParams struct {
	fx.In
	Scrapers []domain.TrackerScraper `group:"scrapers"`
}

func NewRegistry(p registryParams) *Registry {
	r := &Registry{scrapers: make(map[string]domain.TrackerScraper, len(p.Scrapers))}
	for _, s := range p.Scrapers {
		r.scrapers[s.Key()] = s
	}
	return r
}

func (r *Registry) Get(key string) (domain.TrackerScraper, bool) {
	s, ok := r.scrapers[key]
	return s, ok
}

func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.scrapers))
	for k := range r.scrapers {
		keys = append(keys, k)
	}
	return keys
}

// AsScraperRegistry adapts *Registry to the domain.ScraperRegistry interface for FX.
func AsScraperRegistry(r *Registry) domain.ScraperRegistry { return r }
