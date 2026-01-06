package db

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"mailbox/internal/config"
	"mailbox/internal/model"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig, adminCfg *config.AdminConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	DB = db

	if err := db.AutoMigrate(&model.Domain{}, &model.Email{}, &model.Attachment{}, &model.Admin{}, &model.Mailbox{}); err != nil {
		return err
	}

	var count int64
	if err := db.Model(&model.Admin{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(adminCfg.DefaultPass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		admin := &model.Admin{
			Username: adminCfg.DefaultUser,
			Password: string(hashedPass),
		}

		if err := db.Create(admin).Error; err != nil {
			return err
		}
	}

	return nil
}
