/*
* The Bus Information Agent
* Created by Earl Balai Jr
 */
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var hea_key string
var enableDatabase bool

const (
	bus_api_url = "http://api.thebus.org/arrivals/"
)

type Arrival struct {
	ID        int    `xml:"id"`
	Route     int    `xml:"route"`
	StopTime  string `xml:"stopTime"`
	Direction string `xml:"direction"`
	HeadSign  string `xml:"headsign"`
}

type StopTimes struct {
	StopID int       `xml:"stop"`
	Data   []Arrival `xml:"arrival"`
}

// Twilio Call Structs
type TwiML struct {
	XMLName xml.Name `xml:"Response"`

	Say    []SayBlock  `xml:",omitempty"`
	Gather GatherBlock `xml:",omitempty"`
	Play   string      `xml:",omitempty"`
	Hangup string      `xml:",omitempty`
}

type SayBlock struct {
	Voice string `xml:"voice,attr"`
	Lang  string `xml:"language,attr"`
	Msg   string `xml:",chardata"`
}

type GatherBlock struct {
	Action string   `xml:"action,attr"`
	Method string   `xml:"method,attr"`
	Say    SayBlock `xml:",omitempty"`
}

type SmsBlock struct {
	XMLName xml.Name `xml:"Response"`
	Message string   `xml:",omitempty"`
}

func phone_arrivals(w http.ResponseWriter, r *http.Request) {
	caller_id := r.FormValue("From")
	call_status := r.FormValue("CallStatus")
	stop_number := r.FormValue("Digits")
	stopNum, err := strconv.Atoi(stop_number)

	fmt.Printf("Caller: %s [%s] Stop Number: %s\n", caller_id, call_status, stop_number)

	twiml := TwiML{
		Say: []SayBlock{
			SayBlock{
				Voice: "alice",
				Lang:  "en-US",
				Msg:   "Please wait while we gather your transit information...",
			},
			SayBlock{
				Voice: "alice",
				Lang:  "en-US",
				Msg:   fmt.Sprintf("The next arrival for bus stop number %s for %s... Thank you for calling!", stop_number, getArrivals(stopNum, "phone")),
			},
		},
		Hangup: "endcall",
	}

	x, err := xml.Marshal(twiml)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(x)

	request_type := "CALL"
	from_city := r.FormValue("FromCity")
	from_state := r.FormValue("FromState")
	from_zip := r.FormValue("FromZip")
	from_country := r.FormValue("FromCountry")
	call_sid := r.FormValue("CallSid")

	log_data := []string{request_type, caller_id, stop_number, from_city, from_state, from_zip, from_country, call_sid, call_status}

	log2DB(log_data)
}

func twiml(w http.ResponseWriter, r *http.Request) {
	caller_id := r.FormValue("From")
	call_status := r.FormValue("CallStatus")

	fmt.Printf("[Incoming] Caller: %s - %s\n", caller_id, call_status)

	twiml := TwiML{
		Say: []SayBlock{
			SayBlock{
				Voice: "alice",
				Lang:  "en-US",
				Msg:   "Thank you for calling the The Bus information agent!",
			},
		},
		Gather: GatherBlock{
			Action: "/getarrivals",
			Method: "POST",
			Say: SayBlock{
				Voice: "alice",
				Lang:  "en-US",
				Msg:   "Please enter your bus stop number, followed by the pound sign.",
			}},
	}

	x, err := xml.Marshal(twiml)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(x)
}

func update_callstatus(w http.ResponseWriter, r *http.Request) {
	call_sid := r.FormValue("CallSid")
	call_status := r.FormValue("CallStatus")
	call_duration := r.FormValue("CallDuration")

	fmt.Printf("CALL SID: %s STATUS: %s DURATION: %s", call_sid, call_status, call_duration)

	db := OpenDB()

	db.Exec("UPDATE log SET call_duration=$1, call_status=$2 WHERE call_sid=$3", call_duration, call_status, call_sid)

	defer db.Close()
}

func sms(w http.ResponseWriter, r *http.Request) {
	sender := r.FormValue("From")
	message, error := strconv.Atoi(r.FormValue("Body"))
	reply := SmsBlock{Message: "There was a problem validating your request... Please try again later."}
	if error != nil {
		reply = SmsBlock{Message: "There was a problem validating your request... Please try again later."}
	} else {
		reply = SmsBlock{
			Message: fmt.Sprintf("\nThe Bus Information System\nStop ID: %s\n----------------\n%s", fmt.Sprintf("%d", message), getArrivals(message, "sms")),
		}
	}

	x, err := xml.Marshal(reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(x)

	fmt.Printf("Sender: %s Message: %d\n", sender, message)

	// Log specific
	requestType := "SMS"
	fromCity := r.FormValue("FromCity")
	fromState := r.FormValue("FromState")
	fromZip := r.FormValue("FromZip")
	fromCountry := r.FormValue("FromCountry")
	callSid := "SMS-METHOD"

	log_data := []string{requestType, sender, fmt.Sprintf("%d", message), fromCity, fromState, fromZip, fromCountry, callSid, "UNKNOWN"}

	log2DB(log_data)

}

func (s Arrival) String() string {
	return fmt.Sprintf("\t Bus ID : %d - Route: %d - Arrival: %s - Direction: %s - Destination: %s \n", s.ID, s.Route, s.StopTime, s.Direction, s.HeadSign)
}

func getArrivals(stopID int, method string) string {
	url := bus_api_url
	url += "?key=" + hea_key
	url += "&stop=" + fmt.Sprintf("%d", stopID)
	res, err := http.Get(url)
	if err != nil {
		panic(err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	ndat := strings.Replace(string(body), "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>", "<?xml version=\"1.0\" encoding=\"UTF-8\"?>", 1) // Temporary work around for xml encoding issue on bus API

	dat := StopTimes{}
	_ = xml.Unmarshal([]byte(ndat), &dat)

	arrival_info := "No arrival information available."

	if dat.Data != nil {

		if method == "phone" {
			arrival_info = fmt.Sprintf("Route number: %d, is %s, heading to: %s, with an estimated arrival at %s.", dat.Data[0].Route, dat.Data[0].Direction, dat.Data[0].HeadSign, dat.Data[0].StopTime)
		} else {
			arrival_info = fmt.Sprintf("Route: %d \nArrival: %s \nDirection: %s \nDestination: %s", dat.Data[0].Route, dat.Data[0].StopTime, dat.Data[0].Direction, dat.Data[0].HeadSign)
		}
	}

	return arrival_info
}

func log2DB(data []string) {
	request_type := data[0]
	phone_number := data[1]
	stop_id := data[2]
	caller_city := data[3]
	caller_state := data[4]
	caller_zip := data[5]
	caller_country := data[6]
	call_sid := data[7]
	call_status := data[8]

	dinfo := fmt.Sprintf("Request Type: %s\nPhone Number: %s\nStop ID: %s\nCaller City: %s\nCaller State: %s\nCaller Zip: %s\nCaller Country: %s\nCall SID: %s\nCall Status: %s", request_type, phone_number, stop_id, caller_city, caller_state, caller_zip, caller_country, call_sid, call_status)
	fmt.Println(dinfo)

	if phone_number != "" && enableDatabase {

		db := OpenDB()

		_, err := db.Exec("INSERT INTO log(request_type, phone_number, stop_id, from_city, from_state, from_zip, from_country, call_sid, call_duration, call_status) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)", request_type, phone_number, stop_id, caller_city, caller_state, caller_zip, caller_country, call_sid, 0, call_status)

		if err != nil {
			fmt.Printf("Error: %v", err)
		}

		defer db.Close()
	}
}

func main() {

	flag.StringVar(&hea_key, "key", "", "HEA The Bus API Key")
	flag.BoolVar(&enableDatabase, "useDB", false, "Enable usage of PostgreSQL database true/false\nExample: -useDB true")

	flag.Parse()

	if len(hea_key) <= 0 {
		log.Fatal("Please specify the -key argument with your api key.\nExample: -key \"2D1AB133-822C-414B-88FF-62CC2C94AE49\"")
	}

	http.HandleFunc("/twiml", twiml)
	http.HandleFunc("/getarrivals", phone_arrivals)
	http.HandleFunc("/callstatus", update_callstatus)
	http.HandleFunc("/sms", sms)
	http.ListenAndServe(":3000", nil)
}
