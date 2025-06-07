package controller

import (
	"liangyuanguo/aw/blob/internal/middleware"
	"liangyuanguo/aw/blob/internal/utils"
	"liangyuanguo/aw/blob/pkg/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type BlobController struct {
	blobService service.BlobService
}

func NewBlobController(fileService service.BlobService) *BlobController {
	return &BlobController{blobService: fileService}
}

func (c *BlobController) Upload(ctx *gin.Context) {
	fileName := ctx.Param("id")
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file name is required"})
		return
	}
	
	uploadedFile, err := c.blobService.UploadFile(fileName, ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":   uploadedFile.ID,
		"name": uploadedFile.Name,
		"size": uploadedFile.Size,
		"path": uploadedFile.Path,
	})
}

func (c *BlobController) Download(ctx *gin.Context) {
	fileID := ctx.Param("id")
	files := strings.Split(fileID, ".")
	if len(files) > 1 {
		fileID = files[0]
	}

	if err := c.blobService.DownloadFile(ctx, fileID); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}
}

func (c *BlobController) ListFiles(ctx *gin.Context) {
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	kw := ctx.Query("kw")

	files, err := c.blobService.GetFileList(kw, offset, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": files})
}

func (c *BlobController) Delete(ctx *gin.Context) {
	fileID := ctx.Param("id")
	files := strings.Split(fileID, ".")
	if len(files) > 1 {
		fileID = files[0]
	}

	if err := c.blobService.DeleteFile(fileID); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}
}

var blobController *BlobController

func RegisterBlobController(router *gin.RouterGroup, fileService service.BlobService) {
	if blobController != nil {
		return
	}

	blobController := NewBlobController(fileService)

	// 公共路由（不需要认证）
	public := router.Group("/blobs")
	{
		public.GET("/:id", blobController.Download)
	}

	// 需要认证的路由
	authRequired := router.Group("/blobs")
	authRequired.Use(middleware.JWTAuthMiddleware(utils.NewJWTUtil()))
	{
		authRequired.POST("/:id", blobController.Upload)
		authRequired.GET("", blobController.ListFiles)
		authRequired.DELETE("/:id", blobController.Delete)
	}
}
