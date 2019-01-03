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

// GetTrips function
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

// CreateTrip function
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
		go func(index int, tripID string){
			destDoc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Add(ctx, trip.Destinations[index])
			if errCreate != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(errCreate)
				response.WriteErrorResponse(w, errCreate)
			}

			_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destDoc.ID).Set(ctx, map[string]interface{}{
				"id": destDoc.ID,
			},firestore.MergeAll)
			if err2 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, err2)
			}
			destinationChannel <- destDoc.ID
		}(i, doc.ID)
	}
	var destIDS []string
	for i:=0; i < len(trip.Destinations); i++ {
		select{
		case res := <- destinationChannel:
			destIDS = append(destIDS,res)
		}
	}
	
	
	tripData := map[string]interface{}{
		"doc": doc,
		"dest_ids": destIDS,
		"wr":  wr,
	}

	response.Write(w, tripData, http.StatusOK)
	return
}

// GetTrip function
func GetTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
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

	snap, err := client.Collection("trips").Doc(tripID).Get(ctx)
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
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Documents(ctx)
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

// UpdateDestination function
func UpdateDestination(w http.ResponseWriter, r *http.Request) {
	destinationID := mux.Vars(r)["destinationId"]
	tripID := mux.Vars(r)["tripId"]
	decoder := json.NewDecoder(r.Body)
	var destination map[string]interface{}
	err := decoder.Decode(&destination)
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


	_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Set(ctx, destination,firestore.MergeAll)

	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)	
		return
	}

	destinationData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, destinationData, http.StatusOK)	
	return
}


// AddDestination function 
func AddDestination(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	exists := false
	decoder := json.NewDecoder(r.Body)
	destinationChannel := make(chan firestore.DocumentRef)
	var destination Destination
	err := decoder.Decode(&destination)
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

	//Check Destination
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Where("destination_id", "==", destination.DestinationID).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			break
		}
		if doc.Exists() == true {
			exists = true;
			break;
		}
	}

	//Adding destinations
	if exists == true {
		response.Write(w, map[string]interface{}{
			"exists": true,
			"message": "This destination already exists for this trip",
		}, http.StatusConflict)	
		return
	}	
	go func(tripID string, destination Destination){
		destDoc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Add(ctx, destination)
		if errCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errCreate)
			response.WriteErrorResponse(w, errCreate)
		}

		_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destDoc.ID).Set(ctx, map[string]interface{}{
			"id": destDoc.ID,
		},firestore.MergeAll)
		if err2 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			response.WriteErrorResponse(w, err2)
		}
		destinationChannel <- *destDoc
	}(tripID, destination)

	var dest firestore.DocumentRef
	for i:=0; i < 1; i++ {
		select{
		case res := <- destinationChannel:
			dest = res
		}
	}
	
	
	
	destinationData := map[string]interface{}{
		"destination": dest,
	}

	response.Write(w, destinationData, http.StatusOK)	
	return
}

// DeleteDestination function 
func DeleteDestination(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	destinationID := mux.Vars(r)["destinationId"]
	
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
	
	_, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, err)
		return
	}
	
	deleteData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, deleteData, http.StatusOK)	
	return
}
