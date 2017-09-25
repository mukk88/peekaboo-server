package peekaboodata

import (
	"gopkg.in/mgo.v2"
	"math/rand"
	"time"
	// "gopkg.in/mgo.v2/bson"
)

// DataStore is the database wrapper
type DataStore struct {
	session *mgo.Session
} 

func (ds *DataStore) setupSession() {
	session, err := mgo.Dial("localhost:27017")
	// session, err := mgo.Dial("mongodb://peekaboo:peekaboo123@ds147964.mlab.com:47964/peekaboo")
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)
	ds.session = session
}

func(ds *DataStore) AllPeeks() ([]Peekaboo, error) {
	result := []Peekaboo{}
	collection := ds.session.DB("peekaboo").C("peeks")
	err := collection.Find(nil).Sort("-date").All(&result)
	return result, err
}

func (ds *DataStore) InsertPeek(peek *Peekaboo) error {
	collection := ds.session.DB("peekaboo").C("peeks")
	return collection.Insert(peek)
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
