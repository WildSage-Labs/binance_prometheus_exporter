package prometheus

type (
	Gauge struct {
		Name string // Actual name that appears after #TYPE
		Type string // Display this is f,d,s etc
	}
)
