package common

// Severity represents a 3-tier status level used across metriccard tiles and verdicts.
type Severity int

const (
	SeverityOK    Severity = iota
	SeverityWarn
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityWarn:
		return "warn"
	case SeverityError:
		return "error"
	default:
		return "ok"
	}
}

// SeverityThresholds defines the boundaries for severity classification.
type SeverityThresholds struct {
	WarnRatio  float64 // ratio at which OK -> Warn (e.g., 0.80)
	ErrorRatio float64 // ratio at which Warn -> Error (e.g., 0.95)
}

// DefaultSeverityThresholds is the standard threshold set for metriccard tiles.
var DefaultSeverityThresholds = SeverityThresholds{
	WarnRatio:  0.80,
	ErrorRatio: 0.95,
}

// SeverityFromRatio returns the severity tier for a given value/max ratio.
func SeverityFromRatio(ratio float64, t SeverityThresholds) Severity {
	switch {
	case ratio >= t.ErrorRatio:
		return SeverityError
	case ratio >= t.WarnRatio:
		return SeverityWarn
	default:
		return SeverityOK
	}
}

// Delta tier threshold constants (percentage boundaries).
const (
	DeltaNeutralPct  = 2.0  // below this: neutral
	DeltaMildPct     = 5.0  // 2-5%: mild improvement/regression
	DeltaModeratePct = 15.0 // 5-15%: moderate; above: strong/severe
)
