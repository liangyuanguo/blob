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
	var rootRouter *gin.RouterGroup

	if config.Config.Http.Prefix != "" {
		rootRouter = router.Group(config.Config.Http.Prefix)
	} else {
		rootRouter = &router.RouterGroup
	}

	if config.Config.Mode == "s3" {
		controller.RegisterBlobController(rootRouter, service.NewS3BlobService())
	} else {
		controller.RegisterBlobController(rootRouter, service.NewLocalService())
	}

	// 启动服务器
	if err := router.Run(fmt.Sprintf("%s:%d", config.Config.Http.Host, config.Config.Http.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
