package itineraries

import (
	//"encoding/json" //"sort"
	"encoding/json"
	"fmt"
	"net/http" //"net/url"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/asqwrd/trotter-api/types"
	"github.com/asqwrd/trotter-api/utils"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"googlemaps.github.io/maps"
	"gopkg.in/maddevsio/fcm.v1"
)

func collectionHandler(iter *firestore.DocumentIterator, client *firestore.Client) (map[string]interface{}, error) {
	ctx := context.Background()
	var itineraries = make([]Itinerary, 0)
	daysChannel := make(chan DaysChannel)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var itinerary Itinerary
		doc.DataTo(&itinerary)
		itineraries = append(itineraries, itinerary)
	}

	for i := 0; i < len(itineraries); i++ {
		go func(index int) {
			var days []Day
			iter := client.Collection("itineraries").Doc(itineraries[index].ID).Collection("days").OrderBy("day", firestore.Asc).Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				var day Day
				var itineraryItem ItineraryItem
				var itineraryItems = make([]ItineraryItem, 0)
				doc.DataTo(&day)
				iterItems := doc.Ref.Collection("itinerary_items").Documents(ctx)
				for {
					i10ItemsDoc, err := iterItems.Next()
					if err == iterator.Done {
						break
					}
					i10ItemsDoc.DataTo(&itineraryItem)
					if itineraryItem.Poi != nil && len(itineraryItem.Poi.Images) > 0 {
						itineraryItem.Image = itineraryItem.Poi.Images[0].Sizes.Medium.URL
					}
					itineraryItems = append(itineraryItems, itineraryItem)

				}
				day.ItineraryItems = itineraryItems
				days = append(days, day)
			}

			res := new(DaysChannel)
			res.Days = days
			res.Index = index
			daysChannel <- *res
		}(i)
	}
	for i := 0; i < len(itineraries); i++ {
		select {
		case res := <-daysChannel:
			if res.Error != nil {
				return nil, res.Error
			}
			itineraries[res.Index].Days = res.Days
		}
	}

	totalDoc, errTotal := client.Collection("itineraries").Doc("total_public").Get(ctx)
	if errTotal != nil {
		return nil, errTotal
	}

	total := totalDoc.Data()

	return map[string]interface{}{
		"itineraries":  itineraries,
		"total_public": total,
	}, nil
}

// GetItineraries function
func GetItineraries(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	//colorChannel := make(chan places.ColorChannel)
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

	var itinerariesCollection firestore.Query
	if len(q.Get("last")) > 0 {
		itinerariesCollection = client.Collection("itineraries").OrderBy("id", firestore.Asc).StartAfter(q.Get("last")).Limit(10)
	} else {
		itinerariesCollection = client.Collection("itineraries").OrderBy("id", firestore.Asc).Limit(10)
	}
	var queries firestore.Query
	var itr *firestore.DocumentIterator

	if len(q.Get("public")) > 0 {
		queries = itinerariesCollection.Where("public", "==", true)
	}

	if len(q.Get("destination")) > 0 {
		notNil := utils.CheckFirestoreQueryResults(ctx, queries)
		if notNil == true {
			queries = queries.Where("destination", "==", q.Get("destination"))
		} else {
			queries = itinerariesCollection.Where("destination", "==", q.Get("destination"))
		}
	}
	if len(q.Get("user_id")) > 0 {
		queries = queries.Where("travelers", "array-contains", q.Get("user_id"))

	}

	notNil := utils.CheckFirestoreQueryResults(ctx, queries)

	if notNil == true {
		itr = queries.Documents(ctx)
	} else {
		itr = itinerariesCollection.Documents(ctx)
	}

	itineraryData, errData := collectionHandler(itr, client)
	if errData != nil {
		fmt.Println(errData)
		response.WriteErrorResponse(w, errData)
		return
	}

	fmt.Println("Got Itineraries")

	response.Write(w, itineraryData, http.StatusOK)
	return
}

//Private getItinerary funtion
func getItinerary(itineraryID string) (map[string]interface{}, error) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	errorChannel := make(chan error)
	destinationChannel := make(chan map[string]interface{})
	app, err := firebase.NewApp(ctx, nil, sa)
	fmt.Println("Got Itinerary")
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	snap, err := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var itinerary Itinerary
	snap.DataTo(&itinerary)
	if itinerary.StartLocation == nil {
		_, errUpdate := client.Collection("itineraries").Doc(itineraryID).Set(ctx, map[string]interface{}{
			"start_location": StartLocation{Location: &LocationSave{Latitude: itinerary.Location.Latitude, Longitude: itinerary.Location.Longitude}, Name: "City center"},
		}, firestore.MergeAll)
		if errUpdate != nil {
			fmt.Println(errUpdate)
			return nil, err
		}
	}

	go func(id string) {
		parent, err := triposo.GetLocation(id)
		if err != nil {
			errorChannel <- err
		}
		parentParam := *parent
		var destination triposo.InternalPlace
		var colors *places.Colors
		if len(parentParam) > 0 {
			destination = places.FromTriposoPlace(parentParam[0], parentParam[0].Type)
			colorsRes, err := places.GetColor(destination.Image)
			if err != nil {
				errorChannel <- err
				return
			}
			colors = colorsRes
		}

		destinationChannel <- map[string]interface{}{
			"colors":      colors,
			"destination": destination,
		}

	}(itinerary.Destination)
	var nestedItineraries []LinkedItinerary
	nestedItr := client.Collection("itineraries").Where("trip_id", "==", itinerary.TripID).Where("start_date", ">", 0).Documents(ctx)
	for {
		nestDoc, errNest := nestedItr.Next()
		if errNest == iterator.Done {
			break
		}
		if errNest != nil {
			fmt.Println(errNest)
			return nil, errNest
		}
		var nestItinerary Itinerary
		errConvert := nestDoc.DataTo(&nestItinerary)
		if errConvert != nil {
			fmt.Println(errConvert)
			return nil, errConvert

		}
		if nestItinerary.ID != itineraryID && nestItinerary.StartDate >= itinerary.StartDate && nestItinerary.EndDate <= itinerary.EndDate {

			var daysCount = 0

			endtm := time.Unix(nestItinerary.EndDate, 0)
			starttm := time.Unix(nestItinerary.StartDate, 0)

			diff := endtm.Sub(starttm)
			daysCount = int(diff.Hours()/24) + 1
			parentStartTime := time.Unix(itinerary.StartDate, 0)
			startDiff := starttm.Sub(parentStartTime)
			startDay := int(startDiff.Hours() / 24)

			linkItinerary := LinkedItinerary{
				NumberOfDays: daysCount,
				Itinerary:    nestItinerary,
				StartDay:     startDay,
			}

			nestedItineraries = append(nestedItineraries, linkItinerary)
		}
	}

	var days []Day
	iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").OrderBy("day", firestore.Asc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		var day Day
		var itineraryItems = make([]ItineraryItem, 0)
		doc.DataTo(&day)
		iterItems := doc.Ref.Collection("itinerary_items").Documents(ctx)
		for {
			i10ItemsDoc, err := iterItems.Next()
			if err == iterator.Done {
				break
			}
			var itineraryItem ItineraryItem
			i10ItemsDoc.DataTo(&itineraryItem)
			if itineraryItem.Poi != nil && len(itineraryItem.Poi.Images) > 0 {
				itineraryItem.Image = itineraryItem.Poi.Images[0].Sizes.Medium.URL
			}
			itineraryItems = append(itineraryItems, itineraryItem)

		}
		day.ItineraryItems = itineraryItems
		days = append(days, day)
	}

	for _, link := range nestedItineraries {
		fmt.Println(link.NumberOfDays)
		startIndex := link.StartDay
		var destination types.Destination
		for i := 0; i < link.NumberOfDays; i++ {
			iter := client.Collection("trips").Doc(itinerary.TripID).Collection("destinations").Where("destination_id", "==", link.Itinerary.Destination).Documents(ctx)
			for {
				destSnap, errDest := iter.Next()
				if errDest == iterator.Done {
					break
				}
				if errDest != nil {
					fmt.Println(errDest)
					return nil, errDest
				}
				destSnap.DataTo(&destination)
				break
			}
			link.Destination = destination
			days[startIndex].LinkedItinerary = &link
			startIndex++
		}
	}

	itinerary.Days = days

	var destination triposo.InternalPlace
	var color string
	for i := 0; i < 1; i++ {
		select {
		case res := <-destinationChannel:
			destination = res["destination"].(triposo.InternalPlace)
			colors := res["colors"].(*places.Colors)
			if len(colors.Vibrant) > 0 {
				color = colors.Vibrant
			} else if len(colors.Muted) > 0 {
				color = colors.Muted
			} else if len(colors.LightVibrant) > 0 {
				color = colors.LightVibrant
			} else if len(colors.LightMuted) > 0 {
				color = colors.LightMuted
			} else if len(colors.DarkVibrant) > 0 {
				color = colors.DarkVibrant
			} else if len(colors.DarkMuted) > 0 {
				color = colors.DarkMuted
			}
		case err := <-errorChannel:
			return nil, err
		}
	}
	hotels := []map[string]interface{}{}
	if len(itinerary.TripID) > 0 {
		fmt.Println(destination.ID)
		destItr := client.Collection("trips").Doc(itinerary.TripID).Collection("destinations").Where("destination_id", "==", destination.ID).Documents(ctx)
		for {
			destDoc, errDest := destItr.Next()
			if errDest == iterator.Done {
				break
			}
			if errDest != nil {
				return nil, errDest
			}

			detailsIter := client.Collection("trips").Doc(itinerary.TripID).Collection("destinations").Doc(destDoc.Ref.ID).Collection("flights_accomodations").Documents(ctx)
			for {
				detailsDoc, errDetail := detailsIter.Next()
				if errDetail == iterator.Done {
					break
				}
				if errDetail != nil {
					return nil, errDetail
				}
				var flightAccomodation types.FlightsAndAccomodations
				detailsDoc.DataTo(&flightAccomodation)
				for _, segment := range flightAccomodation.Segments {
					if segment.Type == "Hotel" {
						travelers := []types.User{}
						for _, id := range flightAccomodation.Travelers {
							userDoc, errUser := client.Collection("users").Doc(id).Get(ctx)
							if errUser != nil {
								return nil, errUser
							}
							var user types.User
							userDoc.DataTo(&user)
							travelers = append(travelers, user)

						}
						detail := map[string]interface{}{
							"hotel":     segment,
							"travelers": travelers,
						}
						hotels = append(hotels, detail)
					}
				}

			}

		}

	}

	itineraryData := map[string]interface{}{
		"itinerary":   itinerary,
		"destination": destination,
		"hotels":      hotels,
		"color":       color,
	}
	return itineraryData, err
}

//GetItinerary function
func GetItinerary(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	//tripChannel := make(chan Trip)
	itineraryData, err := getItinerary(itineraryID)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	response.Write(w, itineraryData, http.StatusOK)
	return
}

//GetComments function
func GetComments(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	itineraryItemID := mux.Vars(r)["itineraryItemId"]
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var q *url.Values
	args := r.URL.Query()
	q = &args

	app, errApp := firebase.NewApp(ctx, nil, sa)
	if errApp != nil {
		fmt.Println(errApp)
		response.WriteErrorResponse(w, errApp)
		return
	}

	client, errClient := app.Firestore(ctx)
	if errClient != nil {
		fmt.Println(errClient)
		response.WriteErrorResponse(w, errClient)
		return
	}

	defer client.Close()
	comments := []Comment{}
	var docs *firestore.DocumentIterator
	if len(q.Get("last")) > 0 {
		timeStamp, errParse := strconv.ParseInt(q.Get("last"), 10, 64)
		if errParse != nil {
			fmt.Println(errParse)
			response.WriteErrorResponse(w, errParse)
			return
		}
		docs = client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").OrderBy("created_at", firestore.Asc).StartAfter(timeStamp).Limit(20).Documents(ctx)
	} else {
		docs = client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").OrderBy("created_at", firestore.Asc).Limit(20).Documents(ctx)
	}
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}
		var comment Comment
		doc.DataTo(&comment)
		comments = append(comments, comment)
	}

	total := map[string]interface{}{
		"total": 0,
		"id":    "total_comments",
	}
	totalDoc, errTotal := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Doc("total_comments").Get(ctx)
	if errTotal != nil {
		_, errSetTotal := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Doc("total_comments").Set(ctx, total)
		if errSetTotal != nil {
			fmt.Println(errSetTotal)
			response.WriteErrorResponse(w, errSetTotal)
			return
		}

	} else {
		total = totalDoc.Data()
	}

	commentsData := map[string]interface{}{
		"comments":       comments,
		"total_comments": total,
	}

	response.Write(w, commentsData, http.StatusOK)
	return
}

//AddComment function
func AddComment(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	itineraryItemID := mux.Vars(r)["itineraryItemId"]
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var q *url.Values
	args := r.URL.Query()
	q = &args

	decoder := json.NewDecoder(r.Body)
	var comment Comment
	err := decoder.Decode(&comment)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	app, errApp := firebase.NewApp(ctx, nil, sa)
	if errApp != nil {
		fmt.Println(errApp)
		response.WriteErrorResponse(w, errApp)
		return
	}

	client, errClient := app.Firestore(ctx)
	if errClient != nil {
		fmt.Println(errClient)
		response.WriteErrorResponse(w, errClient)
		return
	}

	defer client.Close()

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(q.Get("tripId")).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}
	tripDoc.DataTo(&trip)

	var deviceIds []string
	devicesItr := client.Collection("users").Doc(comment.User.UID).Collection("devices").Documents(ctx)
	for {
		device, errDevice := devicesItr.Next()
		if errDevice == iterator.Done {
			break
		}
		deviceIds = append(deviceIds, device.Ref.ID)
	}

	var itinerary Itinerary
	itineraryDoc, errI10 := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if errI10 != nil {
		fmt.Println(errI10)
		response.WriteErrorResponse(w, errI10)
		return
	}
	itineraryDoc.DataTo(&itinerary)

	var itineraryItem ItineraryItem
	item, errItem := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Get(ctx)
	if errItem != nil {
		fmt.Println(errItem)
		response.WriteErrorResponse(w, errItem)
		return
	}

	item.DataTo(&itineraryItem)

	doc, _, errComment := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Add(ctx, comment)
	if errComment != nil {
		fmt.Println(errComment)
		response.WriteErrorResponse(w, errComment)
		return
	}

	_, errCommentID := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
	}, firestore.MergeAll)
	if errCommentID != nil {
		fmt.Println(errCommentID)
		response.WriteErrorResponse(w, errCommentID)
		return
	}

	_, errTotalUpdate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Doc("total_comments").Update(ctx, []firestore.Update{
		{Path: "total", Value: firestore.Increment(1)},
	})
	if errTotalUpdate != nil {
		fmt.Println(errTotalUpdate)
		response.WriteErrorResponse(w, errTotalUpdate)
		return
	}

	totalDoc, errTotal := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Doc("total_comments").Get(ctx)
	if errTotal != nil {
		fmt.Println(errTotal)
		response.WriteErrorResponse(w, errTotal)
		return
	}

	com, errCommentGet := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("comments").Doc(doc.ID).Get(ctx)
	if errCommentGet != nil {
		fmt.Println(errCommentGet)
		response.WriteErrorResponse(w, errCommentGet)
		return
	}

	var commentData Comment
	com.DataTo(&commentData)

	total := totalDoc.Data()

	c := fcm.NewFCM(types.SERVER_KEY)
	var tokens []string
	navigateData := map[string]interface{}{
		"itineraryId":       itineraryID,
		"dayId":             dayID,
		"itineraryItemId":   itineraryItemID,
		"tripId":            q.Get("tripId"),
		"level":             "comments",
		"startLocation":     itinerary.StartLocation.Location,
		"itineraryName":     itinerary.Name,
		"itineraryItemName": itineraryItem.Poi.Name,
	}

	for _, traveler := range trip.Group {
		if traveler != comment.User.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_comment",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": comment.User, "subject": comment.User.DisplayName + " added a comment"},
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
				if !utils.Contains(deviceIds, token.DeviceID) {
					tokens = append(tokens, token.Token)
				}

			}
		}
	}
	if len(tokens) > 0 {

		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"type":             "user_comment",
			"notificationData": navigateData,
			"user":             comment.User,
			"msg":              comment.User.DisplayName + " added a comment",
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  tokens,
			CollapseKey:      "New comment",
			ContentAvailable: true,
			Priority:         fcm.PriorityNormal,
			Notification: fcm.Notification{
				Title:       "New comment in itinerary for " + trip.Name,
				Body:        comment.User.DisplayName + " added a comment",
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

	commentsData := map[string]interface{}{
		"comment":        commentData,
		"total_comments": total,
	}

	response.Write(w, commentsData, http.StatusOK)
	return
}

//ChangeStartLocation function
func ChangeStartLocation(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	decoder := json.NewDecoder(r.Body)
	var location StartLocation
	err := decoder.Decode(&location)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	app, errApp := firebase.NewApp(ctx, nil, sa)
	fmt.Println("Change start location")
	if errApp != nil {
		fmt.Println(errApp)
		response.WriteErrorResponse(w, errApp)
		return
	}

	client, errClient := app.Firestore(ctx)
	if errClient != nil {
		fmt.Println(errClient)
		response.WriteErrorResponse(w, errClient)
		return
	}

	defer client.Close()
	_, errUpdate := client.Collection("itineraries").Doc(itineraryID).Set(ctx, map[string]interface{}{
		"start_location": location,
	}, firestore.MergeAll)
	if errUpdate != nil {
		fmt.Println(errUpdate)
		response.WriteErrorResponse(w, errUpdate)
		return
	}

	response.Write(w, map[string]interface{}{
		"start_location": location,
		"success":        true,
	}, http.StatusOK)
	return
}

func optimizeItinerary(itineraryItems []ItineraryItem, matrix maps.DistanceMatrixResponse) []ItineraryItem {
	var rows = matrix.Rows
	var slice []map[string]interface{}
	var visited = make(map[string]interface{})

	for i := 0; i < len(itineraryItems); i++ {

		var colSlice []map[string]interface{}
		for j := 0; j < len(rows[i].Elements); j++ {
			colSlice = append(colSlice, map[string]interface{}{
				"element": rows[i].Elements[j],
				"item":    itineraryItems[j],
			})
		}
		slice = append(slice, map[string]interface{}{
			"columns": colSlice,
			"item":    itineraryItems[i],
			"index":   i,
		})

	}
	var queue []map[string]interface{}
	queue = append(queue, slice[0])
	visited[slice[0]["item"].(ItineraryItem).ID] = true
	var output []ItineraryItem
	for len(queue) > 0 {
		var read = queue[0]
		queue = queue[1:]
		output = append(output, read["item"].(ItineraryItem))
		var min int
		var next map[string]interface{}
		var nextID string
		var elements = read["columns"].([]map[string]interface{})
		var travel *maps.DistanceMatrixElement

		for k := 0; k < len(elements); k++ {
			var columnDistance = elements[k]["element"].(*maps.DistanceMatrixElement).Distance.Meters
			var col = elements[k]["item"].(ItineraryItem)
			if (min == 0 || min > columnDistance) && visited[col.ID] == nil {
				min = columnDistance
				nextID = col.ID
				travel = elements[k]["element"].(*maps.DistanceMatrixElement)
			}
		}
		if visited[nextID] == nil {
			for i := 0; i < len(slice); i++ {
				if slice[i]["item"].(ItineraryItem).ID == nextID {
					var item = slice[i]["item"].(ItineraryItem)
					item.Travel = *travel
					next = slice[i]
					next["item"] = item

					visited[nextID] = true
					queue = append(queue, next)
					break
				}
			}
		}

	}
	return output

}

//TogglePublic function
func TogglePublic(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	app, errApp := firebase.NewApp(ctx, nil, sa)
	fmt.Println("toggle public")

	if errApp != nil {
		fmt.Println(errApp)
		response.WriteErrorResponse(w, errApp)
		return
	}

	client, errClient := app.Firestore(ctx)
	if errClient != nil {
		fmt.Println(errClient)
		response.WriteErrorResponse(w, errClient)
		return
	}
	defer client.Close()

	doc, errDoc := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if errDoc != nil {
		fmt.Println(errDoc)
		response.WriteErrorResponse(w, errDoc)
		return
	}

	var itinerary Itinerary
	doc.DataTo(&itinerary)

	_, errUpdate := client.Collection("itineraries").Doc(itineraryID).Set(ctx, map[string]interface{}{
		"public": !itinerary.Public,
	}, firestore.MergeAll)
	if errUpdate != nil {
		fmt.Println(errUpdate)
		response.WriteErrorResponse(w, errUpdate)
		return
	}

	if !itinerary.Public == true {
		_, errTotalUpdate := client.Collection("itineraries").Doc("total_public").Update(ctx, []firestore.Update{
			{Path: "count", Value: firestore.Increment(1)},
		})
		if errTotalUpdate != nil {
			fmt.Println(errTotalUpdate)
			response.WriteErrorResponse(w, errTotalUpdate)
			return
		}
	} else {
		_, errTotalUpdate := client.Collection("itineraries").Doc("total_public").Update(ctx, []firestore.Update{
			{Path: "count", Value: firestore.Increment(-1)},
		})
		if errTotalUpdate != nil {
			fmt.Println(errTotalUpdate)
			response.WriteErrorResponse(w, errTotalUpdate)
			return
		}
	}

	response.Write(w, map[string]interface{}{
		"success": true,
	}, http.StatusOK)
	return

}

//ToggleVisited function
func ToggleVisited(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	itemID := mux.Vars(r)["itineraryItemId"]
	q := r.URL.Query()

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	decoder := json.NewDecoder(r.Body)
	var itineraryItem ItineraryItem
	err := decoder.Decode(&itineraryItem)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}
	fmt.Println(itineraryItem.Time)

	app, errApp := firebase.NewApp(ctx, nil, sa)
	fmt.Println("toggle visited")
	if errApp != nil {
		fmt.Println(errApp)
		response.WriteErrorResponse(w, errApp)
		return
	}

	client, errClient := app.Firestore(ctx)
	if errClient != nil {
		fmt.Println(errClient)
		response.WriteErrorResponse(w, errClient)
		return
	}
	timeSpent := Time{Value: "", Unit: ""}
	if len(itineraryItem.Time.Value) > 0 && len(itineraryItem.Time.Unit) > 0 {
		timeSpent = itineraryItem.Time
	}

	defer client.Close()
	_, errUpdate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itemID).Set(ctx, map[string]interface{}{
		"visited": !itineraryItem.Visited,
		"time":    timeSpent,
	}, firestore.MergeAll)
	if errUpdate != nil {
		fmt.Println(errUpdate)
		response.WriteErrorResponse(w, errUpdate)
		return
	}

	getDay(w, r, nil, false, true)

	var user types.User
	fmt.Println("User ID")
	fmt.Println(q.Get("userId"))
	userDoc, errUser := client.Collection("users").Doc(q.Get("userId")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	userDoc.DataTo(&user)

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(q.Get("tripId")).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}
	tripDoc.DataTo(&trip)

	var deviceIds []string
	devicesItr := client.Collection("users").Doc(user.UID).Collection("devices").Documents(ctx)
	for {
		device, errDevice := devicesItr.Next()
		if errDevice == iterator.Done {
			break
		}
		deviceIds = append(deviceIds, device.Ref.ID)
	}

	var itinerary Itinerary
	itineraryDoc, errI10 := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if errI10 != nil {
		fmt.Println(errI10)
		response.WriteErrorResponse(w, errI10)
		return
	}
	itineraryDoc.DataTo(&itinerary)

	var tokens []string
	navigateData := map[string]interface{}{
		"itineraryId":       itineraryID,
		"dayId":             dayID,
		"itineraryItemId":   itemID,
		"tripId":            q.Get("tripId"),
		"level":             "itinerary/day/edit",
		"startLocation":     itinerary.StartLocation.Location,
		"itineraryName":     itinerary.Name,
		"itineraryItemName": itineraryItem.Poi.Name,
	}

	msg := user.DisplayName + " marked " + itineraryItem.Poi.Name + " as visited in " + itinerary.Name
	if !itineraryItem.Visited == false {
		msg = user.DisplayName + " changed " + itineraryItem.Poi.Name + " back to not visited in " + itinerary.Name
	}

	for _, traveler := range trip.Group {
		if traveler != user.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_visited",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": user, "subject": msg},
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
				if !utils.Contains(deviceIds, token.DeviceID) {
					tokens = append(tokens, token.Token)
				}

			}
		}
	}

	utils.SendNotification(navigateData, msg, user, "visited", "Toggled visit", tokens)

	return
}

func getDay(w http.ResponseWriter, r *http.Request, justAdded *string, optimize bool, filter bool) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]

	errorChannel := make(chan error)
	matrixChannel := make(chan maps.DistanceMatrixResponse)
	var q *url.Values
	args := r.URL.Query()
	q = &args
	latlng := q.Get("latlng")
	fmt.Println(latlng)

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

	itinerary, err := getItinerary(itineraryID)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	snap, err := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Get(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}
	day := Day{ItineraryItems: []ItineraryItem{}}
	snap.DataTo(&day)
	day.ItineraryItems = make([]ItineraryItem, 0)
	var itineraryItems = make([]ItineraryItem, 0)
	var visitedItineraryItems = make([]ItineraryItem, 0)
	docs := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Documents(ctx)
	itineraryItemsChannel := make(chan map[string]interface{})
	go func(docs *firestore.DocumentIterator) {
		defer close(itineraryItemsChannel)
		var itineraryItems = make([]ItineraryItem, 0)
		var visitedItineraryItems = make([]ItineraryItem, 0)
		for {
			i10ItemDocs, err := docs.Next()
			if err == iterator.Done {
				break
			}
			var itineraryItem ItineraryItem
			i10ItemDocs.DataTo(&itineraryItem)

			descriptionChannel := make(chan []TravelerDescription)
			go func(itineraryItemID string) {
				defer close(descriptionChannel)
				docs := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("traveler_descriptions").Documents(ctx)
				descriptions := []TravelerDescription{}
				for {
					descDoc, err := docs.Next()
					if err == iterator.Done {
						break
					}
					var description TravelerDescription
					descDoc.DataTo(&description)
					descriptions = append(descriptions, description)

				}
				descriptionChannel <- descriptions

			}(itineraryItem.ID)

			for res := range descriptionChannel {
				itineraryItem.TravelerDescriptions = res
			}

			totalDoc, errTotal := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(i10ItemDocs.Ref.ID).Collection("comments").Doc("total_comments").Get(ctx)
			if errTotal != nil {
				itineraryItem.TotalComments = 0
				if itineraryItem.Visited == true && filter == true {
					visitedItineraryItems = append(visitedItineraryItems, itineraryItem)
				} else {
					itineraryItems = append(itineraryItems, itineraryItem)
				}
			} else {
				total := totalDoc.Data()
				itineraryItem.TotalComments = total["total"].(int64)
				if itineraryItem.Visited == true && filter == true {
					visitedItineraryItems = append(visitedItineraryItems, itineraryItem)
				} else {
					itineraryItems = append(itineraryItems, itineraryItem)
				}
			}

		}
		itineraryItemsChannel <- map[string]interface{}{
			"items":   itineraryItems,
			"visited": visitedItineraryItems,
		}
	}(docs)

	for res := range itineraryItemsChannel {
		itineraryItems = res["items"].([]ItineraryItem)
		visitedItineraryItems = res["visited"].([]ItineraryItem)
	}

	googleClient, err := places.InitGoogle()
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}

	if optimize == false {
		go func(itinerary interface{}) {
			//fmt.Println(itinerary)
			var locations []string
			var chunks [][]string
			start := LocationSave{Latitude: itinerary.(Itinerary).Location.Latitude, Longitude: itinerary.(Itinerary).Location.Longitude}
			if itinerary.(Itinerary).StartLocation != nil {
				start = *itinerary.(Itinerary).StartLocation.Location
			}

			if len(latlng) > 0 {
				locations = append(locations, latlng)
			} else {
				locations = append(locations, fmt.Sprintf("%g,%g", start.Latitude, start.Longitude))
			}

			for i := 0; i < len(itineraryItems); i++ {
				location := fmt.Sprintf("%g,%g", itineraryItems[i].Poi.Location.Lat, itineraryItems[i].Poi.Location.Lng)
				if itineraryItems[i].Poi.Coordinates != nil {
					location = fmt.Sprintf("%g,%g", itineraryItems[i].Poi.Coordinates.Latitude, itineraryItems[i].Poi.Coordinates.Longitude)
				}
				if itineraryItems[i].AddedBy != nil {
					userDoc, errUser := client.Collection("users").Doc(*itineraryItems[i].AddedBy).Get(ctx)
					if errUser != nil {
						errorChannel <- errUser
						return
					}
					var user types.User
					userDoc.DataTo(&user)
					itineraryItems[i].AddedByFull = &user
				}
				locations = append(locations, location)
				if itineraryItems[i].Poi != nil && len(itineraryItems[i].Poi.DescriptionShort) > 0 {
					itineraryItems[i].Description = itineraryItems[i].Poi.DescriptionShort
				}
				if itineraryItems[i].Poi != nil && len(itineraryItems[i].Poi.Images) > 0 {
					itineraryItems[i].Image = itineraryItems[i].Poi.Images[0].Sizes.Medium.URL
					colors, err := places.GetColor(itineraryItems[i].Image)
					if err != nil {
						errorChannel <- err
						return
					}

					if len(colors.Vibrant) > 0 {
						itineraryItems[i].Color = colors.Vibrant
					} else if len(colors.Muted) > 0 {
						itineraryItems[i].Color = colors.Muted
					} else if len(colors.LightVibrant) > 0 {
						itineraryItems[i].Color = colors.LightVibrant
					} else if len(colors.LightMuted) > 0 {
						itineraryItems[i].Color = colors.LightMuted
					} else if len(colors.DarkVibrant) > 0 {
						itineraryItems[i].Color = colors.DarkVibrant
					} else if len(colors.DarkMuted) > 0 {
						itineraryItems[i].Color = colors.DarkMuted
					}
				}
			}

			batchSize := 10

			for batchSize < len(locations) {
				locations, chunks = locations[batchSize:], append(chunks, locations[0:batchSize:batchSize])
			}
			chunks = append(chunks, locations)

			var matrix *maps.DistanceMatrixResponse
			for i := 0; i < len(chunks); i++ {

				r := &maps.DistanceMatrixRequest{
					Origins:      chunks[i],
					Destinations: chunks[i],
				}
				res, err := googleClient.DistanceMatrix(ctx, r)
				if err != nil {
					fmt.Println(err)
					errorChannel <- err
					return
				}
				if matrix == nil {
					matrix = res
				} else {
					matrix.Rows = append(matrix.Rows, res.Rows...)
				}

			}
			matrixChannel <- *matrix

		}(itinerary["itinerary"])

		for i := 0; i < 1; i++ {
			select {
			case matrix := <-matrixChannel:
				var head ItineraryItem
				itineraryItems = append([]ItineraryItem{head}, itineraryItems...)
				day.ItineraryItems = optimizeItinerary(itineraryItems, matrix)
			case err := <-errorChannel:
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
		}
	} else {
		for i := 0; i < len(itineraryItems); i++ {
			if itineraryItems[i].Poi != nil && len(itineraryItems[i].Poi.Images) > 0 {
				itineraryItems[i].Image = itineraryItems[i].Poi.Images[0].Sizes.Medium.URL

				colors, err := places.GetColor(itineraryItems[i].Image)
				if err != nil {
					errorChannel <- err
					return
				}

				if len(colors.Vibrant) > 0 {
					itineraryItems[i].Color = colors.Vibrant
				} else if len(colors.Muted) > 0 {
					itineraryItems[i].Color = colors.Muted
				} else if len(colors.LightVibrant) > 0 {
					itineraryItems[i].Color = colors.LightVibrant
				} else if len(colors.LightMuted) > 0 {
					itineraryItems[i].Color = colors.LightMuted
				} else if len(colors.DarkVibrant) > 0 {
					itineraryItems[i].Color = colors.DarkVibrant
				} else if len(colors.DarkMuted) > 0 {
					itineraryItems[i].Color = colors.DarkMuted
				}
			}
		}
		day.ItineraryItems = itineraryItems

	}

	var dayData map[string]interface{}
	if filter == false {
		dayData = map[string]interface{}{
			"day":       day,
			"itinerary": itinerary,
			"justAdded": justAdded,
		}
	} else {
		for i := 0; i < len(visitedItineraryItems); i++ {
			if visitedItineraryItems[i].Poi != nil && len(visitedItineraryItems[i].Poi.Images) > 0 {
				visitedItineraryItems[i].Image = visitedItineraryItems[i].Poi.Images[0].Sizes.Medium.URL

				colors, err := places.GetColor(visitedItineraryItems[i].Image)
				if err != nil {
					errorChannel <- err
					return
				}

				if len(colors.Vibrant) > 0 {
					visitedItineraryItems[i].Color = colors.Vibrant
				} else if len(colors.Muted) > 0 {
					visitedItineraryItems[i].Color = colors.Muted
				} else if len(colors.LightVibrant) > 0 {
					visitedItineraryItems[i].Color = colors.LightVibrant
				} else if len(colors.LightMuted) > 0 {
					visitedItineraryItems[i].Color = colors.LightMuted
				} else if len(colors.DarkVibrant) > 0 {
					visitedItineraryItems[i].Color = colors.DarkVibrant
				} else if len(colors.DarkMuted) > 0 {
					visitedItineraryItems[i].Color = colors.DarkMuted
				}
			}
		}
		dayData = map[string]interface{}{
			"day":       day,
			"itinerary": itinerary,
			"justAdded": justAdded,
			"visited":   visitedItineraryItems,
		}
	}

	response.Write(w, dayData, http.StatusOK)
	return
}

//GetDay func
func GetDay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got day")
	var q *url.Values
	args := r.URL.Query()
	q = &args
	if len(q.Get("filter")) > 0 {
		getDay(w, r, nil, false, true)
	} else {
		getDay(w, r, nil, false, false)
	}

	return

}

// CreateItineraryHelper function
func CreateItineraryHelper(tripID string, destinationID string, itinerary Itinerary, numOfDays *int) (map[string]interface{}, error) {
	dayChannel := make(chan string)
	errorChannel := make(chan error)
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	doc, _, errCreate := client.Collection("itineraries").Add(ctx, itinerary)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, errCreate
	}

	_, err2 := client.Collection("itineraries").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id":     doc.ID,
		"public": false,
	}, firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, err2
	}

	_, errTrip := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Set(ctx, map[string]interface{}{
		"itinerary_id": doc.ID,
	}, firestore.MergeAll)
	if errTrip != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, errTrip
	}

	//Adding days
	var daysCount int = 0

	if numOfDays != nil && itinerary.EndDate == 0 && itinerary.StartDate == 0 {
		daysCount = *numOfDays
	} else {

		endtm := time.Unix(itinerary.EndDate, 0)
		starttm := time.Unix(itinerary.StartDate, 0)

		diff := endtm.Sub(starttm)
		daysCount = int(diff.Hours()/24) + 1 //include first day
	}

	for i := 0; i < daysCount; i++ {
		go func(index int, itineraryID string) {
			daydoc, _, errCreate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Add(ctx, map[string]interface{}{
				"day": index,
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

			dayChannel <- doc.ID
		}(i, doc.ID)
	}
	var dayIDS []string
	for i := 0; i < daysCount; i++ {
		select {
		case res := <-dayChannel:
			dayIDS = append(dayIDS, res)
		case err := <-errorChannel:
			return nil, err
		}
	}

	id := doc.ID
	return map[string]interface{}{
		"id": id,
	}, nil
}

//CreateItinerary func
func CreateItinerary(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var itinerary ItineraryRes
	err := decoder.Decode(&itinerary)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	itineraryData, err := CreateItineraryHelper(itinerary.Itinerary.TripID, itinerary.TripDestinationID, itinerary.Itinerary, nil)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}
	response.Write(w, itineraryData, http.StatusOK)
	return
}

//SaveDescription func
func SaveDescription(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	itineraryItemID := mux.Vars(r)["itineraryItemId"]
	q := r.URL.Query()
	decoder := json.NewDecoder(r.Body)
	var travelerDescription TravelerDescription
	err := decoder.Decode(&travelerDescription)
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

	fmt.Println("Save description")

	_, errCreate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("traveler_descriptions").Doc(travelerDescription.User.UID).Set(ctx, travelerDescription)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err)
		return
	}

	iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Collection("traveler_descriptions").OrderBy("created_at", firestore.Desc).Documents(ctx)

	descriptions := []TravelerDescription{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		var description TravelerDescription
		doc.DataTo(&description)
		descriptions = append(descriptions, description)

	}

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(q.Get("tripId")).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}
	tripDoc.DataTo(&trip)

	var deviceIds []string
	devicesItr := client.Collection("users").Doc(travelerDescription.User.UID).Collection("devices").Documents(ctx)
	for {
		device, errDevice := devicesItr.Next()
		if errDevice == iterator.Done {
			break
		}
		deviceIds = append(deviceIds, device.Ref.ID)
	}

	var itinerary Itinerary
	itineraryDoc, errI10 := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if errI10 != nil {
		fmt.Println(errI10)
		response.WriteErrorResponse(w, errI10)
		return
	}
	itineraryDoc.DataTo(&itinerary)

	var itineraryItem ItineraryItem
	item, errItem := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Get(ctx)
	if errItem != nil {
		fmt.Println(errItem)
		response.WriteErrorResponse(w, errItem)
		return
	}

	item.DataTo(&itineraryItem)

	var tokens []string
	navigateData := map[string]interface{}{
		"itineraryId":       itineraryID,
		"dayId":             dayID,
		"itineraryItemId":   itineraryItemID,
		"tripId":            q.Get("tripId"),
		"level":             "itinerary/day/edit",
		"startLocation":     itinerary.StartLocation.Location,
		"itineraryName":     itinerary.Name,
		"itineraryItemName": itineraryItem.Poi.Name,
	}

	msg := travelerDescription.User.DisplayName + " added a description to " + itineraryItem.Poi.Name + " in " + itinerary.Name

	for _, traveler := range trip.Group {
		if traveler != travelerDescription.User.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_description",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": travelerDescription.User, "subject": msg},
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
				if !utils.Contains(deviceIds, token.DeviceID) {
					tokens = append(tokens, token.Token)
				}

			}
		}
	}

	utils.SendNotification(navigateData, msg, travelerDescription.User, "description", "A new description added", tokens)

	response.Write(w, map[string]interface{}{
		"descriptions": descriptions,
		"success":      true,
	}, http.StatusOK)
	return

}

//AddToDay func
func AddToDay(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	var id *string
	decoder := json.NewDecoder(r.Body)
	var itineraryItem ItineraryItem
	q := r.URL.Query()
	err := decoder.Decode(&itineraryItem)
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
	var itinerary Itinerary
	itineraryDoc, errItin := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if errItin != nil {
		fmt.Println(errItin)
		response.WriteErrorResponse(w, errItin)
		return
	}

	itineraryDoc.DataTo(&itinerary)

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(itinerary.TripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}

	tripDoc.DataTo(&trip)
	userID := *itineraryItem.AddedBy
	if len(q.Get("userId")) > 0 && q.Get("userId") != "null" {
		userID = q.Get("userId")
	}

	userDoc, errUser := client.Collection("users").Doc(userID).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var addedBy types.User
	userDoc.DataTo(&addedBy)
	var deviceIds []string
	devicesItr := client.Collection("users").Doc(userID).Collection("devices").Documents(ctx)
	for {
		device, errDevice := devicesItr.Next()
		if errDevice == iterator.Done {
			break
		}
		deviceIds = append(deviceIds, device.Ref.ID)
	}

	iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Where("poi_id", "==", itineraryItem.PoiID).Documents(ctx)
	for {
		docCheck, errCheck := iter.Next()
		if errCheck == iterator.Done {
			break
		}
		if errCheck != nil {
			fmt.Println(errCheck)
			response.WriteErrorResponse(w, errCheck)
			return
		}
		id = &docCheck.Ref.ID
	}

	if id == nil {
		doc, _, err2 := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Add(ctx, itineraryItem)
		if err2 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(err2)
			response.WriteErrorResponse(w, err2)
			return
		}

		_, errSet := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(doc.ID).Set(ctx, map[string]interface{}{
			"id": doc.ID,
		}, firestore.MergeAll)
		if errSet != nil {
			fmt.Println(errSet)
			response.WriteErrorResponse(w, errSet)
			return
		}

		_, errComments := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(doc.ID).Collection("comments").Doc("total_comments").Set(ctx, map[string]interface{}{
			"total": 0,
			"id":    "total_comments",
		}, firestore.MergeAll)

		if errComments != nil {
			fmt.Println(errComments)
			response.WriteErrorResponse(w, errComments)
			return
		}

		id = &doc.ID
	}

	if q.Get("optimize") == "true" {
		print("optimize \n")
		getDay(w, r, id, true, true)
	} else {
		print("full \n")
		getDay(w, r, id, false, true)
	}

	c := fcm.NewFCM(types.SERVER_KEY)
	var tokens []string
	navigateData := map[string]interface{}{
		"itineraryId":   itineraryID,
		"dayId":         dayID,
		"startLocation": itinerary.StartLocation.Location,
		"level":         "itinerary/day/edit",
	}

	actionText := " added "
	if len(q.Get("userId")) > 0 && len(q.Get("copied")) == 0 {
		actionText = " moved "
	} else if len(q.Get("copied")) > 0 {
		actionText = " copied "
	}

	for _, traveler := range trip.Group {
		if traveler != addedBy.UID {

			notification := types.Notification{
				CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
				Type:     "user_day",
				Data:     map[string]interface{}{"navigationData": navigateData, "user": addedBy, "subject": addedBy.DisplayName + actionText + itineraryItem.Poi.Name + " to a day in " + itinerary.Name},
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
				if !utils.Contains(deviceIds, token.DeviceID) {
					tokens = append(tokens, token.Token)
				}

			}
		}
	}

	if len(tokens) > 0 {

		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"type":             "user_day",
			"notificationData": navigateData,
			"user":             addedBy,
			"msg":              addedBy.DisplayName + actionText + itineraryItem.Poi.Name + " to " + itinerary.Name,
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  tokens,
			CollapseKey:      "New place Added!",
			ContentAvailable: true,
			Priority:         fcm.PriorityNormal,
			Notification: fcm.Notification{
				Title:       "New place Added!",
				Body:        addedBy.DisplayName + actionText + itineraryItem.Poi.Name + " to " + itinerary.Name,
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

	fmt.Println("added")
	return

}

// DeleteItineraryItem function
func DeleteItineraryItem(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	place := mux.Vars(r)["placeId"]
	var q *url.Values
	args := r.URL.Query()
	q = &args

	movedPlaceID := q.Get("movedPlaceId")
	movedDayID := q.Get("movedDayId")

	fmt.Println("Delete Itinerary Item")

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

	var itineraryItem ItineraryItem

	doc, errItem := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(place).Get(ctx)
	if errItem != nil {
		fmt.Println(errItem)
		response.WriteErrorResponse(w, errItem)
		return
	}

	doc.DataTo(&itineraryItem)

	var itinerary Itinerary
	itineraryDoc, errItin := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if errItin != nil {
		fmt.Println(errItin)
		response.WriteErrorResponse(w, errItin)
		return
	}

	itineraryDoc.DataTo(&itinerary)

	var trip types.Trip
	tripDoc, errTrip := client.Collection("trips").Doc(itinerary.TripID).Get(ctx)
	if errTrip != nil {
		fmt.Println(errTrip)
		response.WriteErrorResponse(w, errTrip)
		return
	}

	tripDoc.DataTo(&trip)

	userDoc, errUser := client.Collection("users").Doc(q.Get("deletedBy")).Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var deletedBy types.User
	userDoc.DataTo(&deletedBy)
	var deviceIds []string
	devicesItr := client.Collection("users").Doc(q.Get("deletedBy")).Collection("devices").Documents(ctx)
	for {
		device, errDevice := devicesItr.Next()
		if errDevice == iterator.Done {
			break
		}
		deviceIds = append(deviceIds, device.Ref.ID)
	}

	commentsItr := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(place).Collection("comments").Documents(ctx)
	for {
		commentSnap, errComment := commentsItr.Next()
		if errComment == iterator.Done {
			break
		}
		if errComment != nil {
			fmt.Println(errComment)
			response.WriteErrorResponse(w, errComment)
			return
		}

		if len(movedDayID) > 0 && len(movedPlaceID) > 0 && commentSnap.Ref.ID != "total_comments" {
			var comment Comment
			commentSnap.DataTo(&comment)
			addDoc, _, errAdd := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(movedDayID).Collection("itinerary_items").Doc(movedPlaceID).Collection("comments").Add(ctx, comment)
			if errAdd != nil {
				fmt.Println(errAdd)
				response.WriteErrorResponse(w, errAdd)
				return
			}
			_, errCommentID := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(movedDayID).Collection("itinerary_items").Doc(movedPlaceID).Collection("comments").Doc(addDoc.ID).Set(ctx, map[string]interface{}{
				"id": addDoc.ID,
			}, firestore.MergeAll)
			if errCommentID != nil {
				fmt.Println(errCommentID)
				response.WriteErrorResponse(w, errCommentID)
				return
			}

			_, errTotalUpdate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(movedDayID).Collection("itinerary_items").Doc(movedPlaceID).Collection("comments").Doc("total_comments").Update(ctx, []firestore.Update{
				{Path: "total", Value: firestore.Increment(1)},
			})
			if errTotalUpdate != nil {
				fmt.Println(errTotalUpdate)
				response.WriteErrorResponse(w, errTotalUpdate)
				return
			}

		}

		_, errComDelete := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(place).Collection("comments").Doc(commentSnap.Ref.ID).Delete(ctx)
		if errComDelete != nil {
			fmt.Println(errComDelete)
			response.WriteErrorResponse(w, errComDelete)
			return
		}
	}

	_, errDelete := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(place).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, errDelete)
		return
	}

	if q.Get("sendNotification") == "true" {

		c := fcm.NewFCM(types.SERVER_KEY)
		var tokens []string
		navigateData := map[string]interface{}{
			"itineraryId":   itineraryID,
			"dayId":         dayID,
			"startLocation": itinerary.StartLocation.Location,
			"level":         "itinerary/day/edit",
		}

		for _, traveler := range trip.Group {
			if traveler != deletedBy.UID {
				notification := types.Notification{
					CreateAt: time.Now().UnixNano() / int64(time.Millisecond),
					Type:     "user_day",
					Data:     map[string]interface{}{"navigationData": navigateData, "user": deletedBy, "subject": deletedBy.DisplayName + " deleted " + itineraryItem.Poi.Name + " from a day in " + itinerary.Name},
					Read:     false,
				}
				notificationDoc, _, errNotifySet := client.Collection("users").Doc(traveler).Collection("notifications").Add(ctx, notification)
				if errNotifySet != nil {
					fmt.Println(errNotifySet)
					response.WriteErrorResponse(w, errNotifySet)
					return
				}
				_, errNotifyID := client.Collection("users").Doc(traveler).Collection("notifications").Doc(notificationDoc.ID).Set(ctx, map[string]interface{}{
					"id": notificationDoc.ID,
				}, firestore.MergeAll)
				if errNotifyID != nil {
					fmt.Println(errNotifyID)
					response.WriteErrorResponse(w, errNotifyID)
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
						response.WriteErrorResponse(w, err)
						return
					}

					var token types.Token
					doc.DataTo(&token)
					if !utils.Contains(deviceIds, token.DeviceID) {
						tokens = append(tokens, token.Token)
					}
				}
			}
		}

		if len(tokens) > 0 {
			data := map[string]interface{}{
				"focus":            "trips",
				"click_action":     "FLUTTER_NOTIFICATION_CLICK",
				"type":             "user_day",
				"notificationData": navigateData,
				"user":             deletedBy,
				"msg":              deletedBy.DisplayName + " deleted " + itineraryItem.Poi.Name + " from " + itinerary.Name,
			}

			notification, err := c.Send(fcm.Message{
				Data:             data,
				RegistrationIDs:  tokens,
				CollapseKey:      "Place removed from itinerary",
				ContentAvailable: true,
				Priority:         fcm.PriorityNormal,
				Notification: fcm.Notification{
					Title:       "Place removed from itinerary",
					Body:        deletedBy.DisplayName + " deleted " + itineraryItem.Poi.Name + " from " + itinerary.Name,
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

	deleteData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, deleteData, http.StatusOK)
	return
}

// TestNotification function
func TestNotification(w http.ResponseWriter, r *http.Request) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	c := fcm.NewFCM(types.SERVER_KEY)

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

	userDoc, errUser := client.Collection("users").Doc("BjPpGEpI0ERGoCnGSdalv22jbV73").Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var user types.User
	userDoc.DataTo(&user)
	iter := client.Collection("users").Doc("BjPpGEpI0ERGoCnGSdalv22jbV73").Collection("devices").Documents(ctx)
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

		navigateData := map[string]interface{}{
			"itineraryId":   "uRrfmJintOxNVhevP74p",
			"dayId":         "5lOZSEJz345IJmOZEED7",
			"startLocation": Location{Latitude: 3.143497, Longitude: 101.704094},
			"level":         "itinerary/day/edit",
		}

		var token types.Token
		doc.DataTo(&token)
		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"notificationData": navigateData,
			"user":             user,
			"msg":              "Hellow World",
			"type":             "user",
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			To:               token.Token,
			ContentAvailable: true,
			CollapseKey:      "Test notification",
			Priority:         fcm.PriorityHigh,
			Notification: fcm.Notification{
				Title:       "Hello",
				Body:        "World",
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
		fmt.Println("Results   :", notification.Results)

	}

	response.Write(w, map[string]interface{}{
		"ok": true,
	}, http.StatusOK)

	return

}
