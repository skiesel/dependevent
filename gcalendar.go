package dependevent

import (
	"encoding/json"
	"fmt"
  "log"
  "net/http"
	"strings"
  "time"

	"google.golang.org/appengine/user"
	"google.golang.org/appengine"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "google.golang.org/api/calendar/v3"
)

func newEvent(title, description, date string) *calendar.Event {
	jsonString := fmt.Sprintf("{ \"Summary\" :\"%s\", \"Description\" :\"%s\", \"Start\" : { \"Date\" : \"%s\" }, \"End\" : { \"Date\" : \"%s\" } }", title, description, date, date)
	decoder := json.NewDecoder(strings.NewReader(jsonString))
	var event calendar.Event
	err := decoder.Decode(&event)
	if err != nil {
		log.Println(jsonString)
		log.Fatalf("Failed to create event: %v", err)
		return nil
	}
	return &event
}

func getClient(ctx context.Context, config *oauth2.Config, token *oauth2.Token) *http.Client {
  return config.Client(ctx, token)
}

func getCalendarService(w http.ResponseWriter, r *http.Request) *calendar.Service {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)

	if u == nil {
		showLogin(w, r, ctx)
		return nil
	}

	account := getAccountByEmail(u.Email, ctx)
	token := &account.OAuthToken

	if !token.Valid() {
		startAuthProcess(w, r, "/dashboard")
		return nil
	}

	client := getClient(ctx, config, token)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve calendar Client %v", err)
	}

	return srv
}

func addEvent(w http.ResponseWriter, r *http.Request) {
	srv := getCalendarService(w, r)

	evt := newEvent("Event Title", "Event Description", "2017-05-04")

	if evt == nil {
		fmt.Fprintf(w, "fail")
		return
	}

	_, err := srv.Events.Insert("primary", evt).Do()
	if err != nil {
		fmt.Fprintf(w, "fail: %v", err)
		return
	}
	fmt.Fprintf(w, "success")
}

func listEvents(w http.ResponseWriter, r *http.Request) {
	srv := getCalendarService(w, r)

	if srv == nil {
		return
	}

  t := time.Now().Format(time.RFC3339)
  events, err := srv.Events.List("primary").ShowDeleted(false).
    SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
  if err != nil {
    fmt.Fprintf(w, "Unable to retrieve next ten of the user's events. %v", err)
  }

  fmt.Fprintf(w, "Upcoming events:")
  if len(events.Items) > 0 {
    for _, i := range events.Items {
      var when string
      // If the DateTime is an empty string the Event is an all-day Event.
      // So only Date is available.
      if i.Start.DateTime != "" {
        when = i.Start.DateTime
      } else {
        when = i.Start.Date
      }
      fmt.Fprintf(w, "%s (%s)\n", i.Summary, when)
    }
  } else {
    fmt.Fprintf(w, "No upcoming events found.\n")
  }
}
