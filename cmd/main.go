package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/controller"
	"liangyuanguo/aw/blob/internal/service"
	"liangyuanguo/aw/blob/internal/utils"
	"log"
)

func main() {
	// 初始化雪花算法
	if err := utils.InitSnowflake(); err != nil {
		log.Fatalf("Failed to initialize snowflake: %v", err)
	}

	// 创建Gin路由
	router := gin.Default()
	router.Static(config.Config.Http.StaticPrefix, config.Config.Http.StaticDir)

	var rootRouter *gin.RouterGroup

	if config.Config.Http.Prefix != "" {
		rootRouter = router.Group(config.Config.Http.Prefix)
	} else {
		rootRouter = &router.RouterGroup
	}

	switch config.Config.Mode {
	case "local":
		controller.RegisterBlobController(rootRouter, service.NewLocalService())
	case "s3":
		controller.RegisterBlobController(rootRouter, service.NewS3BlobService())
	case "none":
	default:
		log.Fatalf("Invalid mode: %s", config.Config.Mode)
	}

	// 启动服务器
	if err := router.Run(fmt.Sprintf("%s:%d", config.Config.Http.Host, config.Config.Http.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
