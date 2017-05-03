package dependevent

import (
	"encoding/json"
	"fmt"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
	"net/http"
)

func saveEvent(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u == nil {
		showLogin(w, r, c)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var event Event
	err := decoder.Decode(&event)
	if err != nil {
		fmt.Fprintf(w, "failure")
		return
	}

	account := getAccountByEmail(u.Email, c)
	account.populateEvents(c)
	accountKey := datastore.NewKey(c, "Account", u.Email, 0, nil)

	if event.ID < 0 {
		event.ID = account.NextEventID
		account.NextEventID++

		//mark the events as active down the leftmost branch
		activeEvent := findFirstEventInSeries(&event)

		//this will populate the ids for the events which we will exploit in a second
		_, err := account.putInDS(accountKey, &event, c)
		if err != nil {
			fmt.Fprintf(w, "failure")
			return
		}

		account.RootEventIDs = append(account.RootEventIDs, event.ID)
		account.ActiveEventIDs = append(account.ActiveEventIDs, activeEvent.ID)

		account.RootEvents = append(account.RootEvents, &event)
		account.ActiveEvents = append(account.ActiveEvents, activeEvent)

		//add new event data to account
		account.saveInDS(c)
	} else {
		replaceIndex := -1
		for i, id := range account.RootEventIDs {
			if id == event.ID {
				replaceIndex = i
				break
			}
		}
		if replaceIndex >= 0 {
			account.RootEvents[replaceIndex] = &event
		}

		replaceIndex = -1
		for i, id := range account.ActiveEventIDs {
			if id == event.ID {
				replaceIndex = i
				break
			}
		}
		if replaceIndex >= 0 {
			account.ActiveEvents[replaceIndex] = &event
		}

		account.saveInDS(c)
		account.updateInDS(accountKey, &event, c)
	}

	fmt.Fprintf(w, "success")
}

func findFirstEventInSeries(root *Event) *Event {
	root.Active = true
	if root.Child != nil {
		return findFirstEventInSeries(root.Child)
	}
	return root
}
