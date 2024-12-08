package server

import (
	"cloud-server/internal/config"
	"cloud-server/internal/controller"
	"cloud-server/internal/database"
	"cloud-server/internal/model"
	"fmt"
	"net/http"

	"github.com/gin-contrib/static"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/health", s.healthHandler)

	// Initialize UserController
	userController := controller.NewUserController(s.db.(*database.ServiceImpl).DB)
	fileController := controller.NewFileController(s.minioClient, s.db.(*database.ServiceImpl).DB)

	// User routes
	userRoutes := r.Group("/users")
	{
		userRoutes.POST("", userController.CreateUser)
		userRoutes.GET("/:id", userController.GetUser)
		userRoutes.PUT("/:id", userController.UpdateUser)
		userRoutes.DELETE("/:id", userController.DeleteUser)
	}

	// File routes
	fileRoutes := r.Group("/files")
	{
		fileRoutes.POST("/upload", fileController.UploadFile)
		fileRoutes.GET("/download", fileController.DownloadFile)
		fileRoutes.DELETE("", fileController.DeleteFile)
		fileRoutes.GET("", fileController.ListFiles)
		fileRoutes.GET("/view", fileController.ViewFile)
		fileRoutes.GET("/getPresignedURL", fileController.GetPresignedURL)
	}

	// Serve static files
	r.Use(static.Serve("/static", static.LocalFile(config.GlobalMinioConfig.StaticFilePath, true)))

	return r
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) autoMigrateModels() {
	models := []interface{}{
		&model.User{},
		&model.File{}, // 添加文件模型
		// 在这里添加其他模型
	}

	for _, m := range models {
		if err := s.db.(*database.ServiceImpl).DB.AutoMigrate(m); err != nil {
			panic(fmt.Sprintf("Failed to migrate model %T: %v", m, err))
		}
	}
}
