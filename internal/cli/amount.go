package cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/diomonogatari/hydrate-cli/internal/config"
)

const mlPerOz = 29.5735

// parseAmount interprets an amount argument for `hydrate log`. An empty string
// means "one glass" (the configured default). Otherwise it accepts a bare
// number (millilitres) or a unit-suffixed value: "500", "500ml", "16oz", "1l".
func parseAmount(arg string, cfg config.Config) (int, error) {
	s := strings.ToLower(strings.TrimSpace(arg))
	if s == "" {
		return cfg.GlassML, nil
	}

	var (
		numStr string
		factor float64
	)
	switch {
	case strings.HasSuffix(s, "ml"):
		numStr, factor = strings.TrimSuffix(s, "ml"), 1
	case strings.HasSuffix(s, "oz"):
		numStr, factor = strings.TrimSuffix(s, "oz"), mlPerOz
	case strings.HasSuffix(s, "l"):
		numStr, factor = strings.TrimSuffix(s, "l"), 1000
	default:
		numStr, factor = s, 1
	}

	n, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q (try 500, 500ml, 16oz, or 1l)", arg)
	}
	ml := int(math.Round(n * factor))
	if ml <= 0 {
		return 0, fmt.Errorf("amount must be positive, got %q", arg)
	}
	return ml, nil
}
