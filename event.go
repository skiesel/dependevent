package dependevent

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"time"
)

type Event struct {
	ID          int
	Name        string
	Description string

	Complete     bool
	CompleteDate time.Time

	Active bool

	SiblingID int
	ChildID   int

	Child   *Event `datastore:"-"`
	Sibling *Event `datastore:"-"`
}

func (account *Account) putEventInMemcache(event *Event, c context.Context) {
	memCacheItem := &memcache.Item{
		Key:    fmt.Sprintf("%s%d", account.Email, event.ID),
		Object: event,
	}
	memcache.JSON.Set(c, memCacheItem)
}

func (account *Account) putInDS(parentKey *datastore.Key, event *Event, c context.Context) (bool, error) {
	modified := false
	if event.ID < 0 {
		event.ID = account.NextEventID
		account.NextEventID++
		modified = true
	}

	if event.Sibling != nil {
		if event.SiblingID < 0 {
			event.SiblingID = account.NextEventID
			event.Sibling.ID = account.NextEventID
			account.NextEventID++
			modified = true
		}
	}

	if event.Child != nil {
		if event.ChildID < 0 {
			event.ChildID = account.NextEventID
			event.Child.ID = account.NextEventID
			account.NextEventID++
			modified = true
		}
	}

	var err error

	key := datastore.NewIncompleteKey(c, "Event", parentKey)

	if key, err = datastore.Put(c, key, event); err != nil {
		log.Debugf(c, "Failed to put event in ds: %v", err)
		return modified, err
	}
	account.putEventInMemcache(event, c)

	if event.Sibling != nil {
		m, err := account.putInDS(key, event.Sibling, c)
		if err != nil {
			log.Debugf(c, "Failed to put sibling event in ds: %v", err)
			return (modified || m), err
		}
	}

	if event.Child != nil {
		m, err := account.putInDS(key, event.Child, c)
		if err != nil {
			log.Debugf(c, "Failed to put child event in ds: %v", err)
			return (modified || m), err
		}
	}

	return modified, nil
}

func (account *Account) updateInDS(accountKey *datastore.Key, event *Event, c context.Context) (bool, error) {
	dsEvent, key := account.retrieveEventFromDS(accountKey, event.ID, c)
	if dsEvent == nil {
		log.Debugf(c, "Failed to find event in datastore! No update occurred.")
		return false, nil
	}

	if _, err := datastore.Put(c, key, event); err != nil {
		log.Debugf(c, "Failed to put event in ds: %v", err)
		return false, err
	}

	account.putEventInMemcache(event, c)

	return false, nil
}

func (account *Account) updateInDSDeep(accountKey *datastore.Key, event *Event, c context.Context) (bool, error) {
	dsEvent := account.retrieveEventFromDSDeep(accountKey, event.ID, c)

	accountUpdated := account.updateTree(dsEvent, event, c)

	return accountUpdated, nil
}

func (account *Account) retrieveEvent(accountKey *datastore.Key, eventID int, c context.Context) *Event {
	event := &Event{}
	// if _, err := memcache.JSON.Get(c, -----, event); err == nil {
	// 	c.Debugf("Found event in memcache")
	// 	return &event
	// }
	event, _ = account.retrieveEventFromDS(accountKey, eventID, c)
	return event
}

func (account *Account) retrieveEventFromDS(accountKey *datastore.Key, eventID int, c context.Context) (*Event, *datastore.Key) {
	if accountKey == nil {
		accountKey = datastore.NewKey(c, "Account", account.Email, 0, nil)
	}

	query := datastore.NewQuery("Event").Ancestor(accountKey).Filter("ID =", eventID)

	var events []Event
	keys, err := query.GetAll(c, &events)
	if err != nil || len(events) == 0 {
		log.Debugf(c, "Failed get event from DS: %v", err)
		return nil, nil
	}
	if len(events) != 1 {
		log.Debugf(c, "Found multiple events with the same ID in DS: %v", len(events))
	}

	return &events[0], keys[0]
}

func (account *Account) retrieveEventDeep(accountKey *datastore.Key, eventID int, c context.Context) *Event {
	return account.retrieveEventFromDSDeep(accountKey, eventID, c)
}

func (account *Account) retrieveEventFromDSDeep(accountKey *datastore.Key, eventID int, c context.Context) *Event {
	event, eventKey := account.retrieveEventFromDS(accountKey, eventID, c)
	if eventKey == nil {
		log.Debugf(c, "Failed get event from DS")
		return nil
	}

	query := datastore.NewQuery("Event").Ancestor(eventKey)

	var events []Event
	_, err := query.GetAll(c, &events)
	if err != nil || len(events) == 0 {
		log.Debugf(c, "Failed get event from DS: %v", err)
		return nil
	}

	lookup := map[int]*Event{}
	for i, e := range events {
		lookup[e.ID] = &events[i]
	}

	event.fillInTree(lookup, c)
	return event
}

func (event *Event) fillInTree(lookup map[int]*Event, c context.Context) {
	if event.SiblingID >= 0 {
		if e, ok := lookup[event.SiblingID]; ok {
			event.Sibling = e
			event.Sibling.fillInTree(lookup, c)
		} else {
			log.Debugf(c, "Failed to find sibling event in lookup: %d", event.SiblingID)
		}
	}
	if event.ChildID >= 0 {
		if e, ok := lookup[event.ChildID]; ok {
			event.Child = e
			event.Child.fillInTree(lookup, c)
		} else {
			log.Debugf(c, "Failed to find child event in lookup: %d", event.ChildID)
		}
	}
}

func (account *Account) updateTree(newEvent *Event, oldEvent *Event, c context.Context) bool {
	// needToUpdateDS := false
	// accountWasModified := false
	//
	// if newEvent.SiblingID != oldEvent.SiblingID {
	// 	needToUpdateDS = true
	// 	if newEvent.SiblingID == -1 { //sibling was deleted
	// 		_, key := account.retrieveEventFromDS(nil, oldEvent.SiblingID, c)
	// 		datastore.Delete(c, key)
	// 	} else if newEvent.Sibling.ID == -1 { //the sibling is new
	// 		_, key := account.retrieveEventFromDS(nil, oldEvent.ID, c)
	// 		accountWasModified = account.putInDS(key, newEvent.Sibling, c)
	// 	} else { //the sibling exists already
	// 		accountWasModified = account.updateTree(newEvent.)
	//
	// 	}
	// }
	return false
}
