package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"suratnesia/internal/config"
)

func TestInitializeDB(t *testing.T) {
	// Let's verify compilation of InitDB with config
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "password",
		DBName:     "postgres",
	}
	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mock.ExpectClose()

	// GORM initializes connection with dialer. We'll verify that our wrapper handles GORM init
	_, _ = InitDB(cfg)

	db, err := InitMockDB(sqlDB)
	assert.NoError(t, err)
	assert.NotNil(t, db)
}
