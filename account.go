package dependevent

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

type Account struct {
	Email          string
	NextEventID    int
	RootEventIDs   []int
	ActiveEventIDs []int

	RootEvents []*Event `datastore:"-"`
	ActiveEvents []*Event `datastore:"-"`
}

func (account *Account) putInMemcache(c appengine.Context) {
	fullLookup := &memcache.Item{
		Key:    account.Email,
		Object: account,
	}
	memcache.JSON.Set(c, fullLookup)
}

func (account *Account) saveInDS(c appengine.Context) error {
	key := datastore.NewKey(c, "Account", account.Email, 0, nil)
	if _, err := datastore.Put(c, key, account); err != nil {
		c.Debugf("failed to save new accont: %s", err)
		return err
	}
	account.putInMemcache(c)
	return nil
}

func getAccountByEmail(email string, c appengine.Context) *Account {
	var account *Account
	if _, err := memcache.JSON.Get(c, email, account); err == nil {
		c.Debugf("got account from memcache")
		makeSafe(account)
		return account
	}

	key := datastore.NewKey(c, "Account", email, 0, nil)
	account = &Account{}
	err := datastore.Get(c, key, account)
	if err == nil {
		makeSafe(account)
		// for _, eventKey := range account.RootEventKeyStrings {
		// 	event := retrieveEventFromDS(eventKey, c)
		// 	if event != nil {
		// 		account.RootEvents[eventKey] = event
		// 		// account.RootEvents = append(account.RootEvents, *event)
		// 	} else {
		// 		c.Debugf("failed to find root event")
		// 	}
		// }
		// for _, eventKey := range account.ActiveEventKeyStrings {
		// 	event := retrieveEventFromDS(eventKey, c)
		// 	if event != nil {
		// 		account.ActiveEvents[eventKey] = event
		// 		// account.ActiveEvents = append(account.ActiveEvents, *event)
		// 	} else {
		// 		c.Debugf("failed to find active event")
		// 	}
		// }

		account.putInMemcache(c)
	} else {
		account = &Account{
			Email: email,
			NextEventID: 0,
		}
		makeSafe(account)
		account.saveInDS(c)
	}

	return account
}

func (account *Account) populateEvents(c appengine.Context) {
	accountKey := datastore.NewKey(c, "Account", account.Email, 0, nil)

	lookup := map[int]*Event{}

	if len(account.RootEventIDs) != len(account.RootEvents) {
		if account.RootEvents == nil {
			account.RootEvents = []*Event{}
		}
		for _, id := range account.RootEventIDs {
			event := account.retrieveEvent(accountKey, id, c)
			if event != nil {
					account.RootEvents = append(account.RootEvents, event)
					lookup[id] = event
			} else {
				c.Debugf("failed to find root event: %d", id)
			}
		}
	}
	if len(account.ActiveEventIDs) != len(account.ActiveEvents) {
		if account.ActiveEvents == nil {
			account.ActiveEvents = []*Event{}
		}
		for _, id := range account.ActiveEventIDs {
			event := lookup[id]
			if event == nil {
				event = account.retrieveEvent(accountKey, id, c)
			}
			account.ActiveEvents = append(account.ActiveEvents, event)
		}
	}
}

func makeSafe(account *Account) {
	if account.RootEventIDs == nil {
		account.RootEventIDs = []int{}
	}
	if account.ActiveEventIDs == nil {
		account.ActiveEventIDs = []int{}
	}
	if account.RootEvents == nil {
		account.RootEvents = []*Event{}
	}
	if account.ActiveEvents == nil {
		account.ActiveEvents = []*Event{}
	}
}
