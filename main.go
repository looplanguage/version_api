package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
	"time"
)

var schema = `
CREATE TABLE version (
	version text,
	released datetime,
	build int,
	prerelease bool,
	platform text,
	architecture text,
	installer bool,
	link text,
	PRIMARY KEY (version, build, platform, architecture)
)
`

type Version struct {
	Version      string    `json:"Version" db:"version"`
	Released     time.Time `json:"Released" db:"released"`
	Build        uint64    `json:"Build" db:"build"`
	Prerelease   bool      `json:"Prerelease" db:"prerelease"`
	Platform     string    `json:"Platform" db:"platform"`
	Architecture string    `json:"Architecture" db:"architecture"`
	Installer    bool      `json:"Installer" db:"installer"`
	Link         string    `json:"Link" db:"link"`
}

var db *sqlx.DB

func rowExists(query string, args ...interface{}) bool {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := db.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("error checking if row exists '%s' %v", args, err)
	}
	return exists
}

func latest(w http.ResponseWriter, r *http.Request) {
	versions := Version{}
	db.Select(&versions, "SELECT * FROM version ORDER BY released DESC LIMIT 1")
	json.NewEncoder(w).Encode(versions)
}

func latestStables(w http.ResponseWriter, r *http.Request) {
	versions := []Version{}
	db.Select(&versions, "SELECT * FROM version WHERE prerelease=false ORDER BY released DESC LIMIT 10")
	json.NewEncoder(w).Encode(versions)
}

func latestPrereleases(w http.ResponseWriter, r *http.Request) {
	versions := []Version{}
	db.Select(&versions, "SELECT * FROM version WHERE prerelease=true ORDER BY released DESC LIMIT 10")
	json.NewEncoder(w).Encode(versions)
}

func removeRelease(w http.ResponseWriter, r *http.Request) {
	version, exists := r.URL.Query()["v"]
	build, exists := r.URL.Query()["b"]
	platform, exists := r.URL.Query()["pl"]
	architecture, exists := r.URL.Query()["a"]
	key, exists := r.URL.Query()["key"]

	if !exists {
		fmt.Fprintf(w, "not all parameters specified")
		return
	}

	if key[0] != os.Getenv("LOOP_API_KEY") {
		fmt.Fprintf(w, "wrong key")
		return
	}

	if rowExists("SELECT * FROM version WHERE version=$1 AND build=$2 AND platform=$3 AND architecture=$4", version[0], build[0], platform[0], architecture[0]) {
		db.MustExec("DELETE FROM version WHERE version=$1 AND build=$2 AND platform=$3 AND architecture=$4", version[0], build[0], platform[0], architecture[0])
		fmt.Fprintf(w, "ok")
	} else {
		fmt.Fprintf(w, "version doesnt exist")
	}
}

// /add?v=1.0.0&r=1637702533&b=1&pr=0&pl=windows&a=x64&i=0&l=https%3A%2F%2Fs3.console.aws.amazon.com%2Fs3%2Fobject%2Floopartifacts%3Fregion%3Dus-east-2%26prefix%3DLoop%2Floop%2FPR-111-7%2Fjobs%2FLoop%2Floop%2FPR-111%2F7%2Floop.exe
func addRelease(w http.ResponseWriter, r *http.Request) {
	version, exists := r.URL.Query()["v"]
	released, exists := r.URL.Query()["r"]
	build, exists := r.URL.Query()["b"]
	prerelease, exists := r.URL.Query()["pr"]
	platform, exists := r.URL.Query()["pl"]
	architecture, exists := r.URL.Query()["a"]
	installer, exists := r.URL.Query()["i"]
	link, exists := r.URL.Query()["l"]
	key, exists := r.URL.Query()["key"]

	if !exists {
		fmt.Fprintf(w, "not all parameters specified")
		return
	}

	if key[0] != os.Getenv("LOOP_API_KEY") {
		fmt.Fprintf(w, "wrong key")
		return
	}

	if rowExists("SELECT * FROM version WHERE version=$1 AND build=$2 AND platform=$3 AND architecture=$4", version[0], build[0], platform[0], architecture[0]) {
		fmt.Fprintf(w, "version already present")
	} else {
		db.MustExec("INSERT INTO version (version, released, build, prerelease, platform, architecture, installer, link) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", version[0], released[0], build[0], prerelease[0], platform[0], architecture[0], installer[0], link[0])
		fmt.Fprintf(w, "ok")
	}
}

func main() {
	db, _ = sqlx.Connect("sqlite3", "database.db")

	_, tableCheck := db.Query("select * from version;")

	println(tableCheck)

	if tableCheck != nil {
		db.MustExec(schema)
	}

	http.HandleFunc("/latest", latest)
	http.HandleFunc("/latest/prerelease", latestPrereleases)
	http.HandleFunc("/latest/stable", latestStables)
	http.HandleFunc("/add", addRelease)
	http.HandleFunc("/remove", removeRelease)
	log.Fatal(http.ListenAndServe(":1515", nil))
}
