package beans

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/company/sample-go/internal/app/persistence"
)

var (
	MetricBuckets = []float64{0.01, 0.1, 0.5, 1}
	KafkaTopic    = "albums"
	Repo          persistence.AlbumRepository
	Producer      *kafka.Producer
	Consumer      *kafka.Consumer
)
