package binance

// SystemStatus represents binance  API status. Either online or under maintenance
type SystemStatus uint

const (
	Online SystemStatus = iota
	Maintenance
)

func (ss SystemStatus) String() string {
	switch ss {
	case 0:
		return "Online"
	case 1:
		return "Under maintenance"
	}
	return "Unknown Status"
}

/** Main Structure definitions **/
type (
	/*
		APIStatus is used to determine the status of the binance api
	*/
	APIStatus struct {
		Status  SystemStatus `json:"status"`
		Message string       `json:"msg"`
	}

	Asset struct {
		Asset        string `json:"asset"`
		Free         string `json:"free"`
		Locked       string `json:"locked"`
		Freeze       string `json:"freeze"`
		Withdrawing  string `json:"withdrawing"`
		Ipoable      string `json:"ipoable"`
		BtcValuation string `json:"btcValuation"`
	}
)
