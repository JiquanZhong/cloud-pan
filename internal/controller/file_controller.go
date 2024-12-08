package controller

import (
	"cloud-server/internal/model"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"time"
)

type FileController struct {
	client *minio.Client
	DB     *gorm.DB
}

func NewFileController(client *minio.Client, DB *gorm.DB) *FileController {
	return &FileController{client: client, DB: DB}
}

// 上传文件
func (fc *FileController) UploadFile(c *gin.Context) {
	bucketName := c.PostForm("bucketName")
	objectName := c.PostForm("objectName")
	filePath := c.PostForm("filePath")
	fileName := filepath.Base(filePath)

	var existingFilie model.File
	if err := fc.DB.Where("bucket_name = ? AND object_name = ?", bucketName, objectName).First(&existingFilie).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file already exists"})
		return
	}

	ctx := context.Background()

	// 确保存储桶存在
	err := fc.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := fc.client.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("Bucket %s already exists", bucketName)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create bucket: %v", err)})
			return
		}
	}

	// Begin a database transaction
	tx := fc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Upload the file
	uploadInfo, err := fc.client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to upload file: %v", err)})
		return
	}

	// Store file metadata in the database
	file := model.File{
		FileName:         fileName,
		BucketName:       bucketName,
		ObjectName:       objectName,
		ETag:             uploadInfo.ETag,
		Size:             uploadInfo.Size,
		Location:         uploadInfo.Location,
		VersionID:        uploadInfo.VersionID,
		ExpirationRuleID: uploadInfo.ExpirationRuleID,
	}
	if err := tx.Create(&file).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file metadata: %v", err)})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	log.Printf("Successfully uploaded %s to %s/%s", filePath, bucketName, objectName)
	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

// 下载文件
func (fc *FileController) DownloadFile(c *gin.Context) {
	bucketName := c.Query("bucketName")
	objectName := c.Query("objectName")
	downloadPath := c.Query("downloadPath")

	if fc.DB.Where("bucket_name = ? AND object_name = ?", bucketName, objectName).First(&model.File{}).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not found"})
		return
	}

	ctx := context.Background()

	// Download the file
	err := fc.client.FGetObject(ctx, bucketName, objectName, downloadPath, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to download file: %v", err)})
		return
	}

	log.Printf("Successfully downloaded %s to %s", objectName, downloadPath)
	c.JSON(http.StatusOK, gin.H{"message": "File downloaded successfully"})
}

// 删除文件
func (fc *FileController) DeleteFile(c *gin.Context) {
	bucketName := c.Query("bucketName")
	objectName := c.Query("objectName")

	ctx := context.Background()

	tx := fc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	result := fc.DB.Delete(&model.File{}, "bucket_name = ? AND object_name = ?", bucketName, objectName)
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete file metadata: %v", result.Error)})
		return
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not found"})
		return
	}

	// 删除文件
	err := fc.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete file: %v", err)})
		return
	}

	log.Printf("Successfully deleted %s from %s", objectName, bucketName)
	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// ListFiles lists the files in a specified bucket and folder
func (fc *FileController) ListFiles(c *gin.Context) {
	bucketName := c.Query("bucketName")
	prefix := c.Query("prefix") // The folder path to list files from

	ctx := context.Background()

	// Initialize a channel to receive object info
	objectCh := fc.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var files []model.File
	for object := range objectCh {
		if object.Err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list files: %v", object.Err)})
			return
		}

		files = append(files, model.File{
			FileName:     filepath.Base(object.Key),
			BucketName:   bucketName,
			ObjectName:   object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
		})
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (fc *FileController) GetPresignedURL(c *gin.Context) {

	presignedURL, err := fc.generatePresignedURL(c)
	if err != nil {
		return
	}

	// Return the presigned URL
	c.JSON(http.StatusOK, gin.H{"presignedURL": presignedURL})
}

func (fc *FileController) ViewFile(c *gin.Context) {
	bucketName := c.Query("bucketName")
	objectName := c.Query("objectName")

	// Fetch the file name from the database
	var file model.File
	if err := fc.DB.Where("bucket_name = ? AND object_name = ?", bucketName, objectName).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Generate the presigned URL
	presignedURL, err := fc.generatePresignedURL(c)
	if err != nil {
		return
	}
	// Construct the preview URL
	previewURL := fmt.Sprintf("%s&fullfilename=%s", presignedURL, url.QueryEscape(file.FileName))

	// Encode the preview URL
	encodedURL := base64.StdEncoding.EncodeToString([]byte(previewURL))
	kkFileViewURL := fmt.Sprintf("http://127.0.0.1:8012/onlinePreview?url=%s", url.QueryEscape(encodedURL))

	// Return the preview URL
	c.JSON(http.StatusOK, gin.H{"previewURL": kkFileViewURL})
}

func (fc *FileController) generatePresignedURL(c *gin.Context) (string, error) {

	bucketName := c.Query("bucketName")
	objectName := c.Query("objectName")

	if fc.DB.Where("bucket_name = ? AND object_name = ?", bucketName, objectName).First(&model.File{}).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not found"})
		return "", fmt.Errorf("file not found")
	}

	ctx := context.Background()

	// Generate a presigned URL for the file
	reqParams := make(url.Values)
	presignedURL, err := fc.client.PresignedGetObject(ctx, bucketName, objectName, time.Duration(1)*time.Hour, reqParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate presigned URL: %v", err)})
		return "", err
	}
	return presignedURL.String(), nil
}
