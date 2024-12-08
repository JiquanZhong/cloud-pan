package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Service interface {
	Health() map[string]string
	Close() error
}

type ServiceImpl struct {
	DB *gorm.DB
}

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	schema     = os.Getenv("DB_SCHEMA")
	DBInstance *ServiceImpl
)

func New() Service {
	if DBInstance != nil {
		return DBInstance
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s", host, username, password, database, port, schema)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DBInstance = &ServiceImpl{
		DB: db,
	}
	return DBInstance
}

func (s *ServiceImpl) Health() map[string]string {
	stats := make(map[string]string)

	sqlDB, err := s.DB.DB()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("DB down: %v", err)
		log.Fatalf("DB down: %v", err)
		return stats
	}

	err = sqlDB.Ping()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("DB down: %v", err)
		log.Fatalf("DB down: %v", err)
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"

	dbStats := sqlDB.Stats()
	stats["open_connections"] = fmt.Sprintf("%d", dbStats.OpenConnections)
	stats["in_use"] = fmt.Sprintf("%d", dbStats.InUse)
	stats["idle"] = fmt.Sprintf("%d", dbStats.Idle)
	stats["wait_count"] = fmt.Sprintf("%d", dbStats.WaitCount)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = fmt.Sprintf("%d", dbStats.MaxIdleClosed)
	stats["max_lifetime_closed"] = fmt.Sprintf("%d", dbStats.MaxLifetimeClosed)

	return stats
}

func (s *ServiceImpl) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	log.Printf("Disconnected from database: %s", database)
	return sqlDB.Close()
}
