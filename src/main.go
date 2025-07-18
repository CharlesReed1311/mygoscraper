package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"goscraper/src/handlers"
	"goscraper/src/helpers/databases"
	"goscraper/src/types"
	"goscraper/src/utils"
)

func main() {
	token := os.Getenv("CRON_TOKEN")
	if token == "" {
		log.Fatal("CRON_TOKEN environment variable not set")
	}

	data, err := fetchAllData(token)
	if err != nil {
		log.Fatalf("Error fetching data: %v", err)
	}

	encodedToken := utils.Encode(token)
	data["token"] = encodedToken

	db, err := databases.NewDatabaseHelper()
	if err != nil {
		log.Fatalf("Database initialization error: %v", err)
	}

	err = db.UpsertData("goscrape", data)
	if err != nil {
		log.Fatalf("Database upsert error: %v", err)
	}

	log.Println("Data successfully fetched and stored.")
}

func fetchAllData(token string) (map[string]interface{}, error) {
	type result struct {
		key  string
		data interface{}
		err  error
	}

	resultChan := make(chan result, 5)

	go func() {
		data, err := handlers.GetUser(token)
		resultChan <- result{"user", data, err}
	}()
	go func() {
		data, err := handlers.GetAttendance(token)
		resultChan <- result{"attendance", data, err}
	}()
	go func() {
		data, err := handlers.GetMarks(token)
		resultChan <- result{"marks", data, err}
	}()
	go func() {
		data, err := handlers.GetCourses(token)
		resultChan <- result{"courses", data, err}
	}()
	go func() {
		data, err := handlers.GetTimetable(token)
		resultChan <- result{"timetable", data, err}
	}()

	data := make(map[string]interface{})
	for i := 0; i < 5; i++ {
		r := <-resultChan
		if r.err != nil {
			return nil, r.err
		}
		data[r.key] = r.data
	}

	if user, ok := data["user"].(*types.User); ok {
		data["regNumber"] = user.RegNumber
	}

	db, err := databases.NewDatabaseHelper()
	if err == nil {
		encodedToken := utils.Encode(token)
		ophour, err := db.GetOphourByToken(encodedToken)
		if err == nil && ophour != "" {
			data["ophour"] = ophour
		}
	}

	return data, nil
}
