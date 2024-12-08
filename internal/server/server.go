package server

import (
	minioc "cloud-server/internal/minio"
	"fmt"
	"github.com/minio/minio-go/v7"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"cloud-server/internal/database"
)

type Server struct {
	port        int
	db          database.Service
	minioClient *minio.Client
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port:        port,
		db:          database.New(),
		minioClient: minioc.New(),
	}

	// 自动迁移所有模型
	NewServer.autoMigrateModels()

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
