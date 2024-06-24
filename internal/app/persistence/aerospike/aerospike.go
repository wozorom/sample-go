package aerospike

import (
	"errors"
	"fmt"
	aero "github.com/aerospike/aerospike-client-go/v7"
	"github.com/company/sample-go/internal/app/model"
)

const (
	Title  = "Title"
	Artist = "Artist"
	Price  = "Price"
)

type AlbumRepository struct {
	Client  *aero.Client
	Ns      string
	SetName string
}

func (repo *AlbumRepository) Delete(id string) (bool, error) {
	key, err := aero.NewKey(repo.Ns, repo.SetName, id)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Cant generate key for id=%s", id))
	}
	_, err = repo.Client.Delete(nil, key)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Cant delete value for id=%s, err=%s", id, err.Error()))
	}

	return true, nil
}

func (repo *AlbumRepository) Save(album *model.Album) error {
	key, err := aero.NewKey(repo.Ns, repo.SetName, album.ID)
	if err != nil {
		return errors.New(fmt.Sprintf("Cant generate key for id=%s", album.ID))
	}
	_, err = repo.Client.Operate(nil, key,
		aero.PutOp(&aero.Bin{Name: Title, Value: aero.NewStringValue(album.Title)}),
		aero.PutOp(&aero.Bin{Name: Artist, Value: aero.NewStringValue(album.Artist)}),
		aero.PutOp(&aero.Bin{Name: Price, Value: aero.NewFloatValue(album.Price)}))
	if err != nil {
		return errors.New(fmt.Sprintf("Cant save value for id=%s, err=%s", album.ID, err.Error()))
	}

	return nil
}

func (repo *AlbumRepository) Get(id string) (*model.Album, error) {
	key, err := aero.NewKey(repo.Ns, repo.SetName, id)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cant generate key for id=%s", id))
	}
	record, err := repo.Client.Get(nil, key)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cant load value for id=%s, err=%s", id, err.Error()))
	}
	price, ok := record.Bins[Price].(float64)
	if !ok {
		price = 0.0
	}
	return &model.Album{
		ID:     id,
		Title:  fmt.Sprintf("%v", record.Bins[Title]),
		Artist: fmt.Sprintf("%v", record.Bins[Artist]),
		Price:  price,
	}, nil
}
