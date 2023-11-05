package binance

/**
Binance API security reference. (https://binance-docs.github.io/apidocs/spot/en/#endpoint-security-type)

Security Type	Description
NONE			Endpoint can be accessed freely.
TRADE			Endpoint requires sending a valid API-Key and signature.
MARGIN			Endpoint requires sending a valid API-Key and signature.
USER_DATA		Endpoint requires sending a valid API-Key and signature.
USER_STREAM		Endpoint requires sending a valid API-Key.
MARKET_DATA		Endpoint requires sending a valid API-Key.

TRADE, MARGIN and USER_DATA endpoints are SIGNED endpoints.
*/

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"os"

	"github.com/Entrio/subenv"
	"go.uber.org/zap"
)

type (
	Client struct {
		httpclient http.Client
		logger     *zap.Logger
		security   security
	}
	security struct {
		PublicKey  string `json:"-"`
		PrivateKey string `json:"-"`
	}
)

func NewBinanceClient(l *zap.Logger) *Client {
	// Fetch private and public keys from the environment
	privKey := subenv.Env("B_PRIVATE_KEY", "")
	pubkey := subenv.Env("B_PUBLIC_KEY", "")

	if len(privKey) == 0 {
		l.Error("Failed to create a new binance client! B_PRIVATE_KEY variable was not set.")
		os.Exit(1)
	}

	if len(pubkey) == 0 {
		l.Error("Failed to create a new binance client! B_PUBLIC_KEY variable was not set.")
		os.Exit(1)
	}

	return &Client{
		httpclient: http.Client{},
		logger:     l,
		security: security{
			PublicKey:  pubkey,
			PrivateKey: privKey,
		},
	}
}

func (s security) generateSignature(payload string) string {
	//TODO: Generate actual signature

	mac := hmac.New(sha1.New, []byte(s.PrivateKey))
	mac.Reset()
	mac.Write([]byte(payload))
	expectedMAC := mac.Sum(nil)
	return hex.EncodeToString(expectedMAC)
}

func (a *Client) prepareRequest(uri string, signed bool) {
	// We need to get current timestamp in ms and add it to the request, if the request is signed add sig to end
	// Need to attach public key top headers too!
}

func (a *Client) GetSystemStatus() SystemStatus {
	return Maintenance
}
