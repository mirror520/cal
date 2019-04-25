package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Calendar ...
type Calendar struct {
	ID           string
	Summary      string
	Prefix       string
	ColorID      int
	SubCalendars []Calendar
	Members      []Member
}

// Member ...
type Member struct {
	Name  string
	Title string
	Cal   Calendar
}

// CalendarContext ...
type CalendarContext struct {
	Service   *calendar.Service
	Calendars []Calendar
}

func (calCtx *CalendarContext) loadCalendarList() {
	calCtx.Calendars = calData
}

func (calCtx *CalendarContext) getLastUpdatedCalendarEvents(cal Calendar, in chan<- []*calendar.Event) {
	for {
		t := time.Now().Add(-24 * time.Second).Format(time.RFC3339)
		events, err := calCtx.Service.Events.List(cal.ID).UpdatedMin(t).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
		}
		fmt.Printf("--- %s ---\n", cal.Summary)
		fmt.Println("Upcoming events:")
		if len(events.Items) == 0 {
			fmt.Println("No upcoming events found.")
		} else {
			for _, item := range events.Items {
				fmt.Printf("%v (%v, %v)\n", item.Summary, item.Id, item.Updated)
				item.Summary = fmt.Sprintf("[%s]%s", cal.Prefix, item.Summary)
				item.ColorId = strconv.Itoa(cal.ColorID)
			}

			in <- events.Items
		}
		fmt.Println()

		time.Sleep(10 * time.Second)
	}
}

func (calCtx *CalendarContext) syncLastUpdatedCalendarEvents(cal Calendar, out <-chan []*calendar.Event) {
	for {
		for _, event := range <-out {
			calCtx.importEvent(cal.ID, event)
		}
	}
}

func (calCtx *CalendarContext) importEvent(calendarID string, event *calendar.Event) {
	_, err := calCtx.Service.Events.Get(calendarID, event.Id).Do()
	if err != nil {
		fmt.Printf("Unable to retrieve event id: %s, %v\n", event.Id, err)
		fmt.Printf("Insert Event: %v (%v)\n", event.Summary, event.Id)
		calCtx.Service.Events.Insert(calendarID, event).Do()
	} else {
		if event.Status == "cancelled" {
			fmt.Printf("Delete Event: %v (%v)\n", event.Summary, event.Id)
			calCtx.Service.Events.Delete(calendarID, event.Id).Do()
		} else {
			fmt.Printf("Update Event: %v (%v)\n", event.Summary, event.Id)
			calCtx.Service.Events.Update(calendarID, event.Id, event).Do()
		}
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	calCtx := CalendarContext{Service: srv}
	calCtx.loadCalendarList()

	for _, cal := range calCtx.Calendars {
		events := make(chan []*calendar.Event)
		go calCtx.syncLastUpdatedCalendarEvents(cal, events)

		for _, member := range cal.Members {
			go calCtx.getLastUpdatedCalendarEvents(member.Cal, events)
		}

		for _, subCal := range cal.SubCalendars {
			go calCtx.getLastUpdatedCalendarEvents(subCal, events)

			subEvents := make(chan []*calendar.Event)
			go calCtx.syncLastUpdatedCalendarEvents(subCal, subEvents)

			for _, member := range subCal.Members {
				go calCtx.getLastUpdatedCalendarEvents(member.Cal, subEvents)
			}
		}
	}

	abort := make(chan bool)
	go func() {
		for {
			os.Stdin.Read(make([]byte, 1))
			abort <- true
		}
	}()

	func() {
		for {
			select {
			case <-abort:
				fmt.Println("Here")
				return
			}
		}
	}()
}
