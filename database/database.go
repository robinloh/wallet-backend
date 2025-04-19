package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/robinloh/wallet-backend/models"
)

func ConnectDb() *gorm.DB {
	dataSource := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)

	gormOptions := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dataSource), gormOptions)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database : %v", err))
	}

	log.Println("database connected")
	db.Logger = logger.Default.LogMode(logger.Info)

	log.Println("running database migrations")
	err = db.AutoMigrate(&models.Account{})
	if err != nil {
		panic(fmt.Sprintf("Failed to automigrate in database. %v", err))
		return nil
	}

	return db
}
