package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
)

type PayloadStatusJson struct {
	PayloadId uint   `json:"payload_id"`
	ServiceId int32  `json:"service_id"`
	SourceId  int32  `json:"source_id"`
	StatusId  int32  `json:"status_id"`
	StatusMsg string `json:"status_msg"`
	Date      string `json:"date"`
}

type Fields struct {
	Services        []models.Services   `json:"services"`
	Sources         []models.Sources    `json:"sources"`
	Statuses        []models.Statuses   `json:"statuses"`
	Payloads        []models.Payloads   `json:"payloads"`
	PayloadStatuses []PayloadStatusJson `json:"payload_statuses"`
}

func main() {
	l.InitLogger()

	cfg := config.Get()

	db.DbConnect(cfg)

	jsonFile, err := os.Open("tools/db-seeder/seed.json")
	if err != nil {
		l.Log.Fatal(err)
	}
	l.Log.Info("seed.json opened")
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var fields Fields

	json.Unmarshal(byteValue, &fields)

	fmt.Println("Seeding Services Table")
	for i := 0; i < len(fields.Services); i++ {
		l.Log.Info(fields.Services[i].Name)
		db.DB.Create(&fields.Services[i])
	}

	fmt.Println("Seeding Sources Table")
	for i := 0; i < len(fields.Sources); i++ {
		l.Log.Info(fields.Sources[i].Name)
		db.DB.Create(&fields.Sources[i])
	}

	fmt.Println("Seeding Statuses Table")
	for i := 0; i < len(fields.Statuses); i++ {
		l.Log.Info(fields.Statuses[i].Name)
		db.DB.Create(&fields.Statuses[i])
	}

	fmt.Println("Seeding Payloads Table")
	for i := 0; i < len(fields.Payloads); i++ {
		l.Log.Info(fields.Payloads[i])
		db.DB.Create(&fields.Payloads[i])
	}

	fmt.Println("Seeding Payload Status Table")
	for i := 0; i < len(fields.PayloadStatuses); i++ {
		date, _ := time.Parse(time.RFC3339, fields.PayloadStatuses[i].Date)

		payloadStatus := models.PayloadStatuses{
			PayloadId: fields.PayloadStatuses[i].PayloadId,
			ServiceId: fields.PayloadStatuses[i].ServiceId,
			SourceId:  fields.PayloadStatuses[i].SourceId,
			StatusId:  fields.PayloadStatuses[i].StatusId,
			StatusMsg: fields.PayloadStatuses[i].StatusMsg,
			Date:      date,
		}
		l.Log.Info(payloadStatus)

		db.DB.Create(&payloadStatus)
	}

	fmt.Println("DB seeding complete")
}
