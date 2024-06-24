package cloud

import (
	"encoding/json"
	"fmt"
	"github.com/hudl/fargo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"sync"

	gelf "github.com/snovichkov/zap-gelf"

	"github.com/go-kit/kit/sd/eureka"
)

var mu sync.Mutex
var logCore *zapcore.Core
var log = NewLogger("cloud")

func BootstrapCloud() error {
	err := loadConfigServerProps()
	if err != nil {
		return err
	}
	// setup sending remote logs
	// https://medium.com/@alcbotta/logging-your-golang-application-to-elasticsearch-with-filebeat-12197a776ade
	// https://logz.io/blog/golang-logs/
	return nil
}

func RegisterInEureka() {
	eurekaAddr := os.Getenv("EUREKA_ADDR")
	localIp := os.Getenv("LOCAL_IP")
	appName := os.Getenv("APP_NAME")

	var fargoConfig fargo.Config
	fargoConfig.Eureka.ServiceUrls = []string{eurekaAddr}
	fargoConfig.Eureka.PollIntervalSeconds = 20
	fargoConfig.Eureka.RegisterWithEureka = true
	fargoConnection := fargo.NewConnFromConfig(fargoConfig)
	instance := &fargo.Instance{
		HostName:         localIp,
		Port:             8080,
		PortEnabled:      true,
		App:              appName,
		IPAddr:           localIp,
		VipAddress:       localIp,
		SecureVipAddress: localIp,
		HealthCheckUrl:   fmt.Sprintf("http://%s:8081/health", localIp),
		StatusPageUrl:    fmt.Sprintf("http://%s:8081/info", localIp),
		HomePageUrl:      fmt.Sprintf("http://%s:8081/info", localIp),
		Status:           fargo.STARTING,
		DataCenterInfo:   fargo.DataCenterInfo{Name: fargo.MyOwn},
		LeaseInfo:        fargo.LeaseInfo{RenewalIntervalInSecs: 25},
	}
	registrar1 := eureka.NewRegistrar(&fargoConnection, instance, &logAdapter{NewLogger("eureka")})
	registrar1.Register()
	log.Info("Registered in eureka")
}

type logAdapter struct {
	log *zap.SugaredLogger
}

func (adapter *logAdapter) Log(keyvals ...interface{}) error {
	adapter.log.Debugf("[Eureka] %v", keyvals...)
	return nil
}

func loadConfigServerProps() error {
	requestURL := os.Getenv("CONFIG_SERVER_URL")
	res, err := http.Get(requestURL)
	if err != nil {
		log.Errorf("Error making http request: %s\n", err)
		return err
	}
	defer res.Body.Close()

	var response struct {
		PropertySources []struct {
			Source map[string]string `json:"source"`
		} `json:"propertySources"`
	}
	json.NewDecoder(res.Body).Decode(&response)
	for _, psource := range response.PropertySources {
		for key, val := range psource.Source {
			if (os.Getenv(key)) == "" {
				os.Setenv(key, val)
			}
		}
	}
	log.Info("Loaded config server props")
	return nil
}

func BootstrapLogger() {
	logUrl := os.Getenv("LOG_URL")
	localIp := os.Getenv("LOCAL_IP")
	fmt.Printf("Connecting to gelf using url = %s\n", logUrl)
	logCor, err := gelf.NewCore(
		gelf.Addr(logUrl),
		gelf.Host(localIp))
	if err != nil {
		fmt.Printf("Failed to instantiate gelf core %v\n", err.Error())
		panic(err)
	}
	logCor.Enabled(zapcore.InfoLevel)
	logCore = &logCor
}

func NewLogger(category string) *zap.SugaredLogger {
	mu.Lock()
	if logCore == nil {
		BootstrapLogger()
	}
	mu.Unlock()
	var logger = zap.New(
		*logCore,
		zap.AddCaller(),
		zap.AddStacktrace(zap.LevelEnablerFunc(func(l zapcore.Level) bool {
			return (*logCore).Enabled(l)
		})),
	)
	return logger.Sugar().
		Named(category).
		With(zap.String("facility", "sample-go")).
		With(zap.String("source", category))
}
