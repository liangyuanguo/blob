package service

import (
	"github.com/gin-gonic/gin"
	"liangyuanguo/aw/blob/pkg/model"
)

type BlobService interface {
	Upload(uid string, fileName string, ctx *gin.Context) (*model.Blob, error)
	Download(uid string, c *gin.Context, fileID string) error
	Delete(uid string, fileID string) error
}
