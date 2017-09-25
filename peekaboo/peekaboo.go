package main

import (
    "log"
    "net/http"
    "github.com/gorilla/mux" 
    "encoding/json"
    "github.com/mukk88/peekaboo-server/peekaboodata"
)

type tokenObj struct {
    Token string
}

func main() {
    router := mux.NewRouter().StrictSlash(true)
    router.HandleFunc("/peekaboo", allPeekaboos).Methods("GET")
    router.HandleFunc("/peekaboo", addPeekaboo).Methods("POST")
    log.Println("starting peekaboo server")
    log.Fatal(http.ListenAndServe(":6060", router))
}

func allPeekaboos(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
    // w.Header().Set("Access-Control-Allow-Origin", "http://hanabi.markwooym.com")
    w.Header().Set("Access-Control-Allow-Credentials", "true")

    log.Println("Getting all peeks..")
    dataStore := peekaboodata.NewDataStore()
    defer dataStore.CloseSession()
    
	allPeeks, err := dataStore.AllPeeks()
    if err != nil {
		allPeeks = []peekaboodata.Peekaboo {} 
	}
	value, err := json.Marshal(allPeeks)	
    w.Write(value)
}

func addPeekaboo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
    // w.Header().Set("Access-Control-Allow-Origin", "http://hanabi.markwooym.com")
    w.Header().Set("Access-Control-Allow-Credentials", "true")

    var peek peekaboodata.Peekaboo
    log.Println("parsing peek body..")
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&peek)
    if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }
    log.Println("Creating peek..")
	dataStore := peekaboodata.NewDataStore()
	defer dataStore.CloseSession()
    token := peekaboodata.GenerateToken(7)
    peek.Token = token
    dataStore.InsertPeek(&peek)
	if err != nil {
        log.Fatal(err)
        w.WriteHeader(400)
		return
    }
    tokenAsObj := tokenObj{token}
    body, _ := json.Marshal(&tokenAsObj)
    w.Write(body)
}