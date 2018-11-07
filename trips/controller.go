package trips

import (
	"encoding/json" //"sort"
	//"time"
	"fmt"
	"net/http" //"net/url"
	//"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go"        //"cloud.google.com/go/firestore"
	"github.com/asqwrd/trotter-api/response" //"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	//"google.golang.org/api/iterator"
)

// GetTrips

func GetTrips(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	tripData := map[string]interface{}{
		"triposo_results": []string{"island", "city", "country", "national_park"},
		"sygic_results":   []string{"island", "city", "country", "national_park"},
	}

	response.Write(w, tripData, http.StatusOK)
	return
}

// CreateTrip

func CreateTrip(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var trip Trip
	err := decoder.Decode(&trip)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	doc, wr, errCreate := client.Collection("trips").Add(ctx, trip)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errCreate)
		response.WriteErrorResponse(w, errCreate)
	}

	tripData := map[string]interface{}{
		"doc": doc,
		"wr":  wr,
	}

	response.Write(w, tripData, http.StatusOK)
	return
}
