package peekaboodata

import (
	"os"
	"log"
	"gopkg.in/mgo.v2"
	"math/rand"
	"time"
	"gopkg.in/mgo.v2/bson"
	"errors"
)

// DataStore is the database wrapper
type DataStore struct {
	session *mgo.Session
} 

func getMongoUrl() string {
    goEnv := os.Getenv("goenv")
    if goEnv == "dev" {
        return "localhost:27017"
    }
    return "mongodb://peekaboo:peekaboo123@ds147964.mlab.com:47964/peekaboo"
}

func (ds *DataStore) setupSession() {
	session, err := mgo.Dial(getMongoUrl())
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)
	ds.session = session
}

func(ds *DataStore) AllPeeks(baby string) ([]Peekaboo, error) {
	result := []Peekaboo{}
	collection := ds.session.DB("peekaboo").C("peeks")
	err := collection.Find(bson.M{"baby": baby}).Sort("-date").All(&result)
	return result, err
}

func (ds *DataStore) InsertPeek(peek *Peekaboo) error {
	collection := ds.session.DB("peekaboo").C("peeks")
	var result Peekaboo
	err := collection.Find(bson.M{"name": peek.Name}).One(&result)
	if err != nil {
		if err.Error() == "not found" {
			log.Println("does not exist, create it")
			return collection.Insert(peek)
		}
		return err 
	}
	log.Println("peek already exists, not creating..")
	return errors.New("already exists")
}

func (ds *DataStore) UpdatePeek(peek *Peekaboo) error {
	collection := ds.session.DB("peekaboo").C("peeks")
	_, err :=  collection.Upsert(
		bson.M{"token": peek.Token},
		peek,
	)
	return err
}

func (ds *DataStore) DeletePeek(peek *Peekaboo) error {
	collection := ds.session.DB("peekaboo").C("peeks")
	err :=  collection.Remove(
		bson.M{"token": peek.Token},
	)
	return err
}

func (ds *DataStore) CloseSession() {
	ds.session.Close()
}

// NewDataStore is the constructor for DataStore
func NewDataStore() *DataStore {
	ds := DataStore{}
	ds.setupSession()
	return &ds
} 

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func GenerateToken(n int) string {
	rand.Seed(time.Now().UnixNano())
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}
