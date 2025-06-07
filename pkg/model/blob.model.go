package model

import "time"

type Blob struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Size        int64     `gorm:"not null" json:"size"`
	Path        string    `gorm:"not null" json:"path"`
	UploadTime  time.Time `gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP" json:"uploadTime"`
	MD5         string    `json:"md5,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
}
