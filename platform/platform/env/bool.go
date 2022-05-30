package env

import (
	"go.uber.org/zap"
	"strconv"
)

// BoolDefault return the result of searching an env var, if the env var value is empty, return a default value as bool
func BoolDefault(log *zap.SugaredLogger, env, def string) bool {
	orDefault := OrDefault(log, env, def)
	b, err := strconv.ParseBool(orDefault)
	if err != nil {
		log.Warn("error parsing ", orDefault, "as bool: ", err)
	}
	return b
}
