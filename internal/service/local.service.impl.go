package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/repository"
	model2 "liangyuanguo/aw/blob/pkg/model"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalService struct {
	meta repository.IMetaStorage
}

func NewLocalService() BlobService {
	return &LocalService{
		meta: repository.MetaStorage,
	}
}

func (s *LocalService) Upload(uid string, fileID string, ctx *gin.Context) (*model2.Blob, error) {
	blobMeta, err := s.meta.Get(fileID)
	if err != nil || (blobMeta.AuthorId != uid && uid != "") {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 1. 生成存储路径和唯一ID
	ext := filepath.Ext(blobMeta.Name)
	filePath := filepath.Join(config.Config.Local.UploadDir, blobMeta.ID+ext)
	tmpFile := filepath.Join(config.Config.Local.UploadDir, blobMeta.ID+"-tmp"+ext)

	// 2. 打开目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %v", err)
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {

		}
	}(dst)

	// 3. 初始化流式处理器
	hash := md5.New()
	multiWriter := io.MultiWriter(dst, hash) // 同时写入文件和计算MD5

	// 4. 限制请求体大小并流式拷贝
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, config.Config.MaxUploadSize)
	written, err := io.Copy(multiWriter, ctx.Request.Body)
	if err != nil {
		_ = os.Remove(tmpFile) // 失败时清理文件
		if strings.Contains(err.Error(), "request body too large") {
			return nil, fmt.Errorf("file size exceeds the limit of %d bytes", config.Config.MaxUploadSize)
		}
		return nil, fmt.Errorf("stream copy failed: %v", err)
	} else {
		_ = os.Rename(tmpFile, filePath) // 成功时重命名临时文件
	}

	// 6. 构建文件元数据
	contentType := ctx.GetHeader("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	blobMeta.MD5 = hex.EncodeToString(hash.Sum(nil))
	blobMeta.Size = written
	blobMeta.Path = filepath.Base(filePath)
	blobMeta.UploadTime = time.Now()

	// 7. 保存元数据到数据库
	if err := s.meta.Put(blobMeta); err != nil {
		_ = os.Remove(filePath)
	}

	return blobMeta, nil
}

func (s *LocalService) Download(uid string, c *gin.Context, fileID string) error {

	var file *model2.Blob
	var err error

	if file, err = s.meta.Get(fileID); err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+file.Name)
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", file.Size))

	c.File(filepath.Join(config.Config.Local.UploadDir, file.Path))
	return nil
}

func (s *LocalService) Delete(uid string, fileID string) error {
	if file, err := s.meta.Get(fileID); err == nil && file != nil {
		if file.AuthorId != uid && uid != "" {
			return fmt.Errorf("permission denied")
		}

		if err := os.Remove(filepath.Join(config.Config.Local.UploadDir, file.Path)); err != nil {
			return fmt.Errorf("failed to delete file: %v", err)
		}
	} else {
		return fmt.Errorf("file not found: %v", err)
	}

	if err := s.meta.Delete(fileID); err != nil {
		return fmt.Errorf("failed to delete file info: %v", err)
	}

	return nil
}
