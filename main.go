package main

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"gopkg.in/gorp.v1"
)

// Location struct for holding data from the Location table from the database
type Location struct {
	ID        int     `json:"-" db:"id"` //`json:"id" db:"id"`
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

func initDB() {
	gob.Register(&Location{})
	gob.Register(&Pair{})
	// connectionString := "root:root@tcp(localhost:8889)/ZING"
	connectionString := "root:288norfolk@/ZING"
	db, _ = sql.Open("mysql", connectionString)
	dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	dbmap.AddTableWithName(Location{}, "Locations").SetKeys(true, "ID")
	dbmap.AddTableWithName(Pair{}, "Pairs").SetKeys(true, "ID")
	dbmap.CreateTablesIfNotExists()
}

func main() {
	initDB()
	defer db.Close()

	router := mux.NewRouter().StrictSlash(true)
	// router.HandleFunc("/", Index)
	// router.HandleFunc("/get", GetIndex)
	router.HandleFunc("/get/{udid}", GetLocation)
	router.HandleFunc("/set/{udid}/{latitude}&{longitude}", SetLocation)
	router.HandleFunc("/create/{udid1}&{udid2}", CreatePair)
	router.HandleFunc("/remove/{udid1}&{udid2}", RemovePair)
	router.HandleFunc("/getall/{udid}", GetAllLocations)

	log.Fatal(http.ListenAndServe(":8080", router))
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

// GetLocation function searches the database for the location of the given udid
func GetLocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid := vars["udid"]
	var location Location
	err := dbmap.SelectOne(&location, "SELECT * FROM Locations WHERE udid=?", udid)
	if err != nil {
		log.Printf("Entry for %s not found", udid)
	}

	json.NewEncoder(w).Encode(location)
}

// SetLocation function sets the location for the given udid in the database
func SetLocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid := vars["udid"]
	latitude, _ := strconv.ParseFloat(vars["latitude"], 64)
	longitude, _ := strconv.ParseFloat(vars["longitude"], 64)

	//var location Location
	loc := &Location{0, "", 0, 0}
	err := dbmap.SelectOne(loc, "SELECT * FROM Locations WHERE udid=?", udid)
	if err != nil {
		log.Printf("Entry for %s not found", udid)
	}

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

	fmt.Fprintf(w, "Location set for %s: lat=%f lng=%f", udid, latitude, longitude)
}

// CreatePair function creates a pairing of the two given udids
func CreatePair(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid1 := vars["udid1"]
	udid2 := vars["udid2"]

	// Check if pair already exists (if it does, exit || re-auth session)
	var pair Pair
	err := dbmap.SelectOne(&pair, "SELECT * FROM Pairs WHERE udid_1=? AND udid_2=?", udid1, udid2)
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
}

// RemovePair function removes the pairing of the two given udids
func RemovePair(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid1 := vars["udid1"]
	udid2 := vars["udid2"]

	// Find pairing between the two udids
	var pair Pair
	err := dbmap.SelectOne(&pair, "SELECT * FROM Pairs WHERE udid_1=? AND udid_2=?", udid1, udid2)
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
}

// GetAllLocations function gets the locations of udids paired with the given udid
func GetAllLocations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	udid := vars["udid"]

	// TODO:
	// Find all pairings with udid
	var ids []string
	_, err := dbmap.Select(&ids, "SELECT udid_2 FROM Pairs WHERE udid_1=?", udid)

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
		err = dbmap.SelectOne(&loc, "SELECT * FROM Locations WHERE udid=?", id)
		if err != nil {
			log.Println("Select error: ", err)
		}

		locs = append(locs, loc)
	}

	// Return the locations to udid
	json.NewEncoder(w).Encode(locs)
}
