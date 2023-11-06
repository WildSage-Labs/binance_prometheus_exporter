package main

import (
	"net/http"
	"os"
	"time"

	"github.com/WildSage-Labs/binance_prometheus_exporter/internal/binance"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	core := zapcore.NewTee(
		//zapcore.NewCore(kafkaEncoder, topicErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		//zapcore.NewCore(kafkaEncoder, topicDebugging, lowPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)

	logger := zap.New(core)
	defer logger.Sync()

	bc := binance.NewBinanceClient(logger)
	ss, err := bc.GetSystemStatus()
	if err != nil {
		logger.Error("Failed to get Binance API status!", zap.Error(err))
		os.Exit(1)
	}

	if ss != binance.Online {
		logger.Error("Binance API is currently under maintenance, exiting...")
		os.Exit(1)
	}

	bc.GetFundingWallet()
	bc.GetUserAssets()

	e := echo.New()
	e.HideBanner = true
	e.Use(ZapLogger(logger))

	e.GET("/metrics", func(c echo.Context) error {

		funding := bc.GetFundingAssets()
		spot := bc.GetSpotAssets()

		return c.String(http.StatusOK, "OK")
	})

	e.Logger.Fatal(e.Start(":1323"))
}

// ZapLogger is an example of echo middleware that logs requests using logger "zap"
func ZapLogger(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()

			id := req.Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = res.Header().Get(echo.HeaderXRequestID)
			}

			fields := []zapcore.Field{
				zap.Int("status", res.Status),
				zap.String("latency", time.Since(start).String()),
				zap.String("id", id),
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.String("host", req.Host),
				zap.String("remote_ip", c.RealIP()),
			}

			n := res.Status
			switch {
			case n >= 500:
				log.Error("Server error", fields...)
			case n >= 400:
				log.Warn("Client error", fields...)
			case n >= 300:
				log.Info("Redirection", fields...)
			default:
				log.Info("Success", fields...)
			}

			return nil
		}
	}
}
