package clients

import (
	"github.com/go-resty/resty/v2"
	"github.com/company/sample-go/internal/cloud"
	"os"
)

var log = cloud.NewLogger("clients")

func PerformRemoteServiceCall() error {
	client := resty.New()
	host := os.Getenv("REMOTE_URL")
	resp, err := client.R().
		EnableTrace().
		SetHeader("Accept", "application/json").
		Get(host)
	if err != nil {
		return err
	}
	log.Infof("[external] Received remote response from %s -> %s", host, resp.String())
	return nil
}
