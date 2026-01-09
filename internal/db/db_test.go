package db

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"mailbox/internal/config"
	"mailbox/internal/model"
)

func openTestDB(t *testing.T) *gorm.DB {
	dsn := "file:db_test_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}
	return db
}

func TestInitWithDBCreatesAdmin(t *testing.T) {
	testDB := openTestDB(t)
	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "admin123"}

	if err := InitWithDB(testDB, adminCfg); err != nil {
		t.Fatalf("init with db failed: %v", err)
	}

	var admin model.Admin
	if err := DB.First(&admin).Error; err != nil {
		t.Fatalf("admin not created: %v", err)
	}
	if admin.Username != "admin" {
		t.Fatalf("unexpected admin username: %s", admin.Username)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("admin123")); err != nil {
		t.Fatalf("admin password not hashed or mismatch")
	}
}

func TestInitWithDBSkipsAdminCreate(t *testing.T) {
	testDB := openTestDB(t)
	if err := testDB.AutoMigrate(&model.Admin{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	if err := testDB.Create(&model.Admin{Username: "exists", Password: "hash"}).Error; err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}

	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "admin123"}
	if err := InitWithDB(testDB, adminCfg); err != nil {
		t.Fatalf("init with db failed: %v", err)
	}

	var count int64
	if err := testDB.Model(&model.Admin{}).Count(&count).Error; err != nil {
		t.Fatalf("count admins failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected admin count: %d", count)
	}
}

func TestInitUsesOpenDB(t *testing.T) {
	orig := openDB
	openDB = func(cfg *config.DatabaseConfig) (*gorm.DB, error) {
		dsn := "file:db_open_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?mode=memory&cache=shared"
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	}
	t.Cleanup(func() { openDB = orig })

	cfg := &config.DatabaseConfig{Host: "x", Port: 1, User: "u", Password: "p", Name: "n"}
	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "pass"}
	if err := Init(cfg, adminCfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	var count int64
	if err := DB.Model(&model.Admin{}).Count(&count).Error; err != nil {
		t.Fatalf("count admin failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected admin count: %d", count)
	}
}

func TestInitOpenDBError(t *testing.T) {
	orig := openDB
	openDB = func(cfg *config.DatabaseConfig) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}
	t.Cleanup(func() { openDB = orig })

	cfg := &config.DatabaseConfig{Host: "x", Port: 1, User: "u", Password: "p", Name: "n"}
	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "pass"}
	if err := Init(cfg, adminCfg); err == nil {
		t.Fatalf("expected open error")
	}
}

func TestInitWithDBAutoMigrateError(t *testing.T) {
	testDB := openTestDB(t)
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	_ = sqlDB.Close()

	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "pass"}
	if err := InitWithDB(testDB, adminCfg); err == nil {
		t.Fatalf("expected migrate error")
	}
}

func TestInitWithDBCountError(t *testing.T) {
	testDB := openTestDB(t)
	orig := countAdmins
	countAdmins = func(db *gorm.DB) (int64, error) {
		return 0, errors.New("count failed")
	}
	t.Cleanup(func() { countAdmins = orig })

	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "pass"}
	if err := InitWithDB(testDB, adminCfg); err == nil {
		t.Fatalf("expected count error")
	}
}

func TestInitWithDBHashError(t *testing.T) {
	testDB := openTestDB(t)
	origCount := countAdmins
	origHash := hashPassword
	countAdmins = func(db *gorm.DB) (int64, error) {
		return 0, nil
	}
	hashPassword = func(password string) ([]byte, error) {
		return nil, errors.New("hash failed")
	}
	t.Cleanup(func() {
		countAdmins = origCount
		hashPassword = origHash
	})

	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "pass"}
	if err := InitWithDB(testDB, adminCfg); err == nil {
		t.Fatalf("expected hash error")
	}
}

func TestInitWithDBCreateError(t *testing.T) {
	testDB := openTestDB(t)
	origCount := countAdmins
	origHash := hashPassword
	origCreate := createAdmin
	countAdmins = func(db *gorm.DB) (int64, error) {
		return 0, nil
	}
	hashPassword = func(password string) ([]byte, error) {
		return []byte("hash"), nil
	}
	createAdmin = func(db *gorm.DB, admin *model.Admin) error {
		return errors.New("create failed")
	}
	t.Cleanup(func() {
		countAdmins = origCount
		hashPassword = origHash
		createAdmin = origCreate
	})

	adminCfg := &config.AdminConfig{DefaultUser: "admin", DefaultPass: "pass"}
	if err := InitWithDB(testDB, adminCfg); err == nil {
		t.Fatalf("expected create error")
	}
}
