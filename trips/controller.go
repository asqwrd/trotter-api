package trips

import (
	"encoding/json" 
	"time"
	"fmt"
	"net/http"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/places" 
	"cloud.google.com/go/firestore" 
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/itineraries"
	"github.com/asqwrd/trotter-api/types"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/url"
	"github.com/asqwrd/trotter-api/utils" 

)

// GetTrips function
func GetTrips(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var trips = make([]types.Trip,0)
	colorChannel := make(chan places.ColorChannel)
	destinationChannel := make(chan types.DestinationChannel)
	var q *url.Values
	args := r.URL.Query()
	q = &args

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

	iter := client.Collection("trips").Where("group", "array-contains",q.Get("user_id")).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var trip types.Trip
		doc.DataTo(&trip)
		trip.Travelers = []types.User{}
		iterTravelers := client.Collection("trips").Doc(trip.ID).Collection("travelers").Documents(ctx)
		for{
			travelersDoc, errTravelers := iterTravelers.Next()
			if errTravelers == iterator.Done {
				break
			}
			if errTravelers != nil {
				response.WriteErrorResponse(w, errTravelers)
				return
			}
			var traveler types.User
			travelersDoc.DataTo(&traveler)
			trip.Travelers = append(trip.Travelers, traveler)
		}
		trips = append(trips, trip)
	}
	for i := 0; i < len(trips); i++ {
		go func(index int) {

			colors, errColor := places.GetColor(trips[index].Image)
				if errColor != nil {
					response.WriteErrorResponse(w, errColor);
					return
				}

				res := new(places.ColorChannel)
				res.Colors = *colors
				res.Index = index
				colorChannel <- *res
				
		}(i)
		go func(index int){
			var dest []types.Destination
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
				var destination types.Destination
				doc.DataTo(&destination)
				dest = append(dest, destination)
			}
			res := new(types.DestinationChannel)
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

	fmt.Println("Got trips");

	response.Write(w, tripsData, http.StatusOK)
	return
}

// CreateTrip function
func CreateTrip(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var trip types.TripRes
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
	fmt.Println(trip.Trip.OwnerID)

	doc, _, errCreate := client.Collection("trips").Add(ctx, trip.Trip)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errCreate)
		response.WriteErrorResponse(w, errCreate)
	}

	_, err2 := client.Collection("trips").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
		"updated_at": firestore.ServerTimestamp,
	},firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)
	}
	_, errUserCreate := client.Collection("trips").Doc(doc.ID).Collection("travelers").Doc(trip.User.UID).Set(ctx, trip.User)
	if errUserCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errUserCreate)
		response.WriteErrorResponse(w, errUserCreate)
		return
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
				return
			}
			var itinerary = itineraries.Itinerary{
				DestinationCountry: trip.Destinations[index].CountryID,
				DestinationCountryName: trip.Destinations[index].CountryName,
				DestinationName: trip.Destinations[index].DestinationName,
				Destination: trip.Destinations[index].DestinationID,
				StartDate: trip.Destinations[index].StartDate,
				EndDate: trip.Destinations[index].EndDate,
				Name: trip.Trip.Name,
				Location: &itineraries.Location{Latitude: trip.Destinations[index].Location.Lat, Longitude: trip.Destinations[index].Location.Lng},
				TripID: tripID,
				OwnerID: trip.Trip.OwnerID,
			}

			_, errDays := itineraries.CreateItineraryHelper(tripID, destDoc.ID, itinerary)
			if errDays != nil {
				response.WriteErrorResponse(w, errDays)
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
	
	id := doc.ID
	tripData, err := getTrip(id)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	response.Write(w, tripData, http.StatusOK)
	return
}

//Private getTrip funtion
func getTrip(tripID string) (map[string]interface{}, error){
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	fmt.Println("Got Trip")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	snap, err := client.Collection("trips").Doc(tripID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var trip types.Trip
	snap.DataTo(&trip)
	

	colors, err := places.GetColor(trip.Image)
	if err != nil {
		return nil, err
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

	var dest []types.Destination
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var destination types.Destination
		doc.DataTo(&destination)
		dest = append(dest, destination)
	}

	trav := []types.User{}
	iterTravelers := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
	for {
		docTravelers, errTravelers := iterTravelers.Next()
		if errTravelers == iterator.Done {
			break
		}
		if errTravelers != nil {
			return nil, errTravelers
		}
		var traveler types.User
		docTravelers.DataTo(&traveler)
		trav = append(trav, traveler)
	}

	trip.Travelers = trav

	tripData := map[string]interface{}{
		"trip": trip,
		"destinations": dest,
		"travelers": trav,
	}
	return tripData, err
}

// GetTrip function
func GetTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	//tripChannel := make(chan Trip)
	tripData, err := getTrip(tripID);
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}
	
	response.Write(w, tripData, http.StatusOK)
	return
}


// UpdateTrip function
func UpdateTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	decoder := json.NewDecoder(r.Body)
	var trip map[string]interface{}
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
	trip["updated_at"] = firestore.ServerTimestamp
	_, err2 := client.Collection("trips").Doc(tripID).Set(ctx, trip,firestore.MergeAll)

	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)	
		return
	}

	if trip["name"] != nil {
		iter := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			var itinerary itineraries.Itinerary
			doc.DataTo(&itinerary)
			_, err3 := client.Collection("itineraries").Doc(itinerary.ID).Set(ctx, map[string]interface{}{"name":trip["name"],},firestore.MergeAll)
			if err3 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, err3)	
				return
			}
		}
	}
	trav := []types.User{}
	if trip["group"] != nil {
		group := trip["group"].([]interface{});
		iter := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			var traveler types.User
			doc.DataTo(&traveler)
			if utils.FindInTripGroup(group, traveler) == false {
				_, err3 := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(traveler.UID).Delete(ctx)
				if err3 != nil {
					// Handle any errors in an appropriate way, such as returning them.
					response.WriteErrorResponse(w, err3)	
					return
				}
			}
		}
		
		iterTravelers := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
		for {
			docTravelers, errTravelers := iterTravelers.Next()
			if errTravelers == iterator.Done {
				break
			}
			if errTravelers != nil {
				response.WriteErrorResponse(w, errTravelers)	
				return
			}
			var traveler types.User
			docTravelers.DataTo(&traveler)
			trav = append(trav, traveler)
		}
	}


	tripData := map[string]interface{}{
		"travelers": trav,
		"success": true,
	}

	response.Write(w, tripData, http.StatusOK)	
	return
}

// AddTraveler function
func AddTraveler(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	decoder := json.NewDecoder(r.Body)
	var trip types.TripRes
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
	
	//Check Destination
	docSnap, _ := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(trip.User.UID).Get(ctx)

	if docSnap.Exists() == false {
		tripSnap, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
		if errTrip != nil {
			response.WriteErrorResponse(w, errTrip)
			return 
		}
		var tripDoc types.Trip
		tripSnap.DataTo(&tripDoc)
		fmt.Println(tripDoc.Group)
		var group = append(tripDoc.Group,trip.User.UID)
		_, errUpdateGroup := client.Collection("trips").Doc(tripID).Set(ctx,map[string]interface{}{
			"group": group,
			"updated_at": firestore.ServerTimestamp,
		},firestore.MergeAll)
	
		if errUpdateGroup != nil {
			fmt.Println("update group failed")
			response.WriteErrorResponse(w, errUpdateGroup)
			return
		}
		_, errUserCreate := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(trip.User.UID).Set(ctx, trip.User)
		if errUserCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errUserCreate)
			response.WriteErrorResponse(w, errUserCreate)
			return
		}
	} else {
		response.Write(w, map[string]interface{}{
			"success": true,
			"exists" : true,
		}, http.StatusOK)	
		return
	}

	fmt.Println("Traveler Added")

	tripData := map[string]interface{}{
		"success": true,
		"exists": false,
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
	errorChannel := make(chan error)
	dayIdsChannel := make(chan string)

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
	 itineraryID := destination["itinerary_id"].(string)
	if len(itineraryID) > 0 {
		_,errUpdateI10 := client.Collection("itineraries").Doc(itineraryID).Set(ctx,map[string]interface{}{
			"end_date": destination["end_date"],
			"start_date": destination["start_date"],
		},firestore.MergeAll)

		if errUpdateI10 != nil {
			response.WriteErrorResponse(w, err2)	
			return
		}
		var daysCount = 0	
		endtm := time.Unix(int64(destination["end_date"].(float64)), 0)
		starttm := time.Unix(int64(destination["start_date"].(float64)), 0)

		diff := endtm.Sub(starttm)
		daysCount = int(diff.Hours()/24) + 1 //include first day
		var itinerariesItems []itineraries.ItineraryItem
		var days []itineraries.Day
		iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").OrderBy("day", firestore.Asc).Documents(ctx)
		for {
			dayDoc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				response.WriteErrorResponse(w, err)
				break
			}
			var day itineraries.Day
			dayDoc.DataTo(&day)
			days = append(days, day)
		}
		if daysCount < len(days) {
			daysRemaining := days[0:daysCount]
			numRemoved := len(days) - len(daysRemaining)
			daysRemoved := days[len(days) - numRemoved:]
			itineraryItemsChannel := make(chan []itineraries.ItineraryItem)
			var itineraryItemsData []itineraries.ItineraryItem
			for i := 0; i < len(daysRemoved); i++ {
				id := daysRemoved[i].ID
				go func(id string){
					itemsItr := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Documents(ctx)
					for {
						doc, err := itemsItr.Next()
						if err == iterator.Done {
							break
						}
						if err != nil {
							errorChannel <- err
							break
						}
						var itineraryItem itineraries.ItineraryItem
						doc.DataTo(&itineraryItem)
						itineraryItemsData = append(itineraryItemsData, itineraryItem)
						_, errItems := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Doc(itineraryItem.ID).Delete(ctx)
						if errItems != nil {
							errorChannel <- errItems
							break
						}

					}

					_, errDays := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Delete(ctx)
					if errDays != nil {
						errorChannel <- errDays
						return
					}
					itineraryItemsChannel <- itineraryItemsData
				}(id)

			}

			for i := 0; i < 1; i++ {
				select{
				case res := <- itineraryItemsChannel:
					itinerariesItems = res
				case err := <- errorChannel:
					response.WriteErrorResponse(w, err)
					return
				}
			}
					
			if len(itinerariesItems) > 0 {
				id := daysRemaining[len(daysRemaining)-1].ID
				for i:=0; i < len(itinerariesItems); i++ {
					_, errAdd := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Doc(itinerariesItems[i].ID).Set(ctx,itinerariesItems[i])
					if errAdd != nil {
						fmt.Println(errAdd)
						//response.WriteErrorResponse(w, errAdd)
						break
					}
				}
			}

		} else if daysCount > len(days) {
			lastIndex := len(days)
			daysAdded := daysCount - len(days)
			for i:=0; i < daysAdded; i++ {
				go func(lastIndex int, i int){
					daydoc, _, errCreate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Add(ctx, map[string]interface{}{
						"day": lastIndex + i,
					})
					if errCreate != nil {
						// Handle any errors in an appropriate way, such as returning them.
						errorChannel <- errCreate
						return
					}
		
					_, errCrUp := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(daydoc.ID).Set(ctx, map[string]interface{}{
						"id": daydoc.ID,
					},firestore.MergeAll)
					if errCrUp != nil {
						// Handle any errors in an appropriate way, such as returning them.
						errorChannel <- errCrUp
						return
					}
					dayIdsChannel <- daydoc.ID
				}(lastIndex, i)
			}
			var ids []string
			for i:=0; i < daysAdded; i++ {
				select{
					case res := <- dayIdsChannel:
						ids = append(ids, res)
					case err := <- errorChannel:
						response.WriteErrorResponse(w, err)
						return
				}
			}
		}
		
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
	var destination types.Destination
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
	go func(tripID string, destination types.Destination){
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

// AddFlightsAndAccomodations function 
func AddFlightsAndAccomodations(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	destinationID := mux.Vars(r)["destinationId"]
	decoder := json.NewDecoder(r.Body)
	var flight types.FlightsAndAccomodations
	err := decoder.Decode(&flight)
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
	doc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Add(ctx, flight)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errCreate)
		response.WriteErrorResponse(w, errCreate)
	}

	_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
	},firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)
	}
	flightData := map[string]interface{}{
		"result": doc,
		"success": true,
	}

	response.Write(w, flightData, http.StatusOK)	
	return
}

// DeleteFlightsAndAccomodation function 
func DeleteFlightsAndAccomodation(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	detailID := mux.Vars(r)["detailId"]
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
	_, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, errDelete)
	}

	flightData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, flightData, http.StatusOK)	
	return
}

// GetFlightsAndAccomodations function 
func GetFlightsAndAccomodations(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	results := make([]map[string]interface{},0)
	var q *url.Values
	args := r.URL.Query()
	q = &args

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

	fmt.Println(q.Get("user_id"))
	defer client.Close()
	itr := client.Collection("trips").Doc(tripID).Collection("destinations").Documents(ctx)
	for {
		dest, errDest := itr.Next()
		if errDest == iterator.Done {
			break
		}
		if errDest != nil {
			response.WriteErrorResponse(w, errDest)
			return
		}
		flightsAccomodations := []types.FlightsAndAccomodations{}
		destination := types.Destination{}
		dest.DataTo(&destination)
		iter := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Collection("flights_accomodations").Where("travelers", "array-contains",q.Get("user_id")).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			var flightAccomodation types.FlightsAndAccomodations
			doc.DataTo(&flightAccomodation)
			flightAccomodation.TravelersFull = []types.User{}
			for _, traveler := range flightAccomodation.Travelers {
				user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
				if errorUser != nil {
					response.WriteErrorResponse(w, errorUser)
					return
				}

				var traveler types.User
				user.DataTo(&traveler)
				flightAccomodation.TravelersFull = append(flightAccomodation.TravelersFull, traveler)

			}
			flightsAccomodations = append(flightsAccomodations, flightAccomodation)
		}
		data :=  map[string]interface{}{
			"destination": destination,
			"details": flightsAccomodations,
		}

		results = append(results, data)

	}
	flightData := map[string]interface{}{
		"flightsAccomodations": results,
		"success": true,
	}

	response.Write(w, flightData, http.StatusOK)	
	return
}

// GetFlightsAndAccomodationTravelers function 
func GetFlightsAndAccomodationTravelers(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]

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

	travelers :=  []types.User{}
	iterTravelers := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
	for{
		travelersDoc, errTravelers := iterTravelers.Next()
		if errTravelers == iterator.Done {
			break
		}
		if errTravelers != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var traveler types.User
		travelersDoc.DataTo(&traveler)
		travelers = append(travelers, traveler)
	}

	
	data := map[string]interface{}{
		"travelers": travelers,
		"success": true,
	}

	response.Write(w, data, http.StatusOK)	
	return
}

// UpdateFlightsAndAccomodationTravelers function 
func UpdateFlightsAndAccomodationTravelers(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	destinationID := mux.Vars(r)["destinationId"]
	detailID := mux.Vars(r)["detailId"]

	decoder := json.NewDecoder(r.Body)
	var detail types.FlightsAndAccomodations
	errDec := decoder.Decode(&detail)
	if errDec != nil {
		fmt.Println(errDec)
		response.WriteErrorResponse(w, errDec)
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

	_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Set(ctx, map[string]interface{}{
			"travelers": detail.Travelers,
		},firestore.MergeAll)
	if err2 != nil {
		fmt.Println(err2)
		response.WriteErrorResponse(w, err2)
		return
	}
	var travelersFull []types.User
	for _, traveler := range detail.Travelers {
		user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
		if errorUser != nil {
			fmt.Println(errorUser);
			response.WriteErrorResponse(w, errorUser)
			return
		}

		var traveler types.User
		user.DataTo(&traveler)
		travelersFull = append(travelersFull, traveler)

	}	

	
	flightData := map[string]interface{}{
		"travelers": travelersFull,
		"success": true,
	}

	response.Write(w, flightData, http.StatusOK)	
	return
}



// AddHotel function 
func AddHotel(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	decoder := json.NewDecoder(r.Body)
	var hotel types.Hotel
	err := decoder.Decode(&hotel)
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
	doc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("hotels").Add(ctx, hotel)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errCreate)
		response.WriteErrorResponse(w, errCreate)
	}

	_, err2 := client.Collection("trips").Doc(tripID).Collection("hotels").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
	},firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)
	}
	hotelData := map[string]interface{}{
		"hotel": doc,
	}

	response.Write(w, hotelData, http.StatusOK)	
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

// DeleteTrip function 
func DeleteTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	var errorChannel = make(chan error)
	var destDeleteChannel = make(chan interface{})
	var travelDeleteChannel = make(chan interface{})

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

	var dest []types.Destination
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
		var destination types.Destination
		doc.DataTo(&destination)
		dest = append(dest, destination)
	}

	for i:=0; i < len(dest); i++ {
		go func(tripID string, destination types.Destination) {
			iter := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Collection("flights_accomodations").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				_, errDeleteDetails := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Collection("flights_accomodations").Doc(doc.Ref.ID).Delete(ctx)
				if errDeleteDetails != nil {
					// Handle any errors in an appropriate way, such as returning them.
					errorChannel <- errDeleteDetails
					return
				}
			}
			deleteRes, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Delete(ctx)
			if errDelete != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errDelete
				return
			}
			
			destDeleteChannel <- deleteRes
		}(tripID,dest[i])
	}

	var travelers []string
	iterTrav := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
	for {
		doc, err := iterTrav.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		dataMap := doc.Ref.ID
		travelers = append(travelers, dataMap)
	}

	for i:=0; i < len(travelers); i++ {
		go func(tripID string, travelerID string) {
			deleteRes, errDelete := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(travelerID).Delete(ctx)
			if errDelete != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errDelete
				return
			}
			
			travelDeleteChannel <- deleteRes
		}(tripID,travelers[i])
	}

	iterI10 := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
	for {
		doc, err := iterI10.Next()
		if err == iterator.Done {
			break
		}
		var itinerary itineraries.Itinerary
		doc.DataTo(&itinerary)

		iterDays := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Documents(ctx)
		for {
			doc, err := iterDays.Next()
			if err == iterator.Done {
				break
			}
			var day itineraries.Day
			doc.DataTo(&day)

			iterItems := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Collection("itinerary_items").Documents(ctx)
			for {
				doc, err := iterItems.Next()
				if err == iterator.Done {
					break
				}
				var item itineraries.ItineraryItem
				doc.DataTo(&item)
				_, errItem := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Collection("itinerary_items").Doc(item.ID).Delete(ctx)
				if errItem != nil {
					// Handle any errors in an appropriate way, such as returning them.
					response.WriteErrorResponse(w, errItem)	
					return
				}
			}

			_, errDay := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Delete(ctx)
			if errDay != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, errDay)	
				return
			}
		}
		_, err3 := client.Collection("itineraries").Doc(itinerary.ID).Delete(ctx)
		if err3 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			response.WriteErrorResponse(w, err3)	
			return
		}
	}

	count := 0
	travCount := 0

	_, errDelete := client.Collection("trips").Doc(tripID).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, err)
		return
	}

	total := len(dest) + len(travelers)

	for i:=0; i < total; i++ {
		select{
		case <- destDeleteChannel:
			count = count + 1
		case <- travelDeleteChannel:
			travCount = travCount + 1
		case err := <- errorChannel:
			response.WriteErrorResponse(w, err)
			return
		}
	}

	
	
	
	deleteData := map[string]interface{}{
		"destinations_deleted": count,
		"travelers_deleted": travCount,
		"success": true,
	}

	response.Write(w, deleteData, http.StatusOK)	
	return
}
