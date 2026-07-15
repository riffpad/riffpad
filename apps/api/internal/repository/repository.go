package repository

import (
	"gorm.io/gorm"
)

type Repositories struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{DB: db}
}
