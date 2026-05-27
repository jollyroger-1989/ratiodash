package scraper

import (
	"go.uber.org/fx"

	"github.com/jose/ratiodash/internal/domain"
)

// Module wires the Registry and all registered tracker scrapers.
//
// To register a new scraper, append an fx.Provide call:
//
//	fx.Provide(
//	    fx.Annotate(
//	        NewMyScraper,
//	        fx.As(new(domain.TrackerScraper)),
//	        fx.ResultTags(`group:"scrapers"`),
//	    ),
//	),
var Module = fx.Options(
	fx.Provide(
		NewRegistry,
		// Expose *Registry as domain.ScraperRegistry so services only depend on
		// the interface, not the concrete type.
		AsScraperRegistry,
	),
	fx.Provide(
		fx.Annotate(
			NewUnit3DScraper,
			fx.As(new(domain.TrackerScraper)),
			fx.ResultTags(`group:"scrapers"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewC411Scraper,
			fx.As(new(domain.TrackerScraper)),
			fx.ResultTags(`group:"scrapers"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewTorr9Scraper,
			fx.As(new(domain.TrackerScraper)),
			fx.ResultTags(`group:"scrapers"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewYggRebornScraper,
			fx.As(new(domain.TrackerScraper)),
			fx.ResultTags(`group:"scrapers"`),
		),
	),
)
