package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"gorm.io/gorm"
	"io"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/utils"
	model2 "liangyuanguo/aw/blob/pkg/model"
	"liangyuanguo/aw/blob/pkg/service"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
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

func (s *LocalService) UploadFile(fileHeader *multipart.FileHeader) (*model2.Blob, error) {
	if fileHeader.Size > config.Config.MaxUploadSize {
		return nil, fmt.Errorf("file size exceeds the limit of %d bytes", config.Config.MaxUploadSize)
	}

	fileID := utils.GenerateID()
	ext := filepath.Ext(fileHeader.Filename)
	fileName := fileID + ext
	filePath := filepath.Join(config.Config.Local.UploadDir, fileName)

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {

		}
	}(src)

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %v", err)
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {

		}
	}(dst)

	hash := md5.New()
	multiWriter := io.MultiWriter(dst, hash)

	if _, err := io.Copy(multiWriter, src); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	md5Sum := hex.EncodeToString(hash.Sum(nil))

	file := &model2.Blob{
		ID:          fileID,
		Name:        fileHeader.Filename,
		Size:        fileHeader.Size,
		Path:        filepath.Base(filePath),
		MD5:         md5Sum,
		ContentType: fileHeader.Header.Get("Content-Type"),
	}

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
