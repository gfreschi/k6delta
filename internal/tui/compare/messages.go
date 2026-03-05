package comparetui

import "github.com/gfreschi/k6delta/internal/report"

type errMsg struct{ err error }
type resultMsg struct{ result *report.ComparisonResult }
type exportDoneMsg struct{ path string }
