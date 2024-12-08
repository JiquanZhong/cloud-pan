package model

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	gorm.Model
	Username     string `gorm:"type:varchar(100);uniqueIndex" json:"username"`
	HashPassword string `gorm:"type:varchar(255)" json:"password"`
	Gender       string `gorm:"type:varchar(10)" json:"gender"`
	Email        string `gorm:"type:varchar(100);uniqueIndex" json:"email"`
	Phone        string `gorm:"type:varchar(20);uniqueIndex" json:"phone"`
	Nickname     string `gorm:"type:varchar(100)" json:"nickname"`
	Introduce    string `gorm:"type:text" json:"introduce"`
	Description  string `gorm:"type:text" json:"description"`
}

type File struct {
	gorm.Model
	FileName         string    `gorm:"type:varchar(255)" json:"file_name"`
	BucketName       string    `gorm:"type:varchar(255)" json:"bucket_name"`
	ObjectName       string    `gorm:"type:varchar(255)" json:"object_name"`
	FileSize         int64     `json:"file_size"`
	ContentType      string    `gorm:"type:varchar(100)" json:"content_type"`
	ETag             string    `gorm:"type:varchar(255)" json:"etag"`
	Size             int64     `json:"size"`
	LastModified     time.Time `json:"last_modified"`
	Location         string    `gorm:"type:varchar(255)" json:"location"`
	VersionID        string    `gorm:"type:varchar(255)" json:"version_id"`
	ExpirationRuleID string    `gorm:"type:varchar(255)" json:"expiration_rule_id"`
	OwnerID          uint      `json:"owner_id"` // 关联到用户
}
