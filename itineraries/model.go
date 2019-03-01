package itineraries

import (
	"github.com/asqwrd/trotter-api/triposo"
)

//Itinerary type for trips response
type Itinerary struct {
	Days                   []Day    `json:"days" firestore:"days"`
	Location               Location `json:"location" firestore:"location"`
	Name                   string   `json:"name" firestore:"name"`
	Destination            string   `json:"destination" firestore:"destination"`
	DestinationName        string   `json:"destination_name" firestore:"destination_name"`
	DestinationCountry     string   `json:"destination_country" firestore:"destination_country"`
	DestinationCountryName string   `json:"destination_country_name" firestore:"destination_country_name"`
	ID                     string   `json:"id" firestore:"id"`
	Public                 bool     `json:"public" firestore:"public"`
	StartDate              int64    `json:"start_date" firestore:"start_date"`
	EndDate                int64    `json:"end_date" firestore:"end_date"`
	TripID                 string   `json:"trip_id" firestore:"trip_id"`
}

//Day struct
type Day struct {
	Date           int             `json:"date" firestore:"date"`
	Day            int             `json:"day" firestore:"day"`
	ID             string          `json:"id" firestore:"id"`
	ItineraryItems []ItineraryItem `json:"itinerary_items" firestore:"itinerary_items"`
}

//Location struct
type Location struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

//ItineraryItem struct
type ItineraryItem struct {
	Description string                 `json:"description" firestore:"description"`
	Poi         *triposo.InternalPlace `json:"poi" firestore:"poi"`
	Title       string                 `json:"title" firestore:"title"`
	Time        Time                   `json:"time" firestore:"time"`
	Image       string                 `json:"image" firestore:"image"`
	ID          string                 `json:"id" firestore:"id"`
	Color       string                 `json:"color" firestore:"color"`
}

//Time struct
type Time struct {
	Unit  string `json:"unit" firestore:"unit"`
	Value string `json:"value" firestore:"value"`
}

// DaysChannel for routines
type DaysChannel struct {
	Days  []Day
	Index int
	Error error
}

// ItineraryRes Struct for post
type ItineraryRes struct {
	Itinerary         Itinerary
	TripDestinationID string `json:"trip_destination_id" firestore:"trip_destination_id"`
}
