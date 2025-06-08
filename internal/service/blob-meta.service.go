package service

import (
	"fmt"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/dto"
	"liangyuanguo/aw/blob/internal/repository"
	"liangyuanguo/aw/blob/internal/utils"
	model2 "liangyuanguo/aw/blob/pkg/model"
	"path/filepath"
)

type BlobMetaService struct {
	meta repository.IMetaStorage
}

func NewBlobMetaService() *BlobMetaService {
	return &BlobMetaService{
		meta: repository.MetaStorage,
	}
}

func (s *BlobMetaService) Query(uid string, q map[string][2]string, offset, limit int) ([]*model2.Blob, uint64, error) {
	return s.meta.Query(q, offset, limit)
}

func (s *BlobMetaService) Put(uid string, req *dto.UpdateBlobReq) (*model2.Blob, error) {
	var blob *model2.Blob
	var err error

	if req.ID == "" {
		if req.Name == "" {
			return nil, fmt.Errorf("name is required")
		}

		fileID := utils.GenerateID()
		ext := filepath.Ext(req.Name)
		filePath := filepath.Join(config.Config.Local.UploadDir, fileID+ext)

		blob = &model2.Blob{
			ID:       fileID,
			Path:     filePath,
			AuthorId: uid,
		}
		req.Apply(blob)
		err = s.meta.Put(blob)
		if err != nil {
			return nil, err
		}
	} else {
		blob, err = s.meta.Get(req.ID)
		if err != nil {
			return nil, err
		}
		if blob.AuthorId != uid && uid != "" {
			return nil, fmt.Errorf("you are not the owner of this blob")
		}
		req.Apply(blob)

		err := s.meta.Put(blob)
		if err != nil {
			return nil, err
		}
	}

	blob, err = s.meta.Get(blob.ID)
	if err != nil {
		return nil, fmt.Errorf("blob not found")
	}
	return blob, nil
}
