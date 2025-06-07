package service

import (
	"github.com/gin-gonic/gin"
	"liangyuanguo/aw/blob/pkg/model"
)

type BlobService interface {
	UploadFile(fileName string, ctx *gin.Context) (*model.Blob, error)
	DownloadFile(c *gin.Context, fileID string) error
	GetFileList(kw string, offset, limit int) ([]model.Blob, error)
	DeleteFile(fileID string) error
}
