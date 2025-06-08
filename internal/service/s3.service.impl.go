package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/repository"
	model2 "liangyuanguo/aw/blob/pkg/model"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Service struct {
	client *minio.Client
	bucket string
	meta   repository.IMetaStorage
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

	return &S3Service{
		client: minioClient,
		bucket: config.Config.S3.Bucket,
		meta:   repository.MetaStorage,
	}
}

func (s *S3Service) Upload(uid string, fileID string, ctx *gin.Context) (*model2.Blob, error) {
	blobMeta, err := s.meta.Get(fileID)
	if err != nil || (blobMeta.AuthorId != uid && uid != "") {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 1. 生成存储路径和唯一ID
	ext := filepath.Ext(blobMeta.Name)
	minioKey := filepath.Join(config.Config.S3.Prefix, blobMeta.ID+ext)

	// 2. 初始化流式处理器
	hash := md5.New()
	pr, pw := io.Pipe() // 创建管道用于流式传输
	defer func(pr *io.PipeReader) {
		err := pr.Close()
		if err != nil {

		}
	}(pr)

	// 3. 限制请求体大小
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, config.Config.MaxUploadSize)

	// 4. 并行处理：流式读取请求体并计算MD5
	var uploadErr error
	var uploadedSize int64
	go func() {
		defer func(pw *io.PipeWriter) {
			err := pw.Close()
			if err != nil {

			}
		}(pw)
		// 使用TeeReader同时写入hash和管道
		tee := io.TeeReader(ctx.Request.Body, hash)
		uploadedSize, uploadErr = io.Copy(pw, tee)
	}()

	// 5. 上传到MinIO/S3（流式）
	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		minioKey,
		pr, // 注意：这里使用管道的读取端
		-1, // 未知大小，设为-1
		minio.PutObjectOptions{
			ContentType: ctx.GetHeader("Content-Type"),
		},
	)

	// 6. 错误处理
	if err != nil || uploadErr != nil {
		if err == nil {
			err = uploadErr
		}
		return nil, fmt.Errorf("上传失败: %v", err)
	}

	// 7. 构建文件元数据
	contentType := ctx.GetHeader("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	blobMeta.MD5 = hex.EncodeToString(hash.Sum(nil))
	blobMeta.Size = uploadedSize
	blobMeta.Path = minioKey
	blobMeta.UploadTime = time.Now()

	// 8. 保存元数据到数据库
	if err := s.meta.Put(blobMeta); err != nil {
		// 回滚：尝试删除已上传的文件
		_ = s.client.RemoveObject(context.Background(), s.bucket, minioKey, minio.RemoveObjectOptions{})
		return nil, fmt.Errorf("保存文件信息失败: %v", err)
	}

	return blobMeta, nil
}
func (s *S3Service) Download(uid string, c *gin.Context, fileID string) error {
	var fileMeta *model2.Blob
	var err error

	if fileMeta, err = s.meta.Get(fileID); err != nil {
		return fmt.Errorf("file not found: %v", err)
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

func (s *S3Service) Delete(uid string, fileID string) error {
	if fileMeta, err := s.meta.Get(fileID); err == nil && fileMeta != nil {
		if fileMeta.AuthorId != uid && uid != "" {
			return fmt.Errorf("permission denied")
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

	} else {
		return fmt.Errorf("file not found: %v", err)
	}

	if err := s.meta.Delete(fileID); err != nil {
		return fmt.Errorf("failed to delete file info: %v", err)
	}
	return nil
}
