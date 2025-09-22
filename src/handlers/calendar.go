package handlers

import (
	"log"
	"goscraper/src/helpers"
	"goscraper/src/types"
	"time"
)

func GetCalendar(token string, month int) (*types.CalendarResponse, error) {
	log.Printf("DEBUG: Calendar handler triggered - Token length: %d, Month: %d", len(token), month)
	if token == "" {
		log.Printf("DEBUG: Error - Empty token provided")
		return &types.CalendarResponse{
			Error:   true,
			Message: "Missing authentication token",
			Status:  401,
		}, nil
	}

	// Set date to target specific month for testing (e.g., October 2025)
	targetDate := time.Date(2025, time.October, 1, 0, 0, 0, 0, time.Local) // Start of October
	scraper := helpers.NewCalendarFetcher(targetDate, token)
	log.Printf("DEBUG: CalendarFetcher created for %v", targetDate)

	calendar, err := scraper.GetCalendar()
	if err != nil {
		log.Printf("DEBUG: Error from GetCalendar: %v", err)
		return calendar, err
	}

	log.Printf("DEBUG: Calendar fetched successfully - %d months in response", len(calendar.Calendar))
	return calendar, nil
}
