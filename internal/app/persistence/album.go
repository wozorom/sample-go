package persistence

import (
	aero "github.com/aerospike/aerospike-client-go/v7"
	"github.com/company/sample-go/internal/app/model"
	"github.com/company/sample-go/internal/app/persistence/aerospike"
)

type AlbumRepository interface {
	Get(id string) (*model.Album, error)

	Save(album *model.Album) error

	Delete(id string) (bool, error)
}

func NewRepo(client *aero.Client) AlbumRepository {
	return &aerospike.AlbumRepository{
		client,
		"test",
		"albums",
	}
}
