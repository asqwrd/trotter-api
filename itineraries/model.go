package itineraries

import (
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/asqwrd/trotter-api/types"
	"googlemaps.github.io/maps"
)

//Itinerary type for trips response
type Itinerary struct {
	Days                   []Day          `json:"days" firestore:"days"`
	Location               *Location      `json:"location" firestore:"location"`
	Name                   string         `json:"name" firestore:"name"`
	Destination            string         `json:"destination" firestore:"destination"`
	DestinationName        string         `json:"destination_name" firestore:"destination_name"`
	DestinationCountry     string         `json:"destination_country" firestore:"destination_country"`
	DestinationCountryName string         `json:"destination_country_name" firestore:"destination_country_name"`
	ID                     string         `json:"id" firestore:"id"`
	Public                 bool           `json:"public" firestore:"public"`
	StartDate              int64          `json:"start_date" firestore:"start_date"`
	EndDate                int64          `json:"end_date" firestore:"end_date"`
	TripID                 string         `json:"trip_id" firestore:"trip_id"`
	OwnerID                string         `json:"owner_id" firestore:"owner_id"`
	Travelers              []string       `json:"travelers" firestore:"travelers"`
	StartLocation          *StartLocation `json:"start_location" firestore:"start_location"`
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

//StartLocation struct
type StartLocation struct {
	Location *Location `json:"location" firestore:"location"`
	Name     string    `json:"name" firestore:"name"`
}

//ItineraryItem struct
type ItineraryItem struct {
	Description string                     `json:"description" firestore:"description"`
	Poi         *triposo.InternalPlace     `json:"poi" firestore:"poi"`
	Title       string                     `json:"title" firestore:"title"`
	Time        Time                       `json:"time" firestore:"time"`
	Image       string                     `json:"image" firestore:"image"`
	ID          string                     `json:"id" firestore:"id"`
	PoiID       string                     `json:"poi_id" firestore:"poi_id"`
	Color       string                     `json:"color" firestore:"color"`
	Travel      maps.DistanceMatrixElement `json:"travel,omitempty" firestore:"travel,omitempty"`
	AddedBy     *string                    `json:"added_by,omitempty" firestore:"added_by,omitempty"`
	AddedByFull *types.User                `json:"added_by_full,omitempty" firestore:"added_by_full,omitempty"`
}

type Comment struct {
	Msg       string     `json:"msg" firestore:"msg"`
	ID        string     `json:"id" firestore:"id"`
	User      types.User `json:"user" firestore:"user"`
	CreatedAt int64      `json:"created_at" firestore:"created_at"`
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
