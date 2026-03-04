package k6runner

import "encoding/json"

// K6Point represents a single data point from k6 JSON output.
type K6Point struct {
	Metric string
	Value  float64
	Time   string
}

type k6JSONLine struct {
	Type   string          `json:"type"`
	Metric string          `json:"metric"`
	Data   json.RawMessage `json:"data"`
}

type k6PointData struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

// ParseJSONLine parses a single line of k6 JSON output.
// Returns nil for non-Point types (Metric definitions, etc.).
func ParseJSONLine(line string) (*K6Point, error) {
	var jl k6JSONLine
	if err := json.Unmarshal([]byte(line), &jl); err != nil {
		return nil, err
	}
	if jl.Type != "Point" {
		return nil, nil
	}
	var pd k6PointData
	if err := json.Unmarshal(jl.Data, &pd); err != nil {
		return nil, err
	}
	return &K6Point{
		Metric: jl.Metric,
		Value:  pd.Value,
		Time:   pd.Time,
	}, nil
}
