package trips

import (
	"encoding/json" //"sort"
	//"time"
	"fmt"
	"net/http" //"net/url"
	//"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/places" 
	"cloud.google.com/go/firestore"
	"github.com/asqwrd/trotter-api/response"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GetTrips

func GetTrips(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var trips []Trip
	colorChannel := make(chan places.ColorChannel)
	destinationChannel := make(chan DestinationChannel)

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

	iter := client.Collection("trips").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var trip Trip
		doc.DataTo(&trip)
		trips = append(trips, trip)
	}
	for i := 0; i < len(trips); i++ {
		go func(index int) {
			colors, err := places.GetColor(trips[index].Image)
				if err != nil {
					response.WriteErrorResponse(w, err);
				}

				res := new(places.ColorChannel)
				res.Colors = *colors
				res.Index = index
				colorChannel <- *res
				
		}(i)
		go func(index int){
			var dest []Destination
			iter := client.Collection("trips").Doc(trips[index].ID).Collection("destinations").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				var destination Destination
				doc.DataTo(&destination)
				dest = append(dest, destination)
			}
			res := new(DestinationChannel)
			res.Destinations = dest
			res.Index = index
			destinationChannel <- *res
		}(i)
	}
	for i:=0; i < len(trips)*2; i++ {
		select{
		case colors := <- colorChannel:
			if len(colors.Colors.Vibrant) > 0 {
				trips[colors.Index].Color = colors.Colors.Vibrant
			} else if len(colors.Colors.Muted) > 0 {
				trips[colors.Index].Color = colors.Colors.Muted
			} else if len(colors.Colors.LightVibrant) > 0 {
				trips[colors.Index].Color = colors.Colors.LightVibrant
			} else if len(colors.Colors.LightMuted) > 0 {
				trips[colors.Index].Color = colors.Colors.LightMuted
			} else if len(colors.Colors.DarkVibrant) > 0 {
				trips[colors.Index].Color = colors.Colors.DarkVibrant
			} else if len(colors.Colors.DarkMuted) > 0 {
				trips[colors.Index].Color = colors.Colors.DarkMuted
			}
		case des := <- destinationChannel:
			trips[des.Index].Destinations = des.Destinations
		}
	}

	tripsData := map[string]interface{}{
		"trips": trips,
	}

	response.Write(w, tripsData, http.StatusOK)
	return
}

// CreateTrip

func CreateTrip(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var trip TripRes
	destinationChannel := make(chan string)
	err := decoder.Decode(&trip)
	if err != nil {
		fmt.Println(err)
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

	doc, wr, errCreate := client.Collection("trips").Add(ctx, trip.Trip)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errCreate)
		response.WriteErrorResponse(w, errCreate)
	}

	_, err2 := client.Collection("trips").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
	},firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)
	}

	//Adding destinations

	for i:=0; i < len(trip.Destinations); i++ {
		go func(index int, tripId string){
			dest_doc, _, errCreate := client.Collection("trips").Doc(tripId).Collection("destinations").Add(ctx, trip.Destinations[index])
			if errCreate != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(errCreate)
				response.WriteErrorResponse(w, errCreate)
			}

			_, err2 := client.Collection("trips").Doc(tripId).Collection("destinations").Doc(dest_doc.ID).Set(ctx, map[string]interface{}{
				"id": dest_doc.ID,
			},firestore.MergeAll)
			if err2 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, err2)
			}
			destinationChannel <- dest_doc.ID
		}(i, doc.ID)
	}
	var dest_ids []string
	for i:=0; i < len(trip.Destinations); i++ {
		select{
		case res := <- destinationChannel:
			dest_ids = append(dest_ids,res)
		}
	}
	
	
	tripData := map[string]interface{}{
		"doc": doc,
		"dest_ids": dest_ids,
		"wr":  wr,
	}

	response.Write(w, tripData, http.StatusOK)
	return
}

func GetTrip(w http.ResponseWriter, r *http.Request) {
	tripId := mux.Vars(r)["tripId"]
	//tripChannel := make(chan Trip)
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	fmt.Println("Got Trips")
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

	snap, err := client.Collection("trips").Doc(tripId).Get(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}
	var trip Trip
	snap.DataTo(&trip)
	

	colors, err := places.GetColor(trip.Image)
	if err != nil {
		response.WriteErrorResponse(w, err);
	}
	if len(colors.Vibrant) > 0 {
		trip.Color = colors.Vibrant
	} else if len(colors.Muted) > 0 {
		trip.Color = colors.Muted
	} else if len(colors.LightVibrant) > 0 {
		trip.Color = colors.LightVibrant
	} else if len(colors.LightMuted) > 0 {
		trip.Color = colors.LightMuted
	} else if len(colors.DarkVibrant) > 0 {
		trip.Color = colors.DarkVibrant
	} else if len(colors.DarkMuted) > 0 {
		trip.Color = colors.DarkMuted
	}

	var dest []Destination
	iter := client.Collection("trips").Doc(tripId).Collection("destinations").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var destination Destination
		doc.DataTo(&destination)
		dest = append(dest, destination)
	}
	
	tripData := map[string]interface{}{
		"trip": trip,
		"destinations": dest,
	}

	response.Write(w, tripData, http.StatusOK)
	return
}
