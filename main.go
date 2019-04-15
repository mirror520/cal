package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Calendar ..
type Calendar struct {
	ID         string
	Summary    string
	Department string
}

// CalendarContext ...
type CalendarContext struct {
	Service   *calendar.Service
	Calendars []Calendar
}

func (calCtx *CalendarContext) loadCalendarList() {
	calCtx.Calendars = []Calendar{
		Calendar{Department: "文檔", Summary: "0. 文檔科行事曆", ID: "6a9uk9gc2d15udchreqeq3k1rk@group.calendar.google.com"},
		Calendar{Department: "總務", Summary: "1. 總務科行事曆", ID: "ktmlkpioqff9nu111p7as526v4@group.calendar.google.com"},
		Calendar{Department: "公關", Summary: "2. 公共關係科行事曆", ID: "o4bkrddgv0b4bq6b5j6kpedae8@group.calendar.google.com"},
		Calendar{Department: "國際", Summary: "3. 國際事務科行事曆", ID: "iidlj6hgoelq6s9rsumob02rrc@group.calendar.google.com"},
		Calendar{Department: "機要", Summary: "4. 機要科行事曆", ID: "h55qgjl1llcbuu0aqqf71hur2o@group.calendar.google.com"},
		Calendar{Department: "廳舍", Summary: "5. 廳舍管理科行事曆", ID: "ts9chk929t0jponh9cegafp4m8@group.calendar.google.com"},
		Calendar{Department: "採購", Summary: "6. 採購管理科行事曆", ID: "qcugjootlce6q62qubtsi4okqg@group.calendar.google.com"},
		Calendar{Department: "人事", Summary: "7. 人事室行事曆", ID: "r7uqipsmab6dseejdcur39f2nc@group.calendar.google.com"},
		Calendar{Department: "會計", Summary: "8. 會計室行事曆", ID: "uhaeie6j2sinimlr8r6196v0rc@group.calendar.google.com"},
		Calendar{Department: "政風", Summary: "9. 政風室行事曆", ID: "in5cciirl6oakjiccac1l1k0l0@group.calendar.google.com"},
	}
}

func (calCtx *CalendarContext) getLastUpdatedCalendarEvents(cal Calendar, in chan<- []*calendar.Event) {
	for {
		t := time.Now().Add(-24 * time.Second).Format(time.RFC3339)
		events, err := calCtx.Service.Events.List(cal.ID).UpdatedMin(t).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
		}
		fmt.Printf("--- %s ---\n", cal.Department)
		fmt.Println("Upcoming events:")
		if len(events.Items) == 0 {
			fmt.Println("No upcoming events found.")
		} else {
			for _, item := range events.Items {
				fmt.Printf("%v (%v, %v)\n", item.Summary, item.Id, item.Updated)
				item.Summary = fmt.Sprintf("[%s] %s", cal.Department, item.Summary)
			}

			in <- events.Items
		}
		fmt.Println()

		time.Sleep(10 * time.Second)
	}
}

func (calCtx *CalendarContext) syncLastUpdatedCalendarEvents(calendarID string, out <-chan []*calendar.Event) {
	for {
		for _, event := range <-out {
			calCtx.importEvent(calendarID, event)
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

	events := make(chan []*calendar.Event)

	go calCtx.syncLastUpdatedCalendarEvents("l3gc5npg7m05a730gqgeobrcfo@group.calendar.google.com", events)

	for _, cal := range calCtx.Calendars {
		go calCtx.getLastUpdatedCalendarEvents(cal, events)
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
