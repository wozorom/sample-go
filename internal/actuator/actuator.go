package actuator

import (
	"fmt"
	"github.com/dimiro1/health"
	"github.com/company/sample-go/internal/app/beans"
	"github.com/company/sample-go/internal/cloud"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"strings"
	"sync"
)

var log = cloud.NewLogger("actuator")

func BootstrapActuator() {
	handler := health.NewHandler()
	handler.AddChecker("aerospike", &AerospikeChecker{})
	handler.AddChecker("kafka", &KafkaChecker{})
	http.Handle("/health", handler)
	http.HandleFunc("/info", infoHandler)
	http.Handle("/prometheus", promhttp.Handler())
	log.Info("Listening actuator port 8081")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Error("Can not listen port 8081")
		log.Sync()
		panic("Can not listen port 8081")
	}
}

var info = ""
var mu = sync.Mutex{}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	if info == "" {
		mu.Lock()
		dat, _ := os.ReadFile("./go.mod")
		info = string(dat)
		mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, info)
}

type AerospikeChecker struct {
}

func (ch *AerospikeChecker) Check() health.Health {
	if beans.Repo == nil {
		return newHealth(true)
	}
	_, err := beans.Repo.Get("_")
	return newHealth(err == nil || !strings.Contains(err.Error(), "KEY_NOT_FOUND_ERROR"))
}

type KafkaChecker struct {
}

func (ch *KafkaChecker) Check() health.Health {
	if beans.Producer == nil {
		return newHealth(true)
	}

	return newHealth(beans.Producer.IsClosed())
}

func newHealth(down bool) health.Health {
	h := health.NewHealth()
	if down {
		h.Down()
		return h
	}
	h.Up()
	return h
}
