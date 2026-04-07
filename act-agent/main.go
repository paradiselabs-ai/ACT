package main

import (
	"github.com/paradiselabs-ai/ACT/act-agent/cmd"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	cmd.Execute()
}
