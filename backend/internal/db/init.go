package db

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var dbHandle *gorm.DB

func InitDB(user, password, host, port, dbname string) *gorm.DB {
	// Create DSN according to go-sql-driver/mysql format.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		user, password, host, port, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&Streamer{}, &Reward{}, &Block{}, &ConfigStore{}, &User{}, &Request{}, &Moderator{}, &Command{})
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	dbHandle = db

	return db
}

func GetDB() *gorm.DB {
	if dbHandle == nil {
		log.Fatal("Database not initialized")
	}
	return dbHandle
}
