// adapted from https://github.com/openfaas/golang-http-template/blob/master/template/golang-http/main.go
// Original license: MIT
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"handler/function"

	sdk "github.com/RafaySystems/function-templates/sdk/go"

	"github.com/RafaySystems/envmgr-pkgs/signals"
)

const defaultTimeout = 10 * time.Second

func main() {
	readTimeout := parseIntOrDurationValue(os.Getenv("read_timeout"), defaultTimeout)
	writeTimeout := parseIntOrDurationValue(os.Getenv("write_timeout"), defaultTimeout)

	functionSDK, err := sdk.NewFunctionSDK(
		sdk.WithReadTimeout(readTimeout),
		sdk.WithWriteTimeout(writeTimeout),
		sdk.WithHandler(function.Handle))
	if err != nil {
		fmt.Println("Error creating function SDK: ", err)
		return
	}

	ctx := signals.SetupSignalHandler()

	functionSDK.Run(ctx)
}

func parseIntOrDurationValue(val string, fallback time.Duration) time.Duration {
	if len(val) > 0 {
		parsedVal, parseErr := strconv.Atoi(val)
		if parseErr == nil && parsedVal >= 0 {
			return time.Duration(parsedVal) * time.Second
		}
	}

	duration, durationErr := time.ParseDuration(val)
	if durationErr != nil {
		return fallback
	}
	return duration
}
