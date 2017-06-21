package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	AccountSID  string
	AuthToken   string
	KylesNumber string
	OurNumber   string
	ApiURL      string
	Port        string
	Template    *template.Template
)

// Partial struct of possible Twilio Responses
type TwilioResponse struct {
	RestException struct {
		Code     int
		Message  string
		Status   int
		MoreInfo string
	}
	Message struct {
		Sid                 string
		DateCreated         string
		DateSent            string
		DateUpdated         string
		AccountSid          string
		To                  string
		From                string
		MessagingServiceSid string
		Body                string
		Status              string
		ErrorCode           int
		ErrorMessage        string
	}
}

type Page struct {
	Message string
	Sent    string
	Error   string
}

func init() {
	var err error

	for _, pair := range os.Environ() {
		split := strings.SplitN(pair, "=", 2)
		switch split[0] {
		case "ACCOUNT_SID":
			AccountSID = split[1]
		case "AUTH_TOKEN":
			AuthToken = split[1]
		case "KYLES_NUMBER":
			KylesNumber = split[1]
		case "OUR_NUMBER":
			OurNumber = split[1]
		case "PORT":
			Port = split[1]
		}
	}

	if AccountSID == "" {
		log.Fatal("Environment variable ACCOUNT_SID is required")
	} else if AuthToken == "" {
		log.Fatal("Environment variable AUTH_TOKEN is required")
	} else if KylesNumber == "" {
		log.Fatal("Environment variable KYLES_NUMBER is required")
	} else if OurNumber == "" {
		log.Fatal("Environment variable OUR_NUMBER is required")
	} else if Port == "" {
		Port = "8080"
	}

	ApiURL = fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages", AccountSID)

	Template, err = template.ParseFiles("template.html")
	if err != nil {
		panic(err)
	}
}
func main() {
	http.HandleFunc("/", handler)

	log.Printf("Serving on :%s\n", Port)
	http.ListenAndServe(":"+Port, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	page := Page{}

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err)
			return
		}

		if len(r.PostForm["message"]) != 1 {
			page.Error = "A message is required."
		} else {
			page.Message = r.PostForm["message"][0]
		}

		if len(page.Message) == 0 {
			page.Error = "A message is required."
		}

		if len(page.Message) > 140 {
			page.Error = "Message too long!"
		}

		if page.Error == "" {
			err := textKyle(page.Message)
			if err != nil {
				page.Error = err.Error()
			} else {
				log.Printf("%s sent \"%s\"", r.RemoteAddr, page.Message)
				page.Sent = page.Message
				page.Message = ""
			}
		}
	}

	Template.Execute(w, page)
}

func textKyle(message string) error {
	v := url.Values{}
	v.Set("To", KylesNumber)
	v.Set("From", OurNumber)
	v.Set("Body", message)

	req, err := http.NewRequest("POST", ApiURL, strings.NewReader(v.Encode()))
	if err != nil {
		return fmt.Errorf("Could not create HTTP request.")
	}
	req.SetBasicAuth(AccountSID, AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("Twilio API error")
	}

	if resp.StatusCode != 201 {
		defer resp.Body.Close()

		decoder := xml.NewDecoder(resp.Body)
		body := TwilioResponse{}
		err = decoder.Decode(&body)
		if err != nil {
			log.Println(err)
		}

		log.Println(body.RestException.Message)

		return fmt.Errorf("Twilio API error: %s", body.RestException.Message)
	}

	return nil
}
