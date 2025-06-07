package service

import (
	"github.com/gin-gonic/gin"
	"liangyuanguo/aw/blob/pkg/model"
	"mime/multipart"
)

type BlobService interface {
	UploadFile(fileHeader *multipart.FileHeader) (*model.Blob, error)
	DownloadFile(c *gin.Context, fileID string) error
	GetFileList(kw string, offset, limit int) ([]model.Blob, error)
	DeleteFile(fileID string) error
}
