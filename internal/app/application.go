package app

import (
	"github.com/Depado/ginprom"
	aero "github.com/aerospike/aerospike-client-go/v7"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/company/sample-go/internal/app/beans"
	"github.com/company/sample-go/internal/app/persistence"
	"github.com/company/sample-go/internal/app/web"
	"github.com/company/sample-go/internal/cloud"
	"go.uber.org/zap"
	"os"
	"time"
)

var log = cloud.NewLogger("app")

func BootstrapPersistence() {
	// https://github.com/aerospike/aerospike-client-go
	aerospikeHostname := os.Getenv("AEROSPIKE_HOST")
	client, err := aero.NewClient(aerospikeHostname, 3000)
	panicOnError(err)
	beans.Repo = persistence.NewRepo(client)
	log.Infof("[aerospike] Creating Aerospike client for host=%s", aerospikeHostname)
}

func BootstrapKafkaProducer() {
	// https://docs.confluent.io/kafka-clients/go/current/overview.html
	kafkaUrl := os.Getenv("KAFKA_URL")
	log.Infof("[kafka-producer] Creating kafka producer with host=%s", kafkaUrl)
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaUrl,
		"client.id":         "local",
		"security.protocol": "PLAINTEXT",
		"acks":              "all"})
	panicOnError(err)
	log.Infof("[kafka-producer] Created producer %s", p.String())
	beans.Producer = p
}

func BootstrapKafkaConsumer() {
	kafkaUrl := os.Getenv("KAFKA_URL")
	log.Infof("[kafka-consumer] Creating kafka consumer with host=%s", kafkaUrl)
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaUrl,
		"group.id":          "foo",
		"security.protocol": "PLAINTEXT",
		"auto.offset.reset": "smallest"})
	panicOnError(err)
	beans.Consumer = c
	defer beans.Consumer.Close()
	err = beans.Consumer.SubscribeTopics([]string{beans.KafkaTopic}, nil)
	panicOnError(err)
	log.Infof("[kafka-consumer] Listening kafka topics=%s", beans.KafkaTopic)
	for {
		message, err := beans.Consumer.ReadMessage(100 * time.Millisecond)
		if err == nil {
			log.Infof("[kafka-consumer] Received event: %s\n", string(message.Value))
		}
	}
}

func BootstrapWeb() {
	//gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	setupGin(router)
	router.GET("/albums/:id", web.GetAlbumByID)
	router.DELETE("/albums/:id", web.DeleteAlbumByID)
	router.POST("/albums", web.PostAlbum)
	log.Infof("[web] Starting listening port 8080")
	err := router.Run(":8080")
	panicOnError(err)
}

func setupGin(router *gin.Engine) {
	adapter := ZapLoggerAdapter{log}
	router.Use(ginzap.Ginzap(&adapter, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(&adapter, true))
	p := ginprom.New(
		ginprom.Engine(router),
		ginprom.Subsystem("pub"),
		ginprom.BucketSize(beans.MetricBuckets))
	router.Use(p.Instrument())
}

func panicOnError(err error) {
	if err != nil {
		log.Errorf("Received error: %s", err.Error())
		log.Sync()
		panic(err)
	}
}

type ZapLoggerAdapter struct {
	aLog *zap.SugaredLogger
}

func (ad *ZapLoggerAdapter) Info(msg string, fields ...zap.Field) {
	ad.aLog.Info(msg)

}

func (ad *ZapLoggerAdapter) Error(msg string, fields ...zap.Field) {
	ad.aLog.Error(msg)
}
