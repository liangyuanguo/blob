package utils

import (
	"github.com/bwmarrin/snowflake"
	"liangyuanguo/aw/blob/internal/config"
	"sync"
)

var (
	node *snowflake.Node
	once sync.Once
)

func InitSnowflake() error {

	var err error
	once.Do(func() {
		node, err = snowflake.NewNode(config.Config.Snowflake.WorkerID)
	})
	return err
}

func GenerateID() string {
	return node.Generate().String()
}
