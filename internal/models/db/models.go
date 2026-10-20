package models

import (
	"time"
)

type PayloadStatuses struct {
	ID        uint  `gorm:"primaryKey;not null;autoIncrement"`
	PayloadId uint  `gorm:"not null"`
	ServiceId int32 `gorm:"not null"`
	SourceId  int32
	StatusId  int32     `gorm:"not null"`
	StatusMsg string    `gorm:"type:varchar"`
	Date      time.Time `gorm:"primaryKey;not null"`
	CreatedAt time.Time `gorm:"not null"`
	Payload   Payloads
	Service   Services
	Source    Sources
	Status    Statuses
}

type Payloads struct {
	Id          uint      `gorm:"primaryKey;not null;autoIncrement"`
	RequestId   string    `json:"request_id" gorm:"not null;type:varchar;unique"`
	Account     string    `json:"account" gorm:"type:varchar"`
	InventoryId string    `json:"inventory_id" gorm:"type:varchar"`
	SystemId    string    `json:"system_id" gorm:"type:varchar"`
	CreatedAt   time.Time `gorm:"not null"`
	OrgId       string    `json:"org_id" gorm:"type:varchar"`
}

type Services struct {
	Id   int32  `gorm:"primaryKey;not null;autoIncrement"`
	Name string `gorm:"not null;type:varchar"`
}

type Sources struct {
	Id   int32  `gorm:"primaryKey;not null;autoIncrement"`
	Name string `gorm:"not null;type:varchar"`
}

type Statuses struct {
	Id   int32  `gorm:"primaryKey;not null;autoIncrement"`
	Name string `gorm:"not null;type:varchar"`
}
