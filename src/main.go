package main

import (
	"log"
	"os"
	"strings"
	"time"

	"goscraper/src/globals"

	"github.com/joho/godotenv"
)

func main() {
	if globals.DevMode {
		_ = godotenv.Load()
	}

	token := os.Getenv("CRON_TOKEN")
	if token == "" {
		log.Fatalln("CRON_TOKEN not provided")
	}

	log.Println("Running in cron job mode with token:", token)

	data, err := fetchAllData(token)
	if err != nil {
		log.Fatalf("Data fetch failed: %v", err)
	}

	data["token"] = encode(token)

	db, err := newDB()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	err = db.UpsertData("goscrape", data)
	if err != nil {
		log.Fatalf("Failed to upsert data: %v", err)
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
		data, err := getUser(token)
		resultChan <- result{"user", data, err}
	}()
	go func() {
		data, err := getAttendance(token)
		resultChan <- result{"attendance", data, err}
	}()
	go func() {
		data, err := getMarks(token)
		resultChan <- result{"marks", data, err}
	}()
	go func() {
		data, err := getCourses(token)
		resultChan <- result{"courses", data, err}
	}()
	go func() {
		data, err := getTimetable(token)
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

	if user, ok := data["user"].(map[string]interface{}); ok {
		data["regNumber"] = user["RegNumber"]
	}

	// Optionally fetch ophour from db
	db, err := newDB()
	if err == nil {
		encodedToken := encode(token)
		ophour, err := db.GetOphourByToken(encodedToken)
		if err == nil && ophour != "" {
			data["ophour"] = ophour
		}
	}

	return data, nil
}

func encode(str string) string {
	// wrapper for utils.Encode
	return utils.Encode(str)
}

func getUser(token string) (interface{}, error) {
	return handlers.GetUser(token)
}
func getAttendance(token string) (interface{}, error) {
	return handlers.GetAttendance(token)
}
func getMarks(token string) (interface{}, error) {
	return handlers.GetMarks(token)
}
func getCourses(token string) (interface{}, error) {
	return handlers.GetCourses(token)
}
func getTimetable(token string) (interface{}, error) {
	return handlers.GetTimetable(token)
}

func newDB() (*databases.Helper, error) {
	return databases.NewDatabaseHelper()
}
