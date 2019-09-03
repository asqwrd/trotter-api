package trips

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/itineraries"
	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/types"
	"github.com/asqwrd/trotter-api/utils"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/maddevsio/fcm.v1"
)

// GetTrips function
func GetTrips(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var trips = make([]types.Trip, 0)
	colorChannel := make(chan places.ColorChannel)
	destinationChannel := make(chan types.DestinationChannel)
	var q *url.Values
	args := r.URL.Query()
	q = &args

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	iter := client.Collection("trips").Where("group", "array-contains", q.Get("user_id")).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		var trip types.Trip
		doc.DataTo(&trip)
		trip.Travelers = []types.User{}
		iterTravelers := client.Collection("trips").Doc(trip.ID).Collection("travelers").Documents(ctx)
		for {
			travelersDoc, errTravelers := iterTravelers.Next()
			if errTravelers == iterator.Done {
				break
			}
			if errTravelers != nil {
				fmt.Println(errTravelers)
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
				fmt.Println(errColor)
				response.WriteErrorResponse(w, errColor)
				return
			}

			res := new(places.ColorChannel)
			res.Colors = *colors
			res.Index = index
			colorChannel <- *res

		}(i)
		go func(index int) {
			var dest []types.Destination
			iter := client.Collection("trips").Doc(trips[index].ID).Collection("destinations").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Println(err)
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
	for i := 0; i < len(trips)*2; i++ {
		select {
		case colors := <-colorChannel:
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
		case des := <-destinationChannel:
			trips[des.Index].Destinations = des.Destinations
		}
	}

	tripsData := map[string]interface{}{
		"trips": trips,
	}

	fmt.Println("Got trips")

	response.Write(w, tripsData, http.StatusOK)
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
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
		"id":         doc.ID,
		"updated_at": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(err2)
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

	for i := 0; i < len(trip.Destinations); i++ {
		go func(index int, tripID string) {

			destDoc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Add(ctx, trip.Destinations[index])
			if errCreate != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(errCreate)
				response.WriteErrorResponse(w, errCreate)
			}

			_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destDoc.ID).Set(ctx, map[string]interface{}{
				"id": destDoc.ID,
			}, firestore.MergeAll)
			if err2 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(err2)
				response.WriteErrorResponse(w, err2)
				return
			}
			var itinerary = itineraries.Itinerary{
				DestinationCountry:     trip.Destinations[index].CountryID,
				DestinationCountryName: trip.Destinations[index].CountryName,
				DestinationName:        trip.Destinations[index].DestinationName,
				Destination:            trip.Destinations[index].DestinationID,
				StartDate:              trip.Destinations[index].StartDate,
				EndDate:                trip.Destinations[index].EndDate,
				Name:                   trip.Trip.Name,
				Location:               &itineraries.Location{Latitude: trip.Destinations[index].Location.Lat, Longitude: trip.Destinations[index].Location.Lng},
				TripID:                 tripID,
				OwnerID:                trip.Trip.OwnerID,
				Travelers:              trip.Trip.Group,
			}

			_, errDays := itineraries.CreateItineraryHelper(tripID, destDoc.ID, itinerary)
			if errDays != nil {
				fmt.Println(errDays)
				response.WriteErrorResponse(w, errDays)
			}
			destinationChannel <- destDoc.ID
		}(i, doc.ID)
	}
	var destIDS []string
	for i := 0; i < len(trip.Destinations); i++ {
		select {
		case res := <-destinationChannel:
			destIDS = append(destIDS, res)
		}
	}

	id := doc.ID
	tripData, err := getTrip(id)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	response.Write(w, tripData, http.StatusOK)
	return
}

//Private getTrip funtion
func getTrip(tripID string) (map[string]interface{}, error) {
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
		"trip":         trip,
		"destinations": dest,
		"travelers":    trav,
	}
	return tripData, err
}

// GetTrip function
func GetTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	//tripChannel := make(chan Trip)
	tripData, err := getTrip(tripID)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	response.Write(w, tripData, http.StatusOK)
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
	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()
	var oldTrip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}

	tripDoc.DataTo(&oldTrip)
	var oldTrav []types.User
	oldTravelers := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
	for {
		docTravelers, errTravelers := oldTravelers.Next()
		if errTravelers == iterator.Done {
			break
		}
		if errTravelers != nil {
			fmt.Println(errTravelers)
			response.WriteErrorResponse(w, errTravelers)
			return
		}
		var traveler types.User
		docTravelers.DataTo(&traveler)
		oldTrav = append(oldTrav, traveler)
	}

	userDoc, errUser := client.Collection("users").Doc(q.Get("updatedBy")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var updatedBy types.User
	userDoc.DataTo(&updatedBy)

	trip["updated_at"] = firestore.ServerTimestamp
	_, err2 := client.Collection("trips").Doc(tripID).Set(ctx, trip, firestore.MergeAll)

	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(err2)
		response.WriteErrorResponse(w, err2)
		return
	}
	c := fcm.NewFCM(types.SERVER_KEY)

	if trip["name"] != nil {

		iter := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			var itinerary itineraries.Itinerary
			doc.DataTo(&itinerary)
			_, err3 := client.Collection("itineraries").Doc(itinerary.ID).Set(ctx, map[string]interface{}{"name": trip["name"]}, firestore.MergeAll)
			if err3 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(err3)
				response.WriteErrorResponse(w, err3)
				return
			}
		}

		var tokens []string
		for _, traveler := range oldTrip.Group {
			if traveler != updatedBy.UID {
				notification := types.Notification{
					CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
					Type:     "user_trip",
					Data: map[string]interface{}{"navigationData": map[string]interface{}{
						"id":    tripID,
						"level": "trip",
					}, "user": updatedBy, "subject": updatedBy.DisplayName + " changed trip name from  " + oldTrip.Name + " to " + trip["name"].(string)},
					Read: false,
				}
				notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler).Collection("notifications").Add(ctx, notification)
				if errNotifySet != nil {
					fmt.Println(errNotifySet)
					//response.WriteErrorResponse(w, errNotifySet)
					return
				}
				_, errNotifyID := client.Collection("users").Doc(traveler).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
					"id": notificationDoc.ID,
				}, firestore.MergeAll)
				if errNotifyID != nil {
					fmt.Println(errNotifyID)
					//response.WriteErrorResponse(w, errNotifyID)
					return
				}

				iter := client.Collection("users").Doc(traveler).Collection("devices").Documents(ctx)
				for {
					doc, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						fmt.Println(err)
						//response.WriteErrorResponse(w, err)
						return
					}

					var token types.Token
					doc.DataTo(&token)
					tokens = append(tokens, token.Token)
				}
			}
		}

		if len(tokens) > 0 {
			navigateData := map[string]interface{}{
				"id":    tripID,
				"level": "trip",
			}
			data := map[string]interface{}{
				"focus":            "trips",
				"click_action":     "FLUTTER_NOTIFICATION_CLICK",
				"type":             "user_trip",
				"notificationData": navigateData,
				"user":             updatedBy,
				"msg":              updatedBy.DisplayName + " changed trip name from  " + oldTrip.Name + " to " + trip["name"].(string),
			}

			notification, err := c.Send(fcm.Message{
				Data:             data,
				RegistrationIDs:  tokens,
				CollapseKey:      "Trip name updated",
				ContentAvailable: true,
				Priority:         fcm.PriorityNormal,
				Notification: fcm.Notification{
					Title:       "Trip name updated",
					Body:        updatedBy.DisplayName + " changed trip name from  " + oldTrip.Name + " to " + trip["name"].(string),
					ClickAction: "FLUTTER_NOTIFICATION_CLICK",
					//Badge: user.PhotoURL,
				},
			})
			if err != nil {
				fmt.Println("Notification send err")
				fmt.Println(err)
				//response.WriteErrorResponse(w, err)
			}
			fmt.Println("Status Code   :", notification.StatusCode)
			fmt.Println("Success       :", notification.Success)
			fmt.Println("Fail          :", notification.Fail)
			fmt.Println("Canonical_ids :", notification.CanonicalIDs)
			fmt.Println("Topic MsgId   :", notification.MsgID)
		}

	}

	trav := []types.User{}

	if trip["deleted"] != nil && trip["added"] != nil {
		deleted := trip["deleted"].([]interface{})
		added := trip["added"].([]interface{})
		if len(deleted) > 0 {
			for _, uid := range deleted {
				_, err3 := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(uid.(string)).Delete(ctx)
				if err3 != nil {
					// Handle any errors in an appropriate way, such as returning them.
					fmt.Println(err3)
					response.WriteErrorResponse(w, err3)
					return
				}
			}
			var group []string
			iter := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				var traveler types.User
				doc.DataTo(&traveler)
				group = append(group, traveler.UID)
			}

			_, errGroup := client.Collection("trips").Doc(tripID).Set(ctx, map[string]interface{}{
				"group": group,
			}, firestore.MergeAll)
			if errGroup != nil {
				fmt.Println(errGroup)
				response.WriteErrorResponse(w, errGroup)
				return
			}

			itr := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
			for {
				docIt, errI10 := itr.Next()
				if errI10 == iterator.Done {
					break
				}
				if errI10 != nil {
					fmt.Println(errI10)
					response.WriteErrorResponse(w, errI10)
					break
				}

				_, errTrav := client.Collection("itineraries").Doc(docIt.Ref.ID).Set(ctx, map[string]interface{}{
					"travelers": group,
				}, firestore.MergeAll)
				if errTrav != nil {
					fmt.Println("deleted It")
					fmt.Println(errTrav)
					response.WriteErrorResponse(w, errTrav)
					break
				}

			}
		}

		if len(added) > 0 {
			for _, user := range added {
				_, err3 := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(user.(map[string]interface{})["uid"].(string)).Set(ctx, user.(map[string]interface{}))
				if err3 != nil {
					// Handle any errors in an appropriate way, such as returning them.
					fmt.Println("Added error It")
					fmt.Println(err3)
					response.WriteErrorResponse(w, err3)
					return
				}
			}
			var group []string
			iter := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				var traveler types.User
				doc.DataTo(&traveler)
				group = append(group, traveler.UID)
			}

			_, errGroup := client.Collection("trips").Doc(tripID).Set(ctx, map[string]interface{}{
				"group": group,
			}, firestore.MergeAll)
			if errGroup != nil {
				fmt.Println(errGroup)
				response.WriteErrorResponse(w, errGroup)
				return
			}

			itr := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
			for {
				docIt, errI10 := itr.Next()
				if errI10 == iterator.Done {
					break
				}
				if errI10 != nil {
					fmt.Println("added errI10")
					fmt.Println(errI10)
					response.WriteErrorResponse(w, errI10)
					break
				}

				_, errTrav := client.Collection("itineraries").Doc(docIt.Ref.ID).Set(ctx, map[string]interface{}{
					"travelers": group,
				}, firestore.MergeAll)
				if errTrav != nil {
					fmt.Println("added errTrav")
					fmt.Println(errTrav)
					response.WriteErrorResponse(w, errTrav)
					break
				}

			}
		}

		var travelersSlice []string

		iterTravelers := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
		for {
			docTravelers, errTravelers := iterTravelers.Next()
			if errTravelers == iterator.Done {
				break
			}
			if errTravelers != nil {
				fmt.Println("errTravelers")
				fmt.Println(errTravelers)
				response.WriteErrorResponse(w, errTravelers)
				return
			}
			var traveler types.User
			docTravelers.DataTo(&traveler)
			trav = append(trav, traveler)
			travelersSlice = append(travelersSlice, traveler.UID)
		}
		for _, traveler := range utils.UniqueUserSlice(append(oldTrav, trav...)) {
			if traveler.UID != updatedBy.UID {
				var msg string
				var notificationType string
				if !utils.Contains(travelersSlice, updatedBy.UID) && utils.Contains(oldTrip.Group, updatedBy.UID) {
					msg = updatedBy.DisplayName + " left " + oldTrip.Name
					notificationType = "user_trip_remove"
				} else if !utils.Contains(travelersSlice, traveler.UID) && utils.Contains(oldTrip.Group, traveler.UID) {
					msg = updatedBy.DisplayName + " removed you from " + oldTrip.Name
					notificationType = "user_trip_remove"
				} else if utils.Contains(travelersSlice, traveler.UID) && !utils.Contains(oldTrip.Group, traveler.UID) {
					msg = updatedBy.DisplayName + " added you to " + oldTrip.Name
					notificationType = "user_trip_added"
				} else {
					msg = updatedBy.DisplayName + " updated travelers for " + oldTrip.Name
					notificationType = "user_trip_updated"
				}

				notification := types.Notification{
					CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
					Type:     notificationType,
					Data: map[string]interface{}{"navigationData": map[string]interface{}{
						"id":    tripID,
						"level": "trip",
					}, "user": updatedBy, "subject": msg},
					Read: false,
				}
				notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler.UID).Collection("notifications").Add(ctx, notification)
				if errNotifySet != nil {
					fmt.Println(errNotifySet)
					//response.WriteErrorResponse(w, errNotifySet)
					return
				}
				_, errNotifyID := client.Collection("users").Doc(traveler.UID).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
					"id": notificationDoc.ID,
				}, firestore.MergeAll)
				if errNotifyID != nil {
					fmt.Println(errNotifyID)
					//response.WriteErrorResponse(w, errNotifyID)
					return
				}

				iter := client.Collection("users").Doc(traveler.UID).Collection("devices").Documents(ctx)
				for {
					doc, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						fmt.Println(err)
						//response.WriteErrorResponse(w, err)
						return
					}

					navigateData := map[string]interface{}{
						"id":    tripID,
						"level": "trip",
					}

					var token types.Token
					doc.DataTo(&token)
					data := map[string]interface{}{
						"focus":            "trips",
						"click_action":     "FLUTTER_NOTIFICATION_CLICK",
						"type":             notificationType,
						"notificationData": navigateData,
						"user":             updatedBy,
						"msg":              msg,
					}

					notification, err := c.Send(fcm.Message{
						Data:             data,
						To:               token.Token,
						CollapseKey:      "Change in travelers!",
						ContentAvailable: true,
						Priority:         fcm.PriorityNormal,
						Notification: fcm.Notification{
							Title:       "Change in travelers!",
							Body:        msg,
							ClickAction: "FLUTTER_NOTIFICATION_CLICK",
							//Badge: user.PhotoURL,
						},
					})
					if err != nil {
						fmt.Println("Notification send err")
						fmt.Println(err)
						//response.WriteErrorResponse(w, err)
					}
					fmt.Println("Status Code   :", notification.StatusCode)
					fmt.Println("Success       :", notification.Success)
					fmt.Println("Fail          :", notification.Fail)
					fmt.Println("Canonical_ids :", notification.CanonicalIDs)
					fmt.Println("Topic MsgId   :", notification.MsgID)

				}
			}
		}
	}

	tripData := map[string]interface{}{
		"travelers": trav,
		"success":   true,
	}

	response.Write(w, tripData, http.StatusOK)
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
	c := fcm.NewFCM(types.SERVER_KEY)

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	//Check Traveler
	docSnap, _ := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(trip.User.UID).Get(ctx)

	if !docSnap.Exists() {
		tripSnap, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
		if errTrip != nil {
			fmt.Println(errTrip)
			response.WriteErrorResponse(w, errTrip)
			return
		}
		var tripDoc types.Trip
		errData1 := tripSnap.DataTo(&tripDoc)
		if errData1 != nil {
			fmt.Println(errData1)
			response.WriteErrorResponse(w, errData1)
			return
		}
		fmt.Println(tripDoc.Group)
		var group = append(tripDoc.Group, trip.User.UID)
		_, errUpdateGroup := client.Collection("trips").Doc(tripID).Set(ctx, map[string]interface{}{
			"group":      group,
			"updated_at": firestore.ServerTimestamp,
		}, firestore.MergeAll)

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

		var tokens []string

		iter := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
		for {
			docIt, errI10 := iter.Next()
			if errI10 == iterator.Done {
				break
			}
			if errI10 != nil {
				fmt.Println(errI10)
				response.WriteErrorResponse(w, errI10)
				break
			}

			var i10 itineraries.Itinerary
			errData := docIt.DataTo(&i10)
			if errData != nil {
				fmt.Println(errData)
				response.WriteErrorResponse(w, errData)
				break
			}
			travelers := i10.Travelers
			travelers = append(travelers, trip.User.UID)
			_, errTrav := client.Collection("itineraries").Doc(docIt.Ref.ID).Set(ctx, map[string]interface{}{
				"travelers": travelers,
			}, firestore.MergeAll)
			if errTrav != nil {
				fmt.Println(errTrav)
				response.WriteErrorResponse(w, errTrav)
				break
			}
			for _, traveler := range tripDoc.Group {
				notification := types.Notification{
					CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
					Type:     "user_trip",
					Data: map[string]interface{}{"navigationData": map[string]interface{}{
						"id":    tripID,
						"level": "trip",
					}, "user": trip.User, "subject": trip.User.DisplayName + " joined " + tripDoc.Name},
					Read: false,
				}
				notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler).Collection("notifications").Add(ctx, notification)
				if errNotifySet != nil {
					fmt.Println(errNotifySet)
					//response.WriteErrorResponse(w, errNotifySet)
					return
				}
				_, errNotifyID := client.Collection("users").Doc(traveler).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
					"id": notificationDoc.ID,
				}, firestore.MergeAll)
				if errNotifyID != nil {
					fmt.Println(errNotifyID)
					//response.WriteErrorResponse(w, errNotifyID)
					return
				}

				iter := client.Collection("users").Doc(traveler).Collection("devices").Documents(ctx)
				for {
					doc, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						fmt.Println(err)
						//response.WriteErrorResponse(w, err)
						return
					}

					var token types.Token
					doc.DataTo(&token)
					tokens = append(tokens, token.Token)

				}
			}

		}

		if len(tokens) > 0 {

			navigateData := map[string]interface{}{
				"id":    tripID,
				"level": "trip",
			}
			data := map[string]interface{}{
				"focus":            "trips",
				"click_action":     "FLUTTER_NOTIFICATION_CLICK",
				"type":             "user_trip",
				"notificationData": navigateData,
				"user":             trip.User,
				"msg":              trip.User.DisplayName + " joined " + tripDoc.Name,
			}

			notification, err := c.Send(fcm.Message{
				Data:             data,
				RegistrationIDs:  tokens,
				CollapseKey:      "New traveler",
				ContentAvailable: true,
				Priority:         fcm.PriorityNormal,
				Notification: fcm.Notification{
					Title:       "New traveler",
					Body:        trip.User.DisplayName + " joined " + tripDoc.Name,
					ClickAction: "FLUTTER_NOTIFICATION_CLICK",
					//Badge: user.PhotoURL,
				},
			})
			if err != nil {
				fmt.Println("Notification send err")
				fmt.Println(err)
				//response.WriteErrorResponse(w, err)
			}
			fmt.Println("Status Code   :", notification.StatusCode)
			fmt.Println("Success       :", notification.Success)
			fmt.Println("Fail          :", notification.Fail)
			fmt.Println("Canonical_ids :", notification.CanonicalIDs)
			fmt.Println("Topic MsgId   :", notification.MsgID)
		}

	} else {
		response.Write(w, map[string]interface{}{
			"success": true,
			"exists":  true,
		}, http.StatusOK)
		return
	}

	fmt.Println("Traveler Added")

	tripData := map[string]interface{}{
		"success": true,
		"exists":  false,
	}

	response.Write(w, tripData, http.StatusOK)
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Set(ctx, destination, firestore.MergeAll)

	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(err2)
		response.WriteErrorResponse(w, err2)
		return
	}
	itineraryID := destination["itinerary_id"].(string)
	if len(itineraryID) > 0 {
		_, errUpdateI10 := client.Collection("itineraries").Doc(itineraryID).Set(ctx, map[string]interface{}{
			"end_date":   destination["end_date"],
			"start_date": destination["start_date"],
		}, firestore.MergeAll)

		if errUpdateI10 != nil {
			fmt.Println(err2)
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
				fmt.Println(err)
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
			daysRemoved := days[len(days)-numRemoved:]
			itineraryItemsChannel := make(chan []itineraries.ItineraryItem)
			var itineraryItemsData []itineraries.ItineraryItem
			for i := 0; i < len(daysRemoved); i++ {
				id := daysRemoved[i].ID
				go func(id string) {
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
				select {
				case res := <-itineraryItemsChannel:
					itinerariesItems = res
				case err := <-errorChannel:
					fmt.Println(err)
					response.WriteErrorResponse(w, err)
					return
				}
			}

			if len(itinerariesItems) > 0 {
				id := daysRemaining[len(daysRemaining)-1].ID
				for i := 0; i < len(itinerariesItems); i++ {
					_, errAdd := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Doc(itinerariesItems[i].ID).Set(ctx, itinerariesItems[i])
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
			for i := 0; i < daysAdded; i++ {
				go func(lastIndex int, i int) {
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
					}, firestore.MergeAll)
					if errCrUp != nil {
						// Handle any errors in an appropriate way, such as returning them.
						errorChannel <- errCrUp
						return
					}
					dayIdsChannel <- daydoc.ID
				}(lastIndex, i)
			}
			var ids []string
			for i := 0; i < daysAdded; i++ {
				select {
				case res := <-dayIdsChannel:
					ids = append(ids, res)
				case err := <-errorChannel:
					fmt.Println(err)
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

	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}

	tripDoc.DataTo(&trip)

	userDoc, errUser := client.Collection("users").Doc(q.Get("updatedBy")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var updatedBy types.User
	userDoc.DataTo(&updatedBy)

	//Check Destination
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Where("destination_id", "==", destination.DestinationID).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			break
		}
		if doc.Exists() == true {
			exists = true
			break
		}
	}

	//Adding destinations
	if exists {
		response.Write(w, map[string]interface{}{
			"exists":  true,
			"message": "This destination already exists for this trip",
		}, http.StatusConflict)
		return
	}
	go func(tripID string, destination types.Destination) {
		var trip types.Trip
		tripDoc, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
		if errTrip != nil {
			fmt.Println(errTrip)
			response.WriteErrorResponse(w, errTrip)
		}
		tripDoc.DataTo(&trip)

		destDoc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Add(ctx, destination)
		if errCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println("here")
			fmt.Println(errCreate)
			response.WriteErrorResponse(w, errCreate)
			return
		}

		_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destDoc.ID).Set(ctx, map[string]interface{}{
			"id": destDoc.ID,
		}, firestore.MergeAll)
		if err2 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(err2)
			response.WriteErrorResponse(w, err2)
			return
		}
		var itinerary = itineraries.Itinerary{
			DestinationCountry:     destination.CountryID,
			DestinationCountryName: destination.CountryName,
			DestinationName:        destination.DestinationName,
			Destination:            destination.DestinationID,
			StartDate:              destination.StartDate,
			EndDate:                destination.EndDate,
			Name:                   trip.Name,
			Location:               &itineraries.Location{Latitude: destination.Location.Lat, Longitude: destination.Location.Lng},
			TripID:                 tripID,
			OwnerID:                trip.OwnerID,
			Travelers:              trip.Group,
		}
		_, errDays := itineraries.CreateItineraryHelper(tripID, destDoc.ID, itinerary)
		if errDays != nil {

			fmt.Println(errDays)
			response.WriteErrorResponse(w, errDays)
		}

		destinationChannel <- *destDoc
	}(tripID, destination)

	var dest firestore.DocumentRef
	for i := 0; i < 1; i++ {
		select {
		case res := <-destinationChannel:
			dest = res
		}
	}

	destinationData := map[string]interface{}{
		"destination": dest,
	}
	c := fcm.NewFCM(types.SERVER_KEY)
	navigateData := map[string]interface{}{
		"tripId": tripID, "level": "trip",
	}

	var tokens []string

	for _, traveler := range trip.Group {
		if traveler != updatedBy.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_trip",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": updatedBy, "subject": updatedBy.DisplayName + " added " + destination.DestinationName + " to " + trip.Name},
				Read:     false,
			}
			notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler).Collection("notifications").Add(ctx, notification)
			if errNotifySet != nil {
				fmt.Println(errNotifySet)
				//response.WriteErrorResponse(w, errNotifySet)
				return
			}
			_, errNotifyID := client.Collection("users").Doc(traveler).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
				"id": notificationDoc.ID,
			}, firestore.MergeAll)
			if errNotifyID != nil {
				fmt.Println(errNotifyID)
				//response.WriteErrorResponse(w, errNotifyID)
				return
			}

			iter := client.Collection("users").Doc(traveler).Collection("devices").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Println(err)
					//response.WriteErrorResponse(w, err)
					return
				}

				var token types.Token
				doc.DataTo(&token)
				tokens = append(tokens, token.Token)

			}
		}
	}
	if len(tokens) > 0 {

		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"type":             "user_trip",
			"notificationData": navigateData,
			"user":             updatedBy,
			"msg":              updatedBy.DisplayName + " added " + destination.DestinationName + " to " + trip.Name,
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  tokens,
			CollapseKey:      "Destination Added!",
			ContentAvailable: true,
			Priority:         fcm.PriorityNormal,
			Notification: fcm.Notification{
				Title:       "Destination Added!",
				Body:        updatedBy.DisplayName + " added " + destination.DestinationName + " to " + trip.Name,
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				//Badge: user.PhotoURL,
			},
		})
		if err != nil {
			fmt.Println("Notification send err")
			fmt.Println(err)
			//response.WriteErrorResponse(w, err)
		}
		fmt.Println("Status Code   :", notification.StatusCode)
		fmt.Println("Success       :", notification.Success)
		fmt.Println("Fail          :", notification.Fail)
		fmt.Println("Canonical_ids :", notification.CanonicalIDs)
		fmt.Println("Topic MsgId   :", notification.MsgID)
	}

	response.Write(w, destinationData, http.StatusOK)
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
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
	}, firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(err2)
		response.WriteErrorResponse(w, err2)
	}
	flightData := map[string]interface{}{
		"result":  doc,
		"success": true,
	}

	response.Write(w, flightData, http.StatusOK)
}

// DeleteFlightsAndAccomodation function
func DeleteFlightsAndAccomodation(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	detailID := mux.Vars(r)["detailId"]
	destinationID := mux.Vars(r)["destinationId"]
	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()
	detailDoc, errDetail := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Get(ctx)
	if errDetail != nil {
		fmt.Println(errDetail)
		response.WriteErrorResponse(w, errDetail)
		return
	}

	var detail types.FlightsAndAccomodations
	erro := detailDoc.DataTo(&detail)
	if erro != nil {
		fmt.Println(erro)
		response.WriteErrorResponse(w, erro)
		return
	}

	userDoc, errUser := client.Collection("users").Doc(q.Get("deletedBy")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var deletedBy types.User
	userDoc.DataTo(&deletedBy)

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
	}
	tripDoc.DataTo(&trip)

	_, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, errDelete)
	}

	var tokens []string
	navigateData := map[string]interface{}{
		"id": tripID, "level": "trip",
	}
	c := fcm.NewFCM(types.SERVER_KEY)

	for _, traveler := range detail.Travelers {
		if traveler != deletedBy.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_trip",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": deletedBy, "subject": deletedBy.DisplayName + " removed a travel itinerary from " + trip.Name},
				Read:     false,
			}
			notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler).Collection("notifications").Add(ctx, notification)
			if errNotifySet != nil {
				fmt.Println(errNotifySet)
				//response.WriteErrorResponse(w, errNotifySet)
				return
			}
			_, errNotifyID := client.Collection("users").Doc(traveler).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
				"id": notificationDoc.ID,
			}, firestore.MergeAll)
			if errNotifyID != nil {
				fmt.Println(errNotifyID)
				//response.WriteErrorResponse(w, errNotifyID)
				return
			}

			iter := client.Collection("users").Doc(traveler).Collection("devices").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Println(err)
					//response.WriteErrorResponse(w, err)
					return
				}

				var token types.Token
				doc.DataTo(&token)
				tokens = append(tokens, token.Token)

			}
		}
	}
	if len(tokens) > 0 {

		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"type":             "user_trip",
			"notificationData": navigateData,
			"user":             deletedBy,
			"msg":              deletedBy.DisplayName + " removed a travel itinerary from " + trip.Name,
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  tokens,
			CollapseKey:      "Travel itinerary removed!",
			ContentAvailable: true,
			Priority:         fcm.PriorityNormal,
			Notification: fcm.Notification{
				Title:       "Travel itinerary removed!",
				Body:        deletedBy.DisplayName + " removed a travel itinerary from " + trip.Name,
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				//Badge: user.PhotoURL,
			},
		})
		if err != nil {
			fmt.Println("Notification send err")
			fmt.Println(err)
			//response.WriteErrorResponse(w, err)
		}
		fmt.Println("Status Code   :", notification.StatusCode)
		fmt.Println("Success       :", notification.Success)
		fmt.Println("Fail          :", notification.Fail)
		fmt.Println("Canonical_ids :", notification.CanonicalIDs)
		fmt.Println("Topic MsgId   :", notification.MsgID)
	}

	flightData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, flightData, http.StatusOK)
}

// GetFlightsAndAccomodations function
func GetFlightsAndAccomodations(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	results := make([]map[string]interface{}, 0)
	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
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
			fmt.Println(errDest)
			response.WriteErrorResponse(w, errDest)
			return
		}
		flightsAccomodations := []types.FlightsAndAccomodations{}
		destination := types.Destination{}
		dest.DataTo(&destination)
		iter := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Collection("flights_accomodations").Where("travelers", "array-contains", q.Get("user_id")).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
			var flightAccomodation types.FlightsAndAccomodations
			doc.DataTo(&flightAccomodation)
			flightAccomodation.TravelersFull = []types.User{}
			for _, traveler := range flightAccomodation.Travelers {
				user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
				if errorUser != nil {
					fmt.Println(errorUser)
					response.WriteErrorResponse(w, errorUser)
					return
				}

				var traveler types.User
				user.DataTo(&traveler)
				flightAccomodation.TravelersFull = append(flightAccomodation.TravelersFull, traveler)

			}
			flightsAccomodations = append(flightsAccomodations, flightAccomodation)
		}
		data := map[string]interface{}{
			"destination": destination,
			"details":     flightsAccomodations,
		}

		results = append(results, data)

	}
	flightData := map[string]interface{}{
		"flightsAccomodations": results,
		"success":              true,
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	travelers := []types.User{}
	iterTravelers := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
	for {
		travelersDoc, errTravelers := iterTravelers.Next()
		if errTravelers == iterator.Done {
			break
		}
		if errTravelers != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		var traveler types.User
		travelersDoc.DataTo(&traveler)
		travelers = append(travelers, traveler)
	}

	data := map[string]interface{}{
		"travelers": travelers,
		"success":   true,
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
	var detail types.TravelersUpdateBody
	errDec := decoder.Decode(&detail)
	if errDec != nil {
		fmt.Println(errDec)
		response.WriteErrorResponse(w, errDec)
		return
	}

	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	detailDoc, errDetail := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Get(ctx)
	if errDetail != nil {
		fmt.Println(errDetail)
		response.WriteErrorResponse(w, errDetail)
		return
	}

	var oldDetail types.FlightsAndAccomodations
	erro := detailDoc.DataTo(&oldDetail)
	if erro != nil {
		fmt.Println(erro)
		response.WriteErrorResponse(w, erro)
		return
	}

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}

	tripDoc.DataTo(&trip)

	userDoc, errUser := client.Collection("users").Doc(q.Get("updatedBy")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var updatedBy types.User
	userDoc.DataTo(&updatedBy)
	var flightData map[string]interface{}
	detailTravelers := oldDetail.Travelers

	if len(detail.Added) > 0 {

		for _, added := range detail.Added {

			_, errAdd := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(added.UID).Set(ctx, added)
			if errAdd != nil {
				fmt.Println(errAdd)
				response.WriteErrorResponse(w, errAdd)
				return
			}

			detailTravelers = append(detailTravelers, added.UID)
		}

		_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Set(ctx, map[string]interface{}{
			"travelers": detailTravelers,
		}, firestore.MergeAll)
		if err2 != nil {
			fmt.Println(err2)
			response.WriteErrorResponse(w, err2)
			return
		}

		var oldTravelersFull []types.User
		var travelersFull []types.User

		for _, traveler := range detailTravelers {
			user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
			if errorUser != nil {
				fmt.Println(errorUser)
				response.WriteErrorResponse(w, errorUser)
				return
			}

			var traveler types.User
			err := user.DataTo(&traveler)
			if err != nil {
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
			travelersFull = append(travelersFull, traveler)

		}

		for _, traveler := range oldDetail.Travelers {
			user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
			if errorUser != nil {
				fmt.Println(errorUser)
				response.WriteErrorResponse(w, errorUser)
				return
			}

			var traveler types.User
			err := user.DataTo(&traveler)
			if err != nil {
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
			oldTravelersFull = append(oldTravelersFull, traveler)

		}

		flightData = map[string]interface{}{
			"travelers": travelersFull,
			"success":   true,
		}

		c := fcm.NewFCM(types.SERVER_KEY)

		for _, traveler := range utils.UniqueUserSlice(append(oldTravelersFull, travelersFull...)) {
			if traveler.UID != updatedBy.UID {
				navigateData := map[string]interface{}{
					"tripId": tripID, "level": "travelinfo", "ownerId": trip.OwnerID,
				}

				var msg string
				var notificationType string
				if utils.Contains(detailTravelers, traveler.UID) && !utils.Contains(oldDetail.Travelers, traveler.UID) {
					msg = updatedBy.DisplayName + " added you to a travel itinerary in " + trip.Name
					notificationType = "user_travel_details_add"
				} else {
					msg = updatedBy.DisplayName + " added " + traveler.DisplayName + " to a travel itinerary in " + trip.Name
					notificationType = "user_travel_details_add"
				}

				notification := types.Notification{
					CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
					Type:     notificationType,
					Data:     map[string]interface{}{"navigationData": navigateData, "user": updatedBy, "subject": msg},
					Read:     false,
				}
				notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler.UID).Collection("notifications").Add(ctx, notification)
				if errNotifySet != nil {
					fmt.Println(errNotifySet)
					//	response.WriteErrorResponse(w, errNotifySet)
					return
				}
				_, errNotifyID := client.Collection("users").Doc(traveler.UID).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
					"id": notificationDoc.ID,
				}, firestore.MergeAll)
				if errNotifyID != nil {
					fmt.Println(errNotifyID)
					//response.WriteErrorResponse(w, errNotifyID)
					return
				}

				iter := client.Collection("users").Doc(traveler.UID).Collection("devices").Documents(ctx)
				for {
					doc, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						fmt.Println(err)
						//response.WriteErrorResponse(w, err)
						return
					}

					var token types.Token
					doc.DataTo(&token)
					data := map[string]interface{}{
						"focus":            "trips",
						"click_action":     "FLUTTER_NOTIFICATION_CLICK",
						"type":             notificationType,
						"notificationData": navigateData,
						"user":             updatedBy,
						"msg":              msg,
					}

					notification, errSend := c.Send(fcm.Message{
						Data:             data,
						To:               token.Token,
						CollapseKey:      "Travel itinerary update",
						ContentAvailable: true,
						Priority:         fcm.PriorityNormal,
						Notification: fcm.Notification{
							Title:       "Travel itinerary update",
							Body:        msg,
							ClickAction: "FLUTTER_NOTIFICATION_CLICK",
							//Badge: user.PhotoURL,
						},
					})
					if errSend != nil {
						fmt.Println("Notification send err")
						fmt.Println(errSend)
						//response.WriteErrorResponse(w, errSend)
					}
					fmt.Println("Status Code   :", notification.StatusCode)
					fmt.Println("Success       :", notification.Success)
					fmt.Println("Fail          :", notification.Fail)
					fmt.Println("Canonical_ids :", notification.CanonicalIDs)
					fmt.Println("Topic MsgId   :", notification.MsgID)

				}
			}
		}
	}

	if len(detail.Deleted) > 0 {

		var detailTravelersKeep []string
		for _, old := range detailTravelers {
			if !utils.Contains(detail.Deleted, old) {
				detailTravelersKeep = append(detailTravelersKeep, old)
			}
		}

		_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(detailID).Set(ctx, map[string]interface{}{
			"travelers": detailTravelersKeep,
		}, firestore.MergeAll)
		if err2 != nil {
			fmt.Println(err2)
			response.WriteErrorResponse(w, err2)
			return
		}

		var oldTravelersFull []types.User
		var travelersFull []types.User

		for _, traveler := range detailTravelersKeep {
			user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
			if errorUser != nil {
				fmt.Println(errorUser)
				response.WriteErrorResponse(w, errorUser)
				return
			}

			var traveler types.User
			err := user.DataTo(&traveler)
			if err != nil {
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
			travelersFull = append(travelersFull, traveler)

		}

		for _, traveler := range detailTravelers {
			user, errorUser := client.Collection("users").Doc(traveler).Get(ctx)
			if errorUser != nil {
				fmt.Println(errorUser)
				response.WriteErrorResponse(w, errorUser)
				return
			}

			var traveler types.User
			err := user.DataTo(&traveler)
			if err != nil {
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
			oldTravelersFull = append(oldTravelersFull, traveler)

		}

		flightData = map[string]interface{}{
			"travelers": travelersFull,
			"success":   true,
		}

		c := fcm.NewFCM(types.SERVER_KEY)

		for _, traveler := range utils.UniqueUserSlice(append(oldTravelersFull, travelersFull...)) {
			if traveler.UID != updatedBy.UID {
				navigateData := map[string]interface{}{
					"tripId": tripID, "level": "travelinfo", "ownerId": trip.OwnerID,
				}

				var msg string
				var notificationType string
				if !utils.Contains(detailTravelers, updatedBy.UID) && utils.Contains(oldDetail.Travelers, updatedBy.UID) {
					msg = updatedBy.DisplayName + " left one of the travel itineraries for " + trip.Name
					notificationType = "user_travel_details_remove"
				} else if !utils.Contains(detailTravelers, traveler.UID) && utils.Contains(oldDetail.Travelers, traveler.UID) {
					msg = updatedBy.DisplayName + " removed you from one of the travel itineraries for " + trip.Name
					notificationType = "user_travel_details_remove"
				}

				notification := types.Notification{
					CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
					Type:     notificationType,
					Data:     map[string]interface{}{"navigationData": navigateData, "user": updatedBy, "subject": msg},
					Read:     false,
				}
				notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler.UID).Collection("notifications").Add(ctx, notification)
				if errNotifySet != nil {
					fmt.Println(errNotifySet)
					//	response.WriteErrorResponse(w, errNotifySet)
					return
				}
				_, errNotifyID := client.Collection("users").Doc(traveler.UID).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
					"id": notificationDoc.ID,
				}, firestore.MergeAll)
				if errNotifyID != nil {
					fmt.Println(errNotifyID)
					//response.WriteErrorResponse(w, errNotifyID)
					return
				}

				iter := client.Collection("users").Doc(traveler.UID).Collection("devices").Documents(ctx)
				for {
					doc, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						fmt.Println(err)
						//response.WriteErrorResponse(w, err)
						return
					}

					var token types.Token
					doc.DataTo(&token)
					data := map[string]interface{}{
						"focus":            "trips",
						"click_action":     "FLUTTER_NOTIFICATION_CLICK",
						"type":             notificationType,
						"notificationData": navigateData,
						"user":             updatedBy,
						"msg":              msg,
					}

					notification, errSend := c.Send(fcm.Message{
						Data:             data,
						To:               token.Token,
						CollapseKey:      "Travel itinerary update",
						ContentAvailable: true,
						Priority:         fcm.PriorityNormal,
						Notification: fcm.Notification{
							Title:       "Travel itinerary update",
							Body:        msg,
							ClickAction: "FLUTTER_NOTIFICATION_CLICK",
							//Badge: user.PhotoURL,
						},
					})
					if errSend != nil {
						fmt.Println("Notification send err")
						fmt.Println(errSend)
						//response.WriteErrorResponse(w, errSend)
					}
					fmt.Println("Status Code   :", notification.StatusCode)
					fmt.Println("Success       :", notification.Success)
					fmt.Println("Fail          :", notification.Fail)
					fmt.Println("Canonical_ids :", notification.CanonicalIDs)
					fmt.Println("Topic MsgId   :", notification.MsgID)

				}
			}
		}
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
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
	}, firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(err2)
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
	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(tripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}

	tripDoc.DataTo(&trip)

	userDoc, errUser := client.Collection("users").Doc(q.Get("updatedBy")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var updatedBy types.User
	userDoc.DataTo(&updatedBy)

	var destination types.Destination
	destDoc, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Get(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, errDelete)
		return
	}

	destDoc.DataTo(&destination)

	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		_, errDeleteDetails := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Collection("flights_accomodations").Doc(doc.Ref.ID).Delete(ctx)
		if errDeleteDetails != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errDeleteDetails)
			response.WriteErrorResponse(w, errDeleteDetails)
			return
		}
	}

	iterI10 := client.Collection("itineraries").Where("trip_id", "==", tripID).Documents(ctx)
	for {
		doc, err := iterI10.Next()
		if err == iterator.Done {
			break
		}
		var itinerary itineraries.Itinerary
		doc.DataTo(&itinerary)

		if itinerary.Destination == destination.DestinationID {
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
						fmt.Println(errItem)
						response.WriteErrorResponse(w, errItem)
						return
					}
				}

				_, errDay := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Delete(ctx)
				if errDay != nil {
					// Handle any errors in an appropriate way, such as returning them.
					fmt.Println(errDay)
					response.WriteErrorResponse(w, errDay)
					return
				}
			}
			_, err3 := client.Collection("itineraries").Doc(itinerary.ID).Delete(ctx)
			if err3 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(err3)
				response.WriteErrorResponse(w, err3)
				return
			}
		}

	}

	_, errDeleteDest := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Delete(ctx)
	if errDeleteDest != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDeleteDest)
		response.WriteErrorResponse(w, errDeleteDest)
		return
	}

	c := fcm.NewFCM(types.SERVER_KEY)

	var tokens []string
	navigateData := map[string]interface{}{
		"tripId": tripID, "level": "trip",
	}

	for _, traveler := range trip.Group {
		if traveler != updatedBy.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_trip",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": updatedBy, "subject": updatedBy.DisplayName + " removed " + destination.DestinationName + " from " + trip.Name},
				Read:     false,
			}
			notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler).Collection("notifications").Add(ctx, notification)
			if errNotifySet != nil {
				fmt.Println(errNotifySet)
				//	response.WriteErrorResponse(w, errNotifySet)
				return
			}
			_, errNotifyID := client.Collection("users").Doc(traveler).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
				"id": notificationDoc.ID,
			}, firestore.MergeAll)
			if errNotifyID != nil {
				fmt.Println(errNotifyID)
				//response.WriteErrorResponse(w, errNotifyID)
				return
			}

			iter := client.Collection("users").Doc(traveler).Collection("devices").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Println(err)
					//response.WriteErrorResponse(w, err)
					return
				}

				var token types.Token
				doc.DataTo(&token)
				tokens = append(tokens, token.Token)

			}
		}
	}

	if len(tokens) > 0 {

		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"type":             "user_trip",
			"notificationData": navigateData,
			"user":             updatedBy,
			"msg":              updatedBy.DisplayName + " removed " + destination.DestinationName + " from " + trip.Name,
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  tokens,
			ContentAvailable: true,
			CollapseKey:      "Destination removed!",
			Priority:         fcm.PriorityNormal,
			Notification: fcm.Notification{
				Title:       "Destination removed!",
				Body:        updatedBy.DisplayName + " removed " + destination.DestinationName + " from " + trip.Name,
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				//Badge: user.PhotoURL,
			},
		})
		if err != nil {
			fmt.Println("Notification send err")
			fmt.Println(err)
			//response.WriteErrorResponse(w, err)
		}
		fmt.Println("Status Code   :", notification.StatusCode)
		fmt.Println("Success       :", notification.Success)
		fmt.Println("Fail          :", notification.Fail)
		fmt.Println("Canonical_ids :", notification.CanonicalIDs)
		fmt.Println("Topic MsgId   :", notification.MsgID)
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
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
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		var destination types.Destination
		doc.DataTo(&destination)
		dest = append(dest, destination)
	}

	for i := 0; i < len(dest); i++ {
		go func(tripID string, destination types.Destination) {
			iter := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Collection("flights_accomodations").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Println(err)
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
		}(tripID, dest[i])
	}

	var travelers []string
	iterTrav := client.Collection("trips").Doc(tripID).Collection("travelers").Documents(ctx)
	for {
		doc, err := iterTrav.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		dataMap := doc.Ref.ID
		travelers = append(travelers, dataMap)
	}

	for i := 0; i < len(travelers); i++ {
		go func(tripID string, travelerID string) {
			deleteRes, errDelete := client.Collection("trips").Doc(tripID).Collection("travelers").Doc(travelerID).Delete(ctx)
			if errDelete != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errDelete
				return
			}

			travelDeleteChannel <- deleteRes
		}(tripID, travelers[i])
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

				comments := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Collection("itinerary_items").Doc(item.ID).Collection("comments").Documents(ctx)
				for {
					commentDoc, errCom := comments.Next()
					if errCom == iterator.Done {
						break
					}
					var comment itineraries.Comment
					commentDoc.DataTo(&comment)
					_, errComDelete := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Collection("itinerary_items").Doc(item.ID).Collection("comments").Doc(comment.ID).Delete(ctx)
					if errComDelete != nil {
						// Handle any errors in an appropriate way, such as returning them.
						fmt.Println(errComDelete)
						response.WriteErrorResponse(w, errComDelete)
						return
					}
				}
				_, errItem := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Collection("itinerary_items").Doc(item.ID).Delete(ctx)
				if errItem != nil {
					// Handle any errors in an appropriate way, such as returning them.
					fmt.Println(errItem)
					response.WriteErrorResponse(w, errItem)
					return
				}
			}

			_, errDay := client.Collection("itineraries").Doc(itinerary.ID).Collection("days").Doc(day.ID).Delete(ctx)
			if errDay != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(errDay)
				response.WriteErrorResponse(w, errDay)
				return
			}
		}
		_, err3 := client.Collection("itineraries").Doc(itinerary.ID).Delete(ctx)
		if err3 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(err3)
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
		response.WriteErrorResponse(w, errDelete)
		return
	}

	total := len(dest) + len(travelers)

	for i := 0; i < total; i++ {
		select {
		case <-destDeleteChannel:
			count = count + 1
		case <-travelDeleteChannel:
			travCount = travCount + 1
		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
	}

	deleteData := map[string]interface{}{
		"destinations_deleted": count,
		"travelers_deleted":    travCount,
		"success":              true,
	}

	response.Write(w, deleteData, http.StatusOK)
	return
}
