package scraper

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/pkg/config"
)

// Loader discovers and parses all YAML scraper definitions from a directory.
type Loader struct {
	dir string
}

// NewLoader creates a Loader that reads definitions from cfg.ScrapersDir.
func NewLoader(cfg *config.Config) *Loader {
	return &Loader{dir: cfg.ScrapersDir}
}

// Load reads every *.yml file in the configured directory and returns one
// YAMLScraper per valid definition. If the directory does not exist the call
// succeeds with an empty slice so the app can start without a scrapers dir.
func (l *Loader) Load() ([]domain.TrackerScraper, error) {
	info, err := os.Stat(l.dir)
	if os.IsNotExist(err) {
		return nil, nil // no directory is fine
	}
	if err != nil {
		return nil, fmt.Errorf("checking scrapers dir %q: %w", l.dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scrapers path %q is not a directory", l.dir)
	}

	pattern := filepath.Join(l.dir, "*.yml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing %q: %w", pattern, err)
	}

	scrapers := make([]domain.TrackerScraper, 0, len(matches))
	for _, path := range matches {
		s, err := loadFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %q: %w", path, err)
		}
		scrapers = append(scrapers, s)
	}

	return scrapers, nil
}

func loadFile(path string) (*YAMLScraper, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if def.ID == "" {
		return nil, fmt.Errorf("definition is missing required field 'id'")
	}
	if def.Stats.Path == "" {
		return nil, fmt.Errorf("definition %q is missing stats.path", def.ID)
	}

	return &YAMLScraper{def: def}, nil
}
