package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/utils"
	model2 "liangyuanguo/aw/blob/pkg/model"
	"liangyuanguo/aw/blob/pkg/service"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LocalService struct {
	db *gorm.DB
}

func NewLocalService() service.BlobService {
	err := utils.InitDB()
	if err != nil {
		panic(err)
	}
	return &LocalService{db: utils.DB}
}

func (s *LocalService) UploadFile(fileName string, ctx *gin.Context) (*model2.Blob, error) {
	// 1. 创建目标文件路径
	fileID := utils.GenerateID()
	ext := filepath.Ext(fileName)
	filePath := filepath.Join(config.Config.Local.UploadDir, fileID+ext)

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
		_ = os.Remove(filePath) // 失败时清理文件
		if strings.Contains(err.Error(), "request body too large") {
			return nil, fmt.Errorf("file size exceeds the limit of %d bytes", config.Config.MaxUploadSize)
		}
		return nil, fmt.Errorf("stream copy failed: %v", err)
	}

	// 5. 获取Content-Type（优先用Header，其次根据扩展名推断）
	contentType := ctx.GetHeader("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// 6. 构建文件记录
	file := &model2.Blob{
		ID:          fileID,
		Name:        fileName,
		Size:        written, // 实际写入的字节数
		Path:        filepath.Base(filePath),
		MD5:         hex.EncodeToString(hash.Sum(nil)),
		ContentType: contentType,
	}

	// 7. 保存元数据到数据库
	if err := s.db.Create(file).Error; err != nil {
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file info: %v", err)
	}

	return file, nil
}

func (s *LocalService) DownloadFile(c *gin.Context, fileID string) error {

	var file model2.Blob
	if err := s.db.First(&file, "id = ?", fileID).Error; err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+file.Name)
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", file.Size))

	c.File(filepath.Join(config.Config.Local.UploadDir, file.Path))
	return nil
}
func (s *LocalService) GetFileList(kw string, offset, limit int) ([]model2.Blob, error) {
	var files []model2.Blob
	query := s.db.Offset(offset).Limit(limit).Order("upload_time desc")

	if kw != "" {
		// Use parameterized query with wildcards
		query = query.Where("name LIKE ?", fmt.Sprintf("%s%%", escapeLikeWildcards(kw)))
	}

	if err := query.Find(&files).Error; err != nil {
		return nil, fmt.Errorf("failed to get file list: %w", err)
	}
	return files, nil
}

// Helper function to escape LIKE wildcards if needed
func escapeLikeWildcards(s string) string {
	// Escape % and _ characters if they should be treated literally
	return strings.ReplaceAll(strings.ReplaceAll(s, "%", "\\%"), "_", "\\_")
}

func (s *LocalService) DeleteFile(fileID string) error {
	var file model2.Blob
	if err := s.db.First(&file, "id = ?", fileID).Error; err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	if err := os.Remove(filepath.Join(config.Config.Local.UploadDir, file.Path)); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	if err := s.db.Delete(&file).Error; err != nil {
		return fmt.Errorf("failed to delete file info: %v", err)
	}

	return nil
}
