package config

import (
	"flag"
	"gopkg.in/yaml.v3"
	"os"
)

type config struct {
	MaxUploadSize int64            `yaml:"maxUploadSize"`
	Mode          string           `yaml:"mode"`
	Local         *LocalConfig     `yaml:"local"`
	S3            *S3Config        `yaml:"s3"`
	Db            *DbConfig        `yaml:"db"`
	Jwt           *JwtConfig       `yaml:"jwt"`
	Snowflake     *SnowflakeConfig `yaml:"snowflake"`
	Http          *HttpConfig      `yaml:"http"`
}

type LocalConfig struct {
	UploadDir string `yaml:"uploadDir"`
}

type DbConfig struct {
	BleveDir string `yaml:"bleveDir"`
}

type S3Config struct {
	AK       string `yaml:"ak"`
	SK       string `yaml:"sk"`
	UseSSL   bool   `yaml:"useSSL"`
	Region   string `yaml:"region"`
	Bucket   string `yaml:"bucket"`
	Endpoint string `yaml:"endpoint"`

	Prefix string `yaml:"prefix"`
}

type JwtConfig struct {
	Secret string `yaml:"secret"`
	Expire int64  `yaml:"expire"`
}

type SnowflakeConfig struct {
	WorkerID     int64 `yaml:"workerID"`
	DataCenterID int64 `yaml:"dataCenterID"`
}

type HttpConfig struct {
	Port         int    `yaml:"port"`
	Host         string `yaml:"host"`
	Prefix       string `yaml:"prefix"`
	StaticDir    string `yaml:"staticDir"`
	StaticPrefix string `yaml:"staticPrefix"`
}

var Config *config

func init() {
	Config = &config{}

	var configPath string
	// 获取配置文件路径的优先级：
	// 1. 命令行参数
	// 2. 环境变量 config
	// 3. 默认值 ./config.yaml

	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG")
		if configPath == "" {
			configPath = "./config.yaml"
		}
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	// 解析YAML
	if err := yaml.Unmarshal(data, Config); err != nil {
		panic(err)
	}
}
