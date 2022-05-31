package env

import (
	"go.uber.org/zap"
	"os"
)

// Must return the result of searching an env var, if the env var value is empty, then return a fatal error
func Must(log *zap.SugaredLogger, env string) (value string) {
	value = os.Getenv(env)
	if value == "" {
		log.Fatalf("env var %s not set", env)
	}
	return value
}

// OrDefault return the result of searching an env var, if the env var value is empty, return a default value
func OrDefault(log *zap.SugaredLogger, env, def string) (value string) {
	value = os.Getenv(env)
	if value == "" {
		value = def
		log.Warnf("env var %s not set, using default %s", env, def)
	}
	return value
}
