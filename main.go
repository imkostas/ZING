package main

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/anachronistic/apns"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"gopkg.in/gorp.v1"
)

// Location struct for holding data from the Location table from the database
type Location struct {
	ID        int     `json:"-" db:"id"` //`json:"id" db:"id"`
	Username  string  `json:"username" db:"username"`
	UDID      string  `json:"udid" db:"udid"`
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
}

// Locations type is a helper type to make the handling of multiple locations easier
type Locations []Location

// Pair struct holds data about pairs between two udids
type Pair struct {
	ID        int    `db:"id"`
	UDID1     string `db:"udid_1"`
	UDID2     string `db:"udid_2"`
	SessionID string `db:"session_id"`
}

// Pairs type is a helper type to hold a slice of pairs
type Pairs []Pair

var db *sql.DB
var dbmap *gorp.DbMap

var local = false

const localString string = "root:root@tcp(localhost:8889)/zing"
const serverString string = "root:288norfolk@/zing"

// connectionString := "root:root@tcp(localhost:8889)/ZING"
// connectionString := "root:288norfolk@/ZING"

func initDB() {
	var connectionString = ""
	if local {
		connectionString = localString
	} else {
		connectionString = serverString
	}

	gob.Register(&Location{})
	gob.Register(&Pair{})
	db, _ = sql.Open("mysql", connectionString)
	dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	dbmap.AddTableWithName(Location{}, "locations").SetKeys(true, "ID")
	dbmap.AddTableWithName(Pair{}, "pairs").SetKeys(true, "ID")
	dbmap.CreateTablesIfNotExists()
}

// func Index(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintln(w, "Welcome!")
// }
//
// func GetIndex(w http.ResponseWriter, r *http.Request) {
// 	var locations Locations
// 	_, err := dbmap.Select(&locations, "SELECT * FROM Locations")
// 	if err != nil {
// 		log.Println("Error fetching results from database", err)
// 	}
//
// 	json.NewEncoder(w).Encode(locations)
// }

// GetIndex function searches the database for data available
func GetIndex(w http.ResponseWriter, r *http.Request) {

	// TODO:
	// Get all indexed data
	var locs Locations
	_, err := dbmap.Select(&locs, "SELECT * FROM locations WHERE 1 ORDER BY username")

	if err != nil {
		log.Println("Select error: ", err)
	}

	// Return the locations to udid
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(locs)

}

// GetLocation function searches the database for the location of the given udid
func GetLocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid := vars["udid"]
	var location Location
	err := dbmap.SelectOne(&location, "SELECT * FROM locations WHERE udid=?", udid)
	if err != nil {
		log.Printf("Entry for %s not found", udid)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(location)
}

// SetLocation function sets the location for the given udid in the database
func SetLocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	udid := vars["udid"]
	latitude, _ := strconv.ParseFloat(vars["latitude"], 64)
	longitude, _ := strconv.ParseFloat(vars["longitude"], 64)

	//var location Location
	loc := &Location{0, "", "", 0, 0}
	err := dbmap.SelectOne(loc, "SELECT * FROM locations WHERE udid=?", udid)
	if err != nil {
		log.Printf("Entry for %s not found", udid)
	}

	loc.Username = username
	loc.UDID = udid
	loc.Latitude = latitude
	loc.Longitude = longitude

	if loc.ID == 0 {
		err = dbmap.Insert(loc)
	} else {
		_, err = dbmap.Update(loc)
	}

	if err != nil {
		log.Println("Error updating database", err)
	}

	fmt.Fprintf(w, "Location set for %s: username=%s lat=%f lng=%f", udid, username, latitude, longitude)
}

// CreatePair function creates a pairing of the two given udids
func CreatePair(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid1 := vars["udid1"]
	udid2 := vars["udid2"]

	// Check if pair already exists (if it does, exit || re-auth session)
	var pair Pair
	err := dbmap.SelectOne(&pair, "SELECT * FROM pairs WHERE udid_1=? AND udid_2=?", udid1, udid2)
	if err != nil {
		log.Println("Select error: ", err)
	}

	if pair.ID != 0 {
		return
	}

	//  - Send notification to udid2 for approval of pairing

	// Write pair into database
	pair.UDID1 = udid1
	pair.UDID2 = udid2
	pair.SessionID = ""

	err = dbmap.Insert(&pair)
	if err != nil {
		log.Println("Insert error: ", err)
	}

	// Write reverse pair into database
	pair.UDID2 = udid1
	pair.UDID1 = udid2
	pair.SessionID = ""

	err = dbmap.Insert(&pair)
	if err != nil {
		log.Println("Insert error: ", err)
	}
}

// RemovePair function removes the pairing of the two given udids
func RemovePair(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid1 := vars["udid1"]
	udid2 := vars["udid2"]

	// Find pairing between the two udids
	var pair Pair
	err := dbmap.SelectOne(&pair, "SELECT * FROM pairs WHERE udid_1=? AND udid_2=?", udid1, udid2)
	if err != nil {
		log.Println("Select error: ", err)
	}

	var pair2 Pair
	err = dbmap.SelectOne(&pair2, "SELECT * FROM pairs WHERE udid_1=? AND udid_2=?", udid2, udid1)
	if err != nil {
		log.Println("Select error: ", err)
	}

	if pair.ID == 0 {
		return
	}

	// Remove the pairings in the database
	_, err = dbmap.Delete(&pair)
	if err != nil {
		log.Println("Delete error: ", err)
	}

	if pair2.ID == 0 {
		return
	}

	// Remove the inverse pairings in the database
	_, err = dbmap.Delete(&pair2)
	if err != nil {
		log.Println("Delete error: ", err)
	}
}

// GetAllLocations function gets the locations of udids paired with the given udid
func GetAllLocations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid := vars["udid"]

	// TODO:
	// Find all pairings with udid
	var ids []string
	_, err := dbmap.Select(&ids, "SELECT udid_2 FROM pairs WHERE udid_1=?", udid)

	if err != nil {
		log.Println("Select error: ", err)
	}

	if ids == nil {
		return
	}

	var locs Locations

	// Get the locations of those paired with udid
	for _, id := range ids {
		var loc Location
		err = dbmap.SelectOne(&loc, "SELECT * FROM locations WHERE udid=?", id)
		if err != nil {
			log.Println("Select error: ", err)
		}

		locs = append(locs, loc)
	}

	// Return the locations to udid
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(locs)
}

func sendNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid := vars["udid"]
	message := vars["message"]

	payload := apns.NewPayload()
	payload.Alert = message
	payload.Badge = 0
	payload.Sound = "bingbong.aiff"

	pn := apns.NewPushNotification()
	pn.DeviceToken = udid
	pn.AddPayload(payload)
	
	pn.Set("udid", udid)

	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", "./certs/zing_cert.pem", "./certs/zing_key.pem")
	resp := client.Send(pn)

	alert, _ := pn.PayloadString()
	fmt.Fprintf(w,"  Alert: %s", alert)
	fmt.Fprintf(w,"  Success: %s", resp.Success)
	fmt.Fprintf(w,"  Error: %s", resp.Error)

}

func main() {
	flag.BoolVar(&local, "local", false, "Defines if the environment is local or not")
	flag.Parse()

	initDB()
	defer db.Close()

	router := mux.NewRouter().StrictSlash(true)
	// router.HandleFunc("/", Index)
	// router.HandleFunc("/get", GetIndex)
	router.HandleFunc("/getindex", GetIndex)
	router.HandleFunc("/get/{udid}", GetLocation)
	router.HandleFunc("/set/{username}/{udid}/{latitude}&{longitude}", SetLocation)
	router.HandleFunc("/create/{udid1}&{udid2}", CreatePair)
	router.HandleFunc("/remove/{udid1}&{udid2}", RemovePair)
	router.HandleFunc("/getall/{udid}", GetAllLocations)
	router.HandleFunc("/notification/{udid}&{message}", sendNotification)

	log.Println("Server started on port :8080")

	log.Fatal(http.ListenAndServe(":8080", router))
}
