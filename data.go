package main

import (
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

type User struct {
	Email string
	Tags  []string
}

func getUser(c appengine.Context) (User, *datastore.Key, error) {
	u := user.Current(c)

	//TODO: we're using google accounts for now,
	// but this may need to change to something more complex if we change.
	uk := datastore.NewKey(c, "User", u.ID, 0, nil)
	var us User
	err := datastore.Get(c, uk, &us)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return us, uk, err
	}
	return us, uk, nil
}

type Link struct {
	URL   string
	Title string
	Tags  []string
	Added time.Time
}

func (l *Link) Save(c appengine.Context, parent *datastore.Key) (*datastore.Key, error) {
	//TODO: trim spaces
	//TODO: fetch the title
	lk := datastore.NewIncompleteKey(c, "Link", parent)
	return datastore.Put(c, lk, l)
}

func linkAlreadyExists(c appengine.Context, uk *datastore.Key, url string) (bool, error) {
	links := []Link{}
	_, err := datastore.NewQuery("Link").
		Ancestor(uk).
		Filter("URL = ", url).
		GetAll(c, &links)
	
	return len(links) > 0, err
}