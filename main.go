package main

import (
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"golang.org/x/net/html"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"

	"github.com/gorilla/mux"
)

var pages *template.Template

func init() {
	pages = template.Must(template.ParseGlob("pages/*.html"))

	r := mux.NewRouter()
	r.HandleFunc("/", showIndex).Methods("GET")
	r.HandleFunc("/add", addLink).Methods("POST")
	http.HandleFunc("/edit", editLinkTitle)
	//TODO: should be delete, but I don't feel like writing JS to access a fundamental HTTP verb right now
	r.HandleFunc("/link/{key}", delLink).Methods("POST")
	http.Handle("/", r)
}

func showIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)

	u := user.Current(c)
	if u == nil {
		showLogin(w, c)
		return
	}

	us, uk, err := getUser(c)
	if err != nil {
		showError(w, "Ask Steve to look.", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user %q: %v", u.String(), err)
	}

	links := []Link{}
	lks, err := datastore.NewQuery("Link").
		Ancestor(uk).
		Order("-Added").
		GetAll(c, &links)
	if err != nil {
		showError(w, "Ask Scott to look.", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user's links %q: %v", u.String(), err)
		return
	}

	x := struct {
		User     User
		Links    []Link
		LinkKeys []*datastore.Key
		EncodedLinkKeys []string
	}{
		User:     us,
		Links:    links,
		LinkKeys: lks,
		EncodedLinkKeys: make([]string, len(lks)),
	}

	for index, linkKey := range x.LinkKeys {
		x.EncodedLinkKeys[index] = linkKey.Encode()
	}

	err = pages.ExecuteTemplate(w, "index.html", x)
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render index: %v", err)
		return
	}
}

func showLogin(w http.ResponseWriter, c appengine.Context) {
	login, err := user.LoginURL(c, "/")
	if err != nil {
		showError(w, "Ask Steve.", http.StatusInternalServerError, c)
		c.Errorf("Failed to get login url: %v", err)
		return
	}

	err = pages.ExecuteTemplate(w, "login.html", login)
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render login: %v", err)
		return
	}
}

func showError(w http.ResponseWriter, msg string, status int, c appengine.Context) {
	err := pages.ExecuteTemplate(w, "oops.html", map[string]string{
		"Message": msg,
		"Status":  http.StatusText(status),
	})
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render error: %v", err)
		return
	}
}

func addLink(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	_, uk, err := getUser(c)
	if err != nil {
		showError(w, "failed to retrieve user", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user %q: %v", u.String(), err)
		return
	}

	url := strings.TrimSpace(r.FormValue("url"))
	if url == "" {
		showError(w, "Empty URLs are not allowed.", http.StatusBadRequest, c)
		return
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		showError(w, "Please specify http:// or https:// in your URL.", http.StatusBadRequest, c)
		return
	}

	l := Link{
		URL:   url,
		Added: time.Now(),
	}

	resp, err := urlfetch.Client(c).Get(url)
	if err != nil {
		l.Title = err.Error()
	} else {
		defer resp.Body.Close()
		l.Title = parseTitle(resp.Body)
	}

	_, err = l.Save(c, uk)
	if err != nil {
		showError(w, "failed to store link", http.StatusInternalServerError, c)
		c.Errorf("failed to store link: %v", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func parseTitle(resp io.Reader) string {
	r := io.LimitedReader{
		R: resp,
		N: 8192,
	}

	h := html.NewTokenizer(&r)
	for {
		tt := h.Next()
		switch tt {
		case html.ErrorToken:
			return "Failed to parse page"
		case html.StartTagToken:
			tag, _ := h.TagName()
			if string(tag) == "title" {
				nt := h.Next()
				switch nt {
				case html.ErrorToken:
					return "Failed to parse title"
				case html.TextToken:
					return h.Token().Data
				}
			}
		}
	}

	return "Failed to find title"
}

func editLinkTitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)
	u := user.Current(c)

	_, _, err := getUser(c)
	if err != nil {
		showError(w, "failed to retrieve user", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user %q: %v", u.String(), err)
		return
	}

	var k *datastore.Key

	if k, err = datastore.DecodeKey(r.FormValue("Key")); err != nil {
        http.Error(w, err.Error(), 501)
        return
	}

	l := new(Link)
	if err := datastore.Get(c, k, l); err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    l.Title = r.FormValue("Title")

	if _, err := datastore.Put(c, k, l); err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
}

func delLink(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	key := mux.Vars(r)["key"]
	if key == "" {
		http.NotFound(w, r)
		return
	}

	k, err := datastore.DecodeKey(key)
	if err != nil {
		showError(w, "Invalid link key.", http.StatusBadRequest, c)
		return
	}

	err = datastore.Delete(c, k)
	if err != nil {
		showError(w, "Failed to delete link.", http.StatusInternalServerError, c)
		c.Errorf("Failed to delete %v: %v", k, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}