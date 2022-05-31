package env

import (
	"go.uber.org/zap"
	"time"
)

// DurationDefault return the result of searching an env var, if the env var value is empty, return a default value as time.Duration
func DurationDefault(log *zap.SugaredLogger, env, def string) time.Duration {
	orDefault := OrDefault(log, env, def)
	duration, err := time.ParseDuration(orDefault)
	if err != nil {
		log.Warn("error parsing ", orDefault, "as duration: ", err)
	}
	return duration
}
