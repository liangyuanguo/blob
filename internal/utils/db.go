package utils

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/pkg/model"
)

var DB *gorm.DB

func InitDB() error {

	if DB != nil {
		return nil
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Config.Db.Username,
		config.Config.Db.Password,
		config.Config.Db.Host,
		config.Config.Db.Port,
		config.Config.Db.DbName,
	)

	var err error

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 自动迁移
	if err := DB.AutoMigrate(&model.Blob{}); err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	return nil
}
