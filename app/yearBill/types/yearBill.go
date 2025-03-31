package types

import (
	"sync"
	"time"
)

type ECardCertificate struct {
	JsSessionId string `json:"js_session_id"`
	HallTicket  string `json:"hall_ticket"`
}

type JwcCertificate struct {
	GsSession    string `json:"gs_session"`
	Emaphome_WEU string `json:"emaphome_WEU"`
}

type PayData struct {
	Maps           map[string]float64
	BestRestaurant string
	RestaurantPay  float64
	EarlyTime      time.Time
	EarlyCount     int
	LastTime       time.Time
	LastCount      int
	OtherPay       float64
	LibraryPay     float64
	Mutex          sync.Mutex
}

type LearnData struct {
	LearnSum   map[string]int
	MostCourse string
	Most       int
	Eight      int
	Ten        int
	Sum        int
	Mutex      sync.Mutex
}

type BookData struct {
	Read     int
	Reading  int
	BookName string
	Longest  time.Time
}
