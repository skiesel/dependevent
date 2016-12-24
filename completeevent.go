package dependevent

import (
	"appengine"
  "appengine/datastore"
	"appengine/user"
	"net/http"
	"strconv"
  "time"
)

func completeEvent(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  u := user.Current(c)

  if u == nil {
    showLogin(w, r, c)
    return
  }

  account := getAccountByEmail(u.Email, c)
  account.populateEvents(c)

  rootEventIDString := r.URL.Query().Get("rid")
  eventIDString := r.URL.Query().Get("id")

  if rootEventIDString == "" || eventIDString == "" {
    http.Redirect(w, r, "/dashboard", http.StatusFound)
  }

  rootEventID, err := strconv.Atoi(rootEventIDString)
  if err != nil {
    showError(w, http.StatusInternalServerError, err, c)
    return
  }

  eventID, err := strconv.Atoi(eventIDString)
  if err != nil {
    showError(w, http.StatusInternalServerError, err, c)
    return
  }

  accountKey := datastore.NewKey(c, "Account", account.Email, 0, nil)

  event := account.retrieveEvent(accountKey, eventID, c)

  event.Active = false
  event.Complete = true
  event.CompleteDate = time.Now()

  account.updateInDS(accountKey, event, c)

  rootEvent := account.retrieveEventDeep(accountKey, rootEventID, c)

  nextEvent := findNextEventInSeries(rootEvent)
  if nextEvent != nil {
    nextEvent.Active = true
    account.updateInDS(accountKey, nextEvent, c)

    for i, id := range account.ActiveEventIDs {
      if eventID == id {
        c.Infof("found event")

        account.ActiveEventIDs[i] = nextEvent.ID
        account.ActiveEvents[i] = nextEvent
        break
      }
    }
  } else {
    for i, id := range account.ActiveEventIDs {
      if eventID == id {
        // account.ActiveEventIDs = append(account.ActiveEventIDs[:i], account.ActiveEventIDs[i+1:]...)
        // account.ActiveEvents = append(account.ActiveEvents[:i], account.ActiveEvents[i+1:]...)
        account.ActiveEventIDs[i] = rootEventID
        account.ActiveEvents[i] = rootEvent
        break
      }
    }
  }

  account.saveInDS(c)

  http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func findNextEventInSeries(root *Event) *Event {
  if root.Child != nil {
    e := findNextEventInSeries(root.Child)
    if e != nil {
      return e
    }
  }

  if !root.Complete {
    return root
  }

  if root.Sibling != nil {
    e := findNextEventInSeries(root.Sibling)
    if e != nil {
      return e
    }
  }

  return nil
}
