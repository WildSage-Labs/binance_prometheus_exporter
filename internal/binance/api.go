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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Entrio/subenv"
	"go.uber.org/zap"
)

var endpoints = [...]string{"https://api.binance.com", "https://api-gcp.binance.com", "https://api1.binance.com", "https://api2.binance.com", "https://api3.binance.com", "https://api4.binance.com"}

type (
	Client struct {
		httpclient http.Client
		logger     *zap.Logger
		security   security
		funding    Data
		spot       Data
	}
	security struct {
		PublicKey  string `json:"-"`
		PrivateKey string `json:"-"`
	}
	Data struct {
		Assets []Asset
		lock   sync.RWMutex
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
		funding: Data{
			Assets: make([]Asset, 0),
		},
		spot: Data{
			Assets: make([]Asset, 0),
		},
	}
}

func (c *Client) GetSpotAssets() []Asset {
	// Make a copy of an asset array
	c.spot.lock.RLock()
	defer c.spot.lock.RUnlock()
	var res []Asset
	res = append(res, c.spot.Assets...)
	return res
}

func (c *Client) GetFundingAssets() []Asset {
	// Make a copy of an asset array
	c.funding.lock.RLock()
	defer c.funding.lock.RUnlock()
	var res []Asset
	res = append(res, c.funding.Assets...)
	return res
}

/*
*
generateSignature uses Client's private key to generate a sha256 hash of provided string.
*/
func (s security) generateSignature(payload string) string {
	//TODO: Generate actual signature

	mac := hmac.New(sha256.New, []byte(s.PrivateKey))
	mac.Reset()
	mac.Write([]byte(payload))
	expectedMAC := mac.Sum(nil)
	return hex.EncodeToString(expectedMAC)
}

/*
*
signrequest grabs the uri, assigns timestamp to it and signs it. URI afterwards is re-assembled and signature is appended
*/
func (c *Client) signrequest(uri string, signed bool) string {
	// Split the url at ? to get the part of the URI we need to sign
	extracted := strings.Split(uri, "?")
	timeStampInMillis := fmt.Sprintf("%d", time.Now().UnixMilli())
	var newUri, root string
	// Do we have any query string after url?
	if len(extracted) == 1 {
		// we have nada, just a plan url
		root = uri
		newUri = fmt.Sprintf("timestamp=%s", timeStampInMillis)
	} else {
		newUri = fmt.Sprintf("%s&timestamp=%s", extracted[1], timeStampInMillis)
		root = extracted[0]
	}

	signature := c.security.generateSignature(newUri)
	signedUri := fmt.Sprintf("%s?%s&signature=%s", root, newUri, signature)
	c.logger.Debug("Generated HMAC sha1 signature for url", zap.String("sha256", signature), zap.String("uri", newUri))
	return signedUri
}

func (c *Client) GetSystemStatus() (SystemStatus, error) {
	c.logger.Debug("GetSystemStatus()")
	req, cancel, err := c.buildGetRequest("sapi/v1/system/status")
	c.logger.Debug("Making status request", zap.String("URL", fmt.Sprintf("%s%s", req.Host, req.URL.Path)))
	if err != nil {
		return Maintenance, err
	}
	defer cancel()

	res, err := c.httpclient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request.", zap.Error(err))
		return Maintenance, err
	}
	defer func() {
		_ = res.Body.Close() // Hate those unhandled errors warning
	}()
	c.logger.Debug("Got server status response", zap.Int("status_code", res.StatusCode))
	status := &APIStatus{}
	err = json.NewDecoder(res.Body).Decode(status)
	if err != nil {
		c.logger.Error("Failed to decode body.", zap.Error(err))
		return Maintenance, err
	}
	c.logger.Info("System status", zap.String("status", fmt.Sprintf("%s", status.Status)))
	return status.Status, nil
}

func (c *Client) GetFundingWallet() {
	c.logger.Debug("GetFundingWallet()")
	req, cancel, err := c.buildPostRequest("sapi/v1/asset/get-funding-asset")
	c.logger.Debug("Making funding wallet data request", zap.String("URL", req.URL.String()))
	if err != nil {
		c.logger.Warn("Failed to form funding wallet request.", zap.Error(err))
		return
	}
	defer cancel()

	res, err := c.httpclient.Do(req)
	if err != nil {
		c.logger.Warn("Failed to get funding wallet data.", zap.Error(err))
		return
	}

	defer res.Body.Close()

	c.logger.Debug("Got server status response", zap.Int("status_code", res.StatusCode))

	if res.StatusCode != 200 {
		c.logger.Warn("Got an invalid status code from API, returning")
		return
	}
	var assets []Asset
	err = json.NewDecoder(res.Body).Decode(&assets)
	if err != nil {
		c.logger.Error("Failed to decode body.", zap.Error(err))
		return
	}
	c.funding.lock.Lock()
	defer c.funding.lock.Unlock()
	c.funding.Assets = assets
}

func (c *Client) GetUserAssets() {
	c.logger.Debug("GetFundingWallet()")
	req, cancel, err := c.buildPostRequest("sapi/v3/asset/getUserAsset")
	c.logger.Debug("Making funding wallet data request", zap.String("URL", req.URL.String()))
	if err != nil {
		c.logger.Warn("Failed to form funding wallet request.", zap.Error(err))
		return
	}
	defer cancel()

	res, err := c.httpclient.Do(req)
	if err != nil {
		c.logger.Warn("Failed to get funding wallet data.", zap.Error(err))
		return
	}

	defer res.Body.Close()

	c.logger.Debug("Got server status response", zap.Int("status_code", res.StatusCode))

	if res.StatusCode != 200 {
		c.logger.Warn("Got an invalid status code from API, returning")
		return
	}
	var assets []Asset
	err = json.NewDecoder(res.Body).Decode(&assets)
	if err != nil {
		c.logger.Error("Failed to decode body.", zap.Error(err))
		return
	}
	c.spot.lock.Lock()
	defer c.spot.lock.Unlock()
	c.spot.Assets = assets
}

func (c *Client) buildGetRequest(url string) (*http.Request, func(), error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	r, e := http.NewRequestWithContext(ctx, http.MethodGet, buildURL(url), nil)
	r.Header.Set("X-MBX-APIKEY", c.security.PublicKey)
	return r, cancel, e
}

func (c *Client) buildPostRequest(url string) (*http.Request, func(), error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	signedUrl := c.signrequest(url, true)
	r, e := http.NewRequestWithContext(ctx, http.MethodPost, buildURL(signedUrl), nil)
	r.Header.Set("X-MBX-APIKEY", c.security.PublicKey)
	return r, cancel, e
}

func buildURL(url string) string {
	return fmt.Sprintf("%s/%s", endpoints[1], url)
}
