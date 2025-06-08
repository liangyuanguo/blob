package controller

import (
	"fmt"
	"liangyuanguo/aw/blob/internal/dto"
	"liangyuanguo/aw/blob/internal/middleware"
	"liangyuanguo/aw/blob/internal/service"
	"liangyuanguo/aw/blob/internal/utils"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type BlobController struct {
	blobService     service.BlobService
	blobMetaService *service.BlobMetaService
}

func NewBlobController(fileService service.BlobService) *BlobController {
	return &BlobController{blobService: fileService, blobMetaService: service.NewBlobMetaService()}
}

func (c *BlobController) Upload(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	fileName := ctx.Param("id")
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file name is required"})
		return
	}

	uploadedFile, err := c.blobService.Upload(uid.(string), fileName, ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, uploadedFile)
}

func (c *BlobController) Download(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	fileID := ctx.Param("id")
	files := strings.Split(fileID, ".")
	if len(files) > 1 {
		fileID = files[0]
	}

	if err := c.blobService.Download(uid.(string), ctx, fileID); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}
}

func (c *BlobController) Query(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	qMap := ctx.QueryMap("q")

	q := map[string][2]string{}
	for k, v := range qMap {
		idx := strings.Index(v, "=")
		if idx <= 0 {
			idx = 0
		}
		q[k] = [2]string{
			v[0:idx],
			v[idx+1:],
		}
	}

	fmt.Println(q)

	files, total, err := c.blobMetaService.Query(uid.(string), q, offset, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": files, "total": total, "offset": offset, "limit": limit})
}

func (c *BlobController) Put(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")

	fileID := ctx.Param("id")
	files := strings.Split(fileID, ".")
	if len(files) > 1 {
		fileID = files[0]
	}

	contentType := ctx.GetHeader("Content-Type")
	if contentType == "" {
		if len(files) > 1 {
			contentType = mime.TypeByExtension(files[1])
		} else {
			contentType = "application/octet-stream"
		}
	}

	dto := &dto.UpdateBlobReq{}
	err := ctx.ShouldBindBodyWithJSON(dto)
	if err != nil {
		return
	}

	if files[0] != "_" && files[0] != "" {
		dto.ID = files[0]
	}
	blob, err := c.blobMetaService.Put(uid.(string), dto)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, blob)
}

func (c *BlobController) Delete(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	fileID := ctx.Param("id")
	files := strings.Split(fileID, ".")
	if len(files) > 1 {
		fileID = files[0]
	}

	if err := c.blobService.Delete(uid.(string), fileID); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}
}

var blobController *BlobController

func RegisterBlobController(router *gin.RouterGroup, fileService service.BlobService) {
	if blobController != nil {
		return
	}

	blobController := NewBlobController(fileService)

	//// 公共路由（不需要认证）
	//public := router.Group("/blobs")
	//{
	//	public.GET("/:id", blobController.Download)
	//}

	// 需要认证的路由
	authRequired := router.Group("/blobs")
	authRequired.Use(middleware.JWTAuthMiddleware(utils.NewJWTUtil()))
	{
		authRequired.GET("/:id", blobController.Download)
		authRequired.POST("/:id", blobController.Upload)
		authRequired.PUT("/:id", blobController.Put)
		authRequired.GET("", blobController.Query)
		authRequired.DELETE("/:id", blobController.Delete)
	}
}
