package dependevent

import (
	"encoding/json"
	"fmt"
  "io/ioutil"
  "log"
  "net/http"
	"strings"
  "time"

	"google.golang.org/appengine/user"
	"google.golang.org/appengine"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/calendar/v3"
)

var (
	tokens = map[string]*oauth2.Token{}
	config *oauth2.Config
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

func init() {
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	config, err = google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
}

func getCachedToken(usr *user.User) (*oauth2.Token, string) {
	tok, ok := tokens[usr.ID]
  if !ok {
    return nil, getRedirectURLForToken(config)
  }
	return tok, ""
}

func setCachedToken(usr *user.User, token *oauth2.Token) {
	tokens[usr.ID] = token
}

func getClient(ctx context.Context, config *oauth2.Config, token *oauth2.Token) *http.Client {
  return config.Client(ctx, token)
}

func getRedirectURLForToken(config *oauth2.Config) string {
  return config.AuthCodeURL("state-token", oauth2.AccessTypeOnline)
}

func oauthResponse(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)

	if r.URL.Query()["code"] == nil || r.URL.Query()["code"][0] == "" {
		fmt.Fprintf(w, "token is absent")
		log.Fatalf("token is absent")
	}

	tok, err := config.Exchange(ctx, r.URL.Query()["code"][0])
  if err != nil {
		fmt.Fprintf(w, "Unable to retrieve token %v", err)
    log.Fatalf("Unable to retrieve token %v", err)
  }

	setCachedToken(u, tok)

	http.Redirect(w, r, "/push_event", 303)
}

func getCalendarService(w http.ResponseWriter, r *http.Request) *calendar.Service {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)

	if u == nil {
		showLogin(w, r, ctx)
		return nil
	}

	token, redirectUrl := getCachedToken(u)

	if token == nil {
		http.Redirect(w, r, redirectUrl, 303)
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
