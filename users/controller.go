package users

import (
	//"sort"
	"encoding/json"
	"fmt"
	"net/http"

	//"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/response"
	triptypes "github.com/asqwrd/trotter-api/types/trips"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

// SaveLogin function
func SaveLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var user triptypes.User
	err := decoder.Decode(&user)
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

	//Check User
	docSnap, _ := client.Collection("users").Doc(user.UID).Get(ctx)
	if docSnap.Exists() == false {

		_, errUserCreate := client.Collection("users").Doc(user.UID).Set(ctx, user)
		if errUserCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errUserCreate)
			response.WriteErrorResponse(w, errUserCreate)
			return
		}
	} else {
		response.Write(w, map[string]interface{}{
			"success": true,
			"exists":  true,
		}, http.StatusOK)
		return
	}

	fmt.Println("User Added")

	userData := map[string]interface{}{
		"success": true,
		"exists":  false,
	}

	response.Write(w, userData, http.StatusOK)
	return
}
