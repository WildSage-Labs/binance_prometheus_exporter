package binance

// SystemStatus represents binance  API status. Either online or under maintenance
type SystemStatus uint

const (
	Online SystemStatus = iota
	Maintenance
)

type (
	APIStatus struct {
		Status  SystemStatus `json:"status"`
		Message string       `json:"message"`
	}
)
