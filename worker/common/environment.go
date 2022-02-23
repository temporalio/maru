package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.temporal.io/sdk/log"
)

func GetEnvOrDefaultString(logger log.Logger, envVarName string, defaultValue string) string {
	value := os.Getenv(envVarName)
	if value == "" {
		logger.Info(fmt.Sprintf("'%s' env variable not set, defaulting to '%s'", envVarName, defaultValue))
		value = defaultValue
	} else {
		logger.Info(fmt.Sprintf("'%s' env variable read as '%s'", envVarName, value))
	}
	return value
}

func GetEnvOrDefaultBool(logger log.Logger, envVarName string, defaultValue bool) bool {
	result := defaultValue
	envValue := os.Getenv(envVarName)
	switch strings.ToLower(envValue) {
	case "true":
		logger.Info(fmt.Sprintf("'%s' env variable set to '%s'", envVarName, envValue))
		result = true
	case "false":
		logger.Info(fmt.Sprintf("'%s' env variable set to '%s'", envVarName, envValue))
		result = false
	case "":
		logger.Info(fmt.Sprintf("'%s' env variable not set, defaulting to '%t'", envVarName, defaultValue))
	default:
		logger.Info(fmt.Sprintf("'%s' env variable set to unknown value '%s', defaulting to '%t'", envVarName, envValue, result))
	}
	return result
}

func GetEnvOrDefaultInt(logger log.Logger, envVarName string, defaultValue int) int {
	value := defaultValue
	envValue := os.Getenv(envVarName)

	if envValue == "" {
		logger.Info(fmt.Sprintf("'%s' env variable not set, defaulting to '%v'", envVarName, defaultValue))
		value = defaultValue
	} else {
		parsedValue, err := strconv.Atoi(envValue)
		if err != nil {
			logger.Info(fmt.Sprintf("error parsing '%s' env variable, defaulting to '%v'. err: %v", envVarName, defaultValue, err))
		} else {
			value = parsedValue
		}
	}

	return value
}
