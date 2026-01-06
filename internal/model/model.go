package model

import "time"

type Domain struct {
	ID        uint      `gorm:"primaryKey"`
	Domain    string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Enabled   bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type Email struct {
	ID          uint         `gorm:"primaryKey"`
	FromAddr    string       `gorm:"type:varchar(255);index;not null"`
	ToAddr      string       `gorm:"type:varchar(255);index;not null"`
	Subject     string       `gorm:"type:text"`
	Body        string       `gorm:"type:text"`
	HtmlBody    string       `gorm:"type:text"`
	RawData     []byte       `gorm:"type:longblob"`
	ReceivedAt  time.Time    `gorm:"index;autoCreateTime"`
	Attachments []Attachment `gorm:"foreignKey:EmailID"`
}

type Attachment struct {
	ID          uint   `gorm:"primaryKey"`
	EmailID     uint   `gorm:"index;not null"`
	Filename    string `gorm:"type:varchar(255);not null"`
	ContentType string `gorm:"type:varchar(100)"`
	Size        int64
	Data        []byte `gorm:"type:longblob"`
}

type Admin struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
