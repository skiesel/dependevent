package dependevent

import (
	"appengine"
	"appengine/user"
	"encoding/json"
	"net/http"
	"strconv"
)

func editEvent(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u == nil {
		showLogin(w, r, c)
		return
	}

	account := getAccountByEmail(u.Email, c)

	eventIDString := r.URL.Query().Get("id")
	eventID := -1

	if eventIDString != "" {
		var err error
		eventID, err = strconv.Atoi(eventIDString)
		if err != nil {
			showError(w, http.StatusInternalServerError, err, c)
			return
		}
	}

	var event *Event

	if eventID >= 0 {
		event = account.retrieveEventDeep(nil, eventID, c)
		if event == nil {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		}
	}
	if event == nil {
		event = &Event{
			ID: -1,
			SiblingID: -1,
			ChildID: -1,
			Name:        "New Event",
			Description: "New Event Description",
		}
	}

	encodedEventBytes, err := json.Marshal(event)
	if err != nil {
		showError(w, http.StatusInternalServerError, err, c)
		return
	}

	err = pages.ExecuteTemplate(w, "editevent.template", string(encodedEventBytes))
	if err != nil {
		showError(w, http.StatusInternalServerError, err, c)
	}
}
