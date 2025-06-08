package repository

import "liangyuanguo/aw/blob/pkg/model"

type IMetaStorage interface {
	Get(key string) (*model.Blob, error)
	Put(v *model.Blob) error
	Delete(key string) error
	Query(q map[string][2]string, offset, limit int) ([]*model.Blob, uint64, error)
}

var (
	MetaStorage IMetaStorage
)

func RegisterMetaStorage(m IMetaStorage) {
	MetaStorage = m
}
