package scraper

import (
	"go.uber.org/fx"
)

// Module wires the YAML scraper Loader and the Registry.
//
// YAML scrapers are discovered from the directory configured by SCRAPERS_DIR
// (default: ./scrapers). Each *.yml file in that directory is loaded as a
// TrackerScraper at startup.
//
// To register a code-defined scraper, append an fx.Provide call:
//
//	fx.Provide(
//	    fx.Annotate(
//	        NewMyScraper,
//	        fx.As(new(domain.TrackerScraper)),
//	        fx.ResultTags(`group:"scrapers"`),
//	    ),
//	),
var Module = fx.Options(
	fx.Provide(NewLoader),
	fx.Provide(
		NewRegistry,
		// Expose *Registry as domain.ScraperRegistry so services only depend on
		// the interface, not the concrete type.
		AsScraperRegistry,
	),
)
