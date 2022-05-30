package env

import (
	"go.uber.org/zap"
	"strconv"
)

// IntDefault return the result of searching an env var, if the env var value is empty, return a default value as int
func IntDefault(log *zap.SugaredLogger, env, def string) int {
	orDefault := OrDefault(log, env, def)
	duration, err := strconv.Atoi(orDefault)
	if err != nil {
		log.Warn("error parsing ", orDefault, "as int: ", err)
	}
	return duration
}
