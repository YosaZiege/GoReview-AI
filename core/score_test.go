package core

import "testing"

func TestCalculate(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     int
	}{
		{
			name:     "no findings gives 100",
			findings: []Finding{},
			want:     100,
		},
		{
			name: "2 CRITIQUE + 1 MOYEN gives 63",
			findings: []Finding{
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
				{Severity: SeverityMedium},
			},
			want: 63, // 100 - 15 - 15 - 7
		},
		{
			name: "score clamped to 0 when findings exceed 100",
			findings: func() []Finding {
				// 7 CRITIQUE = -105, clamped to 0
				findings := make([]Finding, 7)
				for i := range findings {
					findings[i] = Finding{Severity: SeverityCritical}
				}
				return findings
			}(),
			want: 0,
		},
		{
			name: "FAIBLE deducts 3",
			findings: []Finding{
				{Severity: SeverityLow},
			},
			want: 97,
		},
		{
			name: "mixed severities",
			findings: []Finding{
				{Severity: SeverityCritical},
				{Severity: SeverityMedium},
				{Severity: SeverityLow},
			},
			want: 75, // 100 - 15 - 7 - 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate(tt.findings)
			if got != tt.want {
				t.Errorf("Calculate() = %d, want %d", got, tt.want)
			}
		})
	}
}
