package peekaboodata

import "time"

// Peekaboo is
type Peekaboo struct {
	Token string `json:"token"`
	Name string `json:"name"`
	Baby string `json:"baby"`
	IsVideo bool `json:"isVideo"`
	Date time.Time `json:"date"`
	Comment string `json:"comment"`
	Orientation int `json:"orientation"`
	ThumbCreated bool `json:"thumbCreated"`
}