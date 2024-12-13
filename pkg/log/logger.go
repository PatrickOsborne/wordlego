package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

type loggerKey struct{}

var defaultLogger *zap.Logger
var logger *zap.Logger
var mutex sync.RWMutex

func init() {
	defaultLogger = CreateDefaultLogger().Named("default")
}

func Get() *zap.Logger {
	mutex.RLock()
	defer mutex.RUnlock()

	if logger == nil {
		return defaultLogger
	}
	return logger
}

func FromCtx(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}
	return Get()
}

func WithCtx(ctx context.Context, logger *zap.Logger) context.Context {
	if foundLogger, existsInContext := ctx.Value(loggerKey{}).(*zap.Logger); existsInContext {
		if foundLogger == logger {
			return ctx
		}
	}

	return context.WithValue(ctx, loggerKey{}, logger)
}

func SetFromFile(path string) *zap.Logger {
	newLogger := LoadFromFile(path)
	return setLogger(newLogger)
}

// LoadFromFile load logger from config defined in path file, or return default logger
func LoadFromFile(path string) *zap.Logger {
	defaultLogger.Sugar().Infof("creating logger from log config file (%s)", path)

	newLogger, err := CreateLoggerFromFile(path)
	if err != nil {
		return defaultLogger
	}

	return newLogger
}

// CreateLoggerFromFile create zap logger from config defined in path file or return an error
func CreateLoggerFromFile(path string) (*zap.Logger, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		defaultLogger.Sugar().Errorf("failed to read log file (%s)", path)
		return nil, err
	}
	defaultLogger.Sugar().Debugf("logging yaml file (%s)", yamlFile)

	var cfg zap.Config
	if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
		defaultLogger.Sugar().Errorf("failed to unmarshall log config in log file (%s)", path)
		return nil, err
	}

	return cfg.Build()
}

// safely set the logger
func setLogger(newLogger *zap.Logger) *zap.Logger {
	mutex.Lock()
	defer mutex.Unlock()

	logger = newLogger
	return logger
}

func CreateDefaultLogger() *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "dateTime"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stdout",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
	}

	return zap.Must(config.Build())
}
