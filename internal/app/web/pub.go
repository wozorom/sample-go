package web

import (
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	"github.com/company/sample-go/internal/app/beans"
	"github.com/company/sample-go/internal/app/clients"
	"github.com/company/sample-go/internal/app/model"
	"github.com/company/sample-go/internal/cloud"
	"net/http"
	"time"
)

var log = cloud.NewLogger("web")

func PostAlbum(c *gin.Context) {
	var newAlbum model.Album

	if err := c.BindJSON(&newAlbum); err != nil {
		log.Error(err)
		return
	}
	if err := beans.Repo.Save(&newAlbum); err != nil {
		log.Error(err)
		return
	}
	payload, err := json.Marshal(&newAlbum)
	if err != nil {
		log.Error(err)
		return
	}

	deliveryChan := make(chan kafka.Event, 1)
	err = beans.Producer.Produce(&kafka.Message{
		Value:          payload,
		Key:            payload,
		Timestamp:      time.Now(),
		Headers:        make([]kafka.Header, 0),
		TopicPartition: kafka.TopicPartition{Topic: &beans.KafkaTopic, Partition: kafka.PartitionAny}},
		deliveryChan)
	if err != nil {
		log.Error(err)
		return
	}
	beans.Producer.Flush(100)
	c.IndentedJSON(http.StatusCreated, newAlbum)
}

func GetAlbumByID(c *gin.Context) {
	id := c.Param("id")

	result, err := beans.Repo.Get(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
	}
	if err := clients.PerformRemoteServiceCall(); err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Remote call failed"})
		return
	}
	c.IndentedJSON(http.StatusOK, result)
}

func DeleteAlbumByID(c *gin.Context) {
	id := c.Param("id")

	result, err := beans.Repo.Delete(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
	}
	if !result {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album was NOT deleted"})
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "album was deleted successfully"})
}
