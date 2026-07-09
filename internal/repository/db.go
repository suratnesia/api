package repository

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"suratnesia/internal/config"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// InitMockDB is a testing helper to create a GORM db connected to mock sql
func InitMockDB(conn *sql.DB) (*gorm.DB, error) {
	return gorm.Open(postgres.New(postgres.Config{
		Conn: conn,
	}), &gorm.Config{})
}
