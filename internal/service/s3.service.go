package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/utils"
	model2 "liangyuanguo/aw/blob/pkg/model"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Service struct {
	client *minio.Client
	bucket string
	db     *gorm.DB
}

func NewS3BlobService() *S3Service {
	// 初始化 MinIO 客户端
	endpoint := config.Config.S3.Endpoint
	accessKey := config.Config.S3.AK
	secretKey := config.Config.S3.SK
	useSSL := config.Config.S3.UseSSL

	// 初始化 MinIO 客户端对象
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	// 检查存储桶是否存在，不存在则创建
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, config.Config.S3.Bucket)
	if err != nil {
		panic(err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, config.Config.S3.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			panic(err)
		}
		log.Printf("存储桶 %s 创建成功", config.Config.S3.Bucket)
	}

	err = utils.InitDB()
	if err != nil {
		panic(err)
	}
	return &S3Service{
		client: minioClient,
		bucket: config.Config.S3.Bucket,
		db:     utils.DB, // 假设你有一个获取数据库连接的方法
	}
}

func (s *S3Service) UploadFile(fileHeader *multipart.FileHeader) (*model2.Blob, error) {
	if fileHeader.Size > config.Config.MaxUploadSize {
		return nil, fmt.Errorf("文件大小超过限制 %d 字节", config.Config.MaxUploadSize)
	}

	fileID := utils.GenerateID()
	ext := filepath.Ext(fileHeader.Filename)
	fileName := fileID + ext
	minioKey := filepath.Join(config.Config.S3.Prefix, fileName)

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("无法打开上传的文件: %v", err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	// 计算 MD5 哈希
	hash := md5.New()
	tee := io.TeeReader(file, hash)

	// 上传文件到 MinIO
	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		minioKey,
		tee,
		fileHeader.Size,
		minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("上传到 MinIO 失败: %v", err)
	}

	md5Sum := hex.EncodeToString(hash.Sum(nil))

	fileMeta := &model2.Blob{
		ID:          fileID,
		Name:        fileHeader.Filename,
		Size:        fileHeader.Size,
		MD5:         md5Sum,
		Path:        minioKey,
		ContentType: fileHeader.Header.Get("Content-Type"),
		UploadTime:  time.Now(),
	}

	if err := s.db.Create(fileMeta).Error; err != nil {
		return nil, fmt.Errorf("保存文件信息失败: %v", err)
	}

	return fileMeta, nil
}

func (s *S3Service) DownloadFile(c *gin.Context, fileID string) error {
	var fileMeta model2.Blob
	if err := s.db.First(&fileMeta, "id = ?", fileID).Error; err != nil {
		return fmt.Errorf("文件未找到: %v", err)
	}

	// 生成预签名 URL
	presignedURL, err := s.client.PresignedGetObject(
		context.Background(),
		s.bucket,
		fileMeta.Path,
		15*time.Minute, // URL 有效期
		nil,
	)
	if err != nil {
		return fmt.Errorf("生成下载 URL 失败: %v", err)
	}

	c.Redirect(http.StatusFound, presignedURL.String())
	return nil
}

func (s *S3Service) GetFileList(kw string, offset, limit int) ([]model2.Blob, error) {
	var files []model2.Blob
	query := s.db.Offset(offset).Limit(limit).Order("upload_time desc")

	if kw != "" {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%%s%%", kw))
	}

	if err := query.Find(&files).Error; err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %w", err)
	}
	return files, nil
}

func (s *S3Service) DeleteFile(fileID string) error {
	var fileMeta model2.Blob
	if err := s.db.First(&fileMeta, "id = ?", fileID).Error; err != nil {
		return fmt.Errorf("文件未找到: %v", err)
	}

	// 从 MinIO 删除文件
	err := s.client.RemoveObject(
		context.Background(),
		s.bucket,
		fileMeta.Path,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("从 MinIO 删除文件失败: %v", err)
	}

	// 从数据库删除记录
	if err := s.db.Delete(&fileMeta).Error; err != nil {
		return fmt.Errorf("从数据库删除文件记录失败: %v", err)
	}

	return nil
}
