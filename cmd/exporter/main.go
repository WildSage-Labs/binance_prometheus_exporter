package main

import (
	"os"

	"github.com/WildSage-Labs/binance_prometheus_exporter/internal/binance"
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
}
