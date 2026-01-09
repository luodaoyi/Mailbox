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

var openDB = func(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

var countAdmins = func(db *gorm.DB) (int64, error) {
	var count int64
	if err := db.Model(&model.Admin{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

var hashPassword = func(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

var createAdmin = func(db *gorm.DB, admin *model.Admin) error {
	return db.Create(admin).Error
}

func Init(cfg *config.DatabaseConfig, adminCfg *config.AdminConfig) error {
	db, err := openDB(cfg)
	if err != nil {
		return err
	}

	return InitWithDB(db, adminCfg)
}

func InitWithDB(db *gorm.DB, adminCfg *config.AdminConfig) error {
	DB = db

	if err := db.AutoMigrate(&model.Domain{}, &model.Email{}, &model.Attachment{}, &model.Admin{}, &model.Mailbox{}); err != nil {
		return err
	}

	count, err := countAdmins(db)
	if err != nil {
		return err
	}

	if count == 0 {
		hashedPass, err := hashPassword(adminCfg.DefaultPass)
		if err != nil {
			return err
		}

		admin := &model.Admin{
			Username: adminCfg.DefaultUser,
			Password: string(hashedPass),
		}

		if err := createAdmin(db, admin); err != nil {
			return err
		}
	}

	return nil
}
