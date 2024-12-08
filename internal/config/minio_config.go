package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type MinioConfig struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	StaticFilePath string
}

// 全局变量
var GlobalMinioConfig *MinioConfig

func init() {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// 读取 MinIO 配置
	useSSL, err := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))
	if err != nil {
		useSSL = false
	}

	// 初始化全局变量
	GlobalMinioConfig = &MinioConfig{
		Endpoint:       os.Getenv("MINIO_ENDPOINT"),
		AccessKey:      os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey:      os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:         useSSL,
		StaticFilePath: os.Getenv("STATIC_FILE_PATH"),
	}
}
