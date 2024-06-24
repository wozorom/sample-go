package main

import (
	"github.com/company/sample-go/internal/actuator"
	"github.com/company/sample-go/internal/app"
	"github.com/company/sample-go/internal/cloud"
	"log"
)

func main() {
	log.Println("[main] Application starting")

	go actuator.BootstrapActuator()
	err := cloud.BootstrapCloud()
	if err != nil {
		log.Println("[main] Application startup failed")
		return
	}
	app.BootstrapPersistence()
	go app.BootstrapKafkaProducer()
	go app.BootstrapKafkaConsumer()
	go cloud.RegisterInEureka()
	app.BootstrapWeb()

	log.Println("[main] Application finished")
}
