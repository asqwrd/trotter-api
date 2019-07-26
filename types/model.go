package types

import (
	"firebase.google.com/go/auth"
	"github.com/asqwrd/trotter-api/triposo"
)

//Trip type for trips response
type Trip struct {
	Image        string        `json:"image" firestore:"image"`
	Name         string        `json:"name" firestore:"name"`
	Group        []string      `json:"group" firestore:"group"`
	OwnerID      string        `json:"owner_id" firestore:"owner_id"`
	ID           string        `json:"id" firestore:"id"`
	Color        string        `json:"color" firestore:"color"`
	Destinations []Destination `json:"destinations" firestore:"destinations"`
	ItineraryIDS []string      `json:"itinerary_ids" firestore:"itinerary_ids"`
	UpdatedAt    interface{}   `json:"updated_at" firestore:"updated_at"`
	Travelers    []User        `json:"travelers"`
}

// TripRes Struct for post
type TripRes struct {
	Trip         Trip
	Destinations []Destination
	User         User
	Travelers    []auth.UserInfo
}

// User struct
type User struct {
	DisplayName string `json:"displayName" firestore:"displayName"`
	Email       string `json:"email" firestore:"email"`
	PhoneNumber string `json:"phoneNumber" firestore:"phoneNumber"`
	PhotoURL    string `json:"photoUrl" firestore:"photoUrl"`
	UID         string `json:"uid" firestore:"uid"`
}

// Notification struct
type Notification struct {
	CreateAt int64       `json:"created_at" firestore:"created_at"`
	Type     string      `json:"type" firestore:"type"`
	Data     interface{} `json:"data" firestore:"data"`
	Read     bool        `json:"read" firestore:"read"`
}

// Flight struct
type Flight struct {
	Source   string          `json:"source" firestore:"source"`
	Segments []FlightSegment `json:"segments" firestore:"segments"`
}

// Hotel struct
type Hotel struct {
	Source   string         `json:"source" firestore:"source"`
	Segments []HotelSegment `json:"segments" firestore:"segments"`
}

// FlightSegment struct
type FlightSegment struct {
	Airline              string `json:"airline" firestore:"airline"`
	ArrivalDatetime      string `json:"arrival_datetime" firestore:"arrival_datetime"`
	ArrivalTimeZoneID    string `json:"arrival_time_zone_id" firestore:"arrival_time_zone_id"`
	ClassOfService       string `json:"class_of_service" firestore:"class_of_service"`
	ConfirmationNo       string `json:"confirmation_no" firestore:"confirmation_no"`
	Currency             string `json:"currency" firestore:"currency"`
	DepartureDatetime    string `json:"departure_datetime" firestore:"departure_datetime"`
	Destination          string `json:"destination" firestore:"destination"`
	DestinationAdminCode string `json:"destination_admin_code" firestore:"destination_admin_code"`
	DestinationCityName  string `json:"destination_city_name" firestore:"destination_city_name"`
	DestinationCountry   string `json:"destination_country" firestore:"destination_country"`
	DestinationLat       string `json:"destination_lat" firestore:"destination_lat"`
	DestinationLon       string `json:"destination_lon" firestore:"destination_lon"`
	DestinationName      string `json:"destination_name" firestore:"destination_name"`
	FlightNumber         string `json:"flight_number" firestore:"flight_number"`
	IataCode             string `json:"iata_code" firestore:"iata_code"`
	NumberOfPax          int64  `json:"number_of_pax" firestore:"number_of_pax"`
	Origin               string `json:"origin" firestore:"origin"`
	OriginAdminCode      string `json:"origin_admin_code" firestore:"origin_admin_code"`
	OriginCityName       string `json:"origin_city_name" firestore:"origin_city_name"`
	OriginCountry        string `json:"origin_country" firestore:"origin_country"`
	OriginLat            string `json:"origin_lat" firestore:"origin_lat"`
	OriginLon            string `json:"origin_lon" firestore:"origin_lon"`
	OriginName           string `json:"origin_name" firestore:"origin_name"`
	Price                string `json:"price" firestore:"price"`
	Address1             string `json:"address1" firebase:"address1"`
	Address2             string `json:"address2" firebase:"address2"`
	CheckinDate          string `json:"checkin_date" firebase:"checkin_date"`
	CheckoutDate         string `json:"checkout_date" firebase:"checkout_date"`
	CityName             string `json:"city_name" firebase:"city_name"`
	Country              string `json:"country" firebase:"country"`
	HotelName            string `json:"hotel_name" firebase:"hotel_name"`
	Lat                  string `json:"lat" firebase:"lat"`
	Lon                  string `json:"lon" firebase:"lon"`
	NumberOfRooms        int64  `json:"number_of_rooms" firebase:"number_of_rooms"`
	PostalCode           string `json:"postal_code" firebase:"postal_code"`
	Phone                string `json:"phone" firebase:"phone"`
}

// HotelSegment struct
type HotelSegment struct {
	Address1       string `json:"address1" firebase:"address1"`
	Address2       string `json:"address2" firebase:"address2"`
	CheckinDate    string `json:"checkin_date" firebase:"checkin_date"`
	CheckoutDate   string `json:"checkout_date" firebase:"checkout_date"`
	CityName       string `json:"city_name" firebase:"city_name"`
	ConfirmationNo string `json:"confirmation_no" firebase:"confirmation_no"`
	Country        string `json:"country" firebase:"country"`
	HotelName      string `json:"hotel_name" firebase:"hotel_name"`
	Lat            string `json:"lat" firebase:"lat"`
	Lon            string `json:"lon" firebase:"lon"`
	NumberOfRooms  int64  `json:"number_of_rooms" firebase:"number_of_rooms"`
	PostalCode     string `json:"postal_code" firebase:"postal_code"`
	Phone          string `json:"phone" firebase:"phone"`
}

//Token struct
type Token struct {
	UID   string `json:"uid" firestore:"uid"`
	Token string `json:"token" firestore:"token"`
}

// Destination struct
type Destination struct {
	Location        triposo.Location `json:"location" firestore:"location"`
	DestinationID   string           `json:"destination_id" firestore:"destination_id"`
	DestinationName string           `json:"destination_name" firestore:"destination_name"`
	Level           string           `json:"level" firestore:"level"`
	CountryID       string           `json:"country_id" firestore:"country_id"`
	CountryName     string           `json:"country_name" firestore:"country_name"`
	StartDate       int64            `json:"start_date" firestore:"start_date"`
	EndDate         int64            `json:"end_date" firestore:"end_date"`
	ID              string           `json:"id" firestore:"id"`
	ItineraryID     string           `json:"itinerary_id" firestore:"itinerary_id"`
	Image           string           `json:"image" firestore:"image"`
}

// DestinationChannel for routines
type DestinationChannel struct {
	Destinations []Destination
	Index        int
	Error        error
}

// PriceDetail struct
type PriceDetail struct {
	Type  string `json:"type" firestore:"type"`
	Name  string `json:"name" firestore:"name"`
	Value string `json:"value" firestore:"value"`
	Units string `json:"units" firestore:"units"`
}

// FlightsAndAccomodations struct
type FlightsAndAccomodations struct {
	ID            string    `json:"id" firestore:"id"`
	Source        string    `json:"source" firestore:"source"`
	Segments      []Segment `json:"segments" firestore:"segments"`
	Travelers     []string  `json:"travelers" firestore:"travelers"`
	TravelersFull []User    `json:"travelers_full" firestore:"travelers_full"`
}

// Segment Struct
type Segment struct {
	Type                 string        `json:"type" firestore:"type"`
	Airline              string        `json:"airline,omitempty" firestore:"airline,omitempty"`
	IataCode             string        `json:"iata_code,omitempty" firestore:"iata_code,omitempty"`
	FlightNumber         string        `json:"flight_number,omitempty" firestore:"flight_number,omitempty"`
	SeatAssignment       string        `json:"seat_assignment,omitempty" firestore:"seat_assignment,omitempty"`
	Origin               string        `json:"origin,omitempty" firestore:"origin"`
	OriginName           string        `json:"origin_name,omitempty" firestore:"origin_name,omitempty"`
	OriginCityName       string        `json:"origin_city_name,omitempty" firestore:"origin_city_name,omitempty"`
	OriginAdminCode      string        `json:"origin_admin_code,omitempty" firestore:"origin_admin_code,omitempty"`
	OriginCountry        string        `json:"origin_country,omitempty" firestore:"origin_country,omitempty"`
	OriginLat            string        `json:"origin_lat,omitempty" firestore:"origin_lat,omitempty"`
	OriginLon            string        `json:"origin_lon,omitempty" firestore:"origin_lon,omitempty"`
	Destination          string        `json:"destination,omitempty" firestore:"destination,omitempty"`
	DestinationName      string        `json:"destination_name,omitempty" firestore:"destination_name,omitempty"`
	DestinationCityName  string        `json:"destination_city_name,omitempty" firestore:"destination_city_name,omitempty"`
	DestinationAdminCode string        `json:"destination_admin_code,omitempty" firestore:"destination_admin_code,omitempty"`
	DestinationCountry   string        `json:"destination_country,omitempty" firestore:"destination_country,omitempty"`
	DestinationLat       string        `json:"destination_lat,omitempty" firestore:"destination_lat,omitempty"`
	DestinationLon       string        `json:"destination_lon,omitempty" firestore:"destination_lon,omitempty"`
	DepartureDatetime    string        `json:"departure_datetime,omitempty" firestore:"departure_datetime,omitempty"`
	DepartureTimeZoneID  string        `json:"departure_time_zone_id,omitempty" firestore:"departure_time_zone_id,omitempty"`
	ArrivalDatetime      string        `json:"arrival_datetime,omitempty" firestore:"arrival_datetime,omitempty"`
	ArrivalTimeZoneID    string        `json:"arrival_time_zone_id,omitempty" firestore:"arrival_time_zone_id,omitempty"`
	ConfirmationNo       string        `json:"confirmation_no,omitempty" firestore:"confirmation_no,omitempty"`
	NumberOfPax          int64         `json:"number_of_pax,omitempty" firestore:"number_of_pax,omitempty"`
	TicketNumber         string        `json:"ticket_number,omitempty" firestore:"ticket_number,omitempty"`
	HotelName            string        `json:"hotel_name,omitempty" firestore:"hotel_name,omitempty"`
	Address1             string        `json:"address1,omitempty" firestore:"address1,omitempty"`
	Address2             string        `json:"address2,omitempty" firestore:"address2,omitempty"`
	CityName             string        `json:"city_name,omitempty" firestore:"city_name,omitempty"`
	AdminCode            string        `json:"admin_code,omitempty" firestore:"admin_code,omitempty"`
	Country              string        `json:"country,omitempty" firestore:"country,omitempty"`
	PostalCode           string        `json:"postal_code,omitempty" firestore:"postal_code,omitempty"`
	Lat                  string        `json:"lat,omitempty" firestore:"lat,omitempty"`
	Lon                  string        `json:"lon,omitempty" firestore:"lon,omitempty"`
	CheckinDate          string        `json:"checkin_date,omitempty" firestore:"checkin_date,omitempty"`
	CheckoutDate         string        `json:"checkout_date,omitempty" firestore:"checkout_date,omitempty"`
	TimeZoneID           string        `json:"time_zone_id,omitempty" firestore:"time_zone_id,omitempty"`
	Price                string        `json:"price,omitempty" firestore:"price,omitempty"`
	Currency             string        `json:"currency,omitempty" firestore:"currency,omitempty"`
	NumberOfRooms        int64         `json:"number_of_rooms,omitempty" firestore:"number_of_rooms,omitempty"`
	RoomType             string        `json:"room_type,omitempty" firestore:"room_type,omitempty"`
	PriceDetails         []PriceDetail `json:"price_details,omitempty" firestore:"price_details,omitempty"`
}
