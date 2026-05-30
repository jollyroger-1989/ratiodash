package scraper

import (
	"fmt"
	"html"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// applyFilters runs each filter in sequence on the value string and returns
// the final result.
func applyFilters(value string, filters []Filter) (string, error) {
	for _, f := range filters {
		var err error
		value, err = applyFilter(value, f)
		if err != nil {
			return "", fmt.Errorf("filter %q: %w", f.Name, err)
		}
	}
	return value, nil
}

func applyFilter(value string, f Filter) (string, error) {
	switch f.Name {
	case "replace":
		args := toStrings(f.Args)
		if len(args) < 2 {
			return "", fmt.Errorf("requires [from, to] args")
		}
		return strings.ReplaceAll(value, args[0], args[1]), nil

	case "re_replace":
		args := toStrings(f.Args)
		if len(args) < 2 {
			return "", fmt.Errorf("requires [pattern, replacement] args")
		}
		re, err := regexp.Compile(args[0])
		if err != nil {
			return "", err
		}
		return re.ReplaceAllString(value, args[1]), nil

	case "regexp":
		pattern, ok := f.Args.(string)
		if !ok {
			return "", fmt.Errorf("requires a string pattern")
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
		m := re.FindStringSubmatch(value)
		if len(m) == 0 {
			return "", nil
		}
		if len(m) > 1 {
			return m[1], nil // first capture group
		}
		return m[0], nil

	case "trim":
		if f.Args == nil || f.Args == "" {
			return strings.TrimSpace(value), nil
		}
		chars, _ := f.Args.(string)
		return strings.Trim(value, chars), nil

	case "append":
		suffix, _ := f.Args.(string)
		return value + suffix, nil

	case "prepend":
		prefix, _ := f.Args.(string)
		return prefix + value, nil

	case "tolower":
		return strings.ToLower(value), nil

	case "toupper":
		return strings.ToUpper(value), nil

	case "urlencode":
		return url.QueryEscape(value), nil

	case "urldecode":
		decoded, err := url.QueryUnescape(value)
		if err != nil {
			return "", err
		}
		return decoded, nil

	case "htmldecode":
		return html.UnescapeString(value), nil

	case "querystring":
		key, ok := f.Args.(string)
		if !ok {
			return "", fmt.Errorf("requires a string key")
		}
		parsed, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		return parsed.Query().Get(key), nil

	case "split":
		args := toStrings(f.Args)
		if len(args) < 2 {
			return "", fmt.Errorf("requires [separator, index] args")
		}
		parts := strings.Split(value, args[0])
		idx, err := strconv.Atoi(args[1])
		if err != nil || idx < 0 || idx >= len(parts) {
			return "", fmt.Errorf("invalid split index %q", args[1])
		}
		return parts[idx], nil

	case "parsebytes":
		n, err := parseBytes(value)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(n, 10), nil

	case "parsefloat":
		n, err := parseFloatValue(value)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(n, 'f', -1, 64), nil

	default:
		return "", fmt.Errorf("unknown filter %q", f.Name)
	}
}

// toStrings converts filter args to []string for multi-arg filters.
func toStrings(args interface{}) []string {
	if args == nil {
		return nil
	}
	switch v := args.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprint(item)
		}
		return result
	case string:
		return []string{v}
	default:
		return []string{fmt.Sprint(v)}
	}
}

// parseBytes converts a human-readable byte string to an int64 byte count.
//
// Supported suffixes (all treated as 1024-based multiples for compatibility
// with tracker conventions):
//
//	SI:     B, KB, MB, GB, TB, PB
//	Binary: KiB, MiB, GiB, TiB, PiB
//	French: o, Ko, Mo, Go, To, Po
func parseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	// Locate the boundary between the numeric part and the unit suffix.
	i := strings.LastIndexFunc(s, func(r rune) bool {
		return (r >= '0' && r <= '9') || r == '.' || r == ','
	})
	if i < 0 {
		return 0, fmt.Errorf("no numeric value in %q", s)
	}

	numStr := strings.TrimSpace(s[:i+1])
	numStr = strings.ReplaceAll(numStr, ",", ".") // normalise decimal separator
	unit := strings.TrimSpace(s[i+1:])

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing number in %q: %w", s, err)
	}

	const (
		ki = 1024
		mi = ki * 1024
		gi = mi * 1024
		ti = gi * 1024
		pi = ti * 1024
	)

	multipliers := map[string]float64{
		// bare bytes
		"": 1, "b": 1, "B": 1, "o": 1,
		// SI (treated as binary)
		"k": ki, "K": ki, "kb": ki, "KB": ki,
		"m": mi, "M": mi, "mb": mi, "MB": mi,
		"g": gi, "G": gi, "gb": gi, "GB": gi,
		"t": ti, "T": ti, "tb": ti, "TB": ti,
		"p": pi, "P": pi, "pb": pi, "PB": pi,
		// IEC binary
		"kib": ki, "KiB": ki,
		"mib": mi, "MiB": mi,
		"gib": gi, "GiB": gi,
		"tib": ti, "TiB": ti,
		"pib": pi, "PiB": pi,
		// French (binary)
		"Ko": ki, "Mo": mi, "Go": gi, "To": ti, "Po": pi,
	}

	mult, ok := multipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q in %q", unit, s)
	}

	return int64(math.Round(val * mult)), nil
}

// parseFloatValue parses a ratio/float string, mapping special symbols to 0.
// Accepted "no-data" symbols: ∞  Inf  inf  —  -  N/A  n/a  (empty string).
func parseFloatValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "∞", "Inf", "inf", "—", "-", "N/A", "n/a", "":
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}
