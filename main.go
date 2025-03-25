package main

import (
	"fhir-alarm/config"
	"fhir-alarm/fhir"
)

func main() {
	appConfig := config.LoadConfig()
	config.ConfigureLogger(appConfig)

	c := fhir.NewClient(appConfig)
	if err := c.ConnectAndBind(); err != nil {
		return
	}
}
