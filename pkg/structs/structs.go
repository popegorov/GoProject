package structs

type Driver struct {
	ID      string      `json:"id"`
	Status  int         `json:"status"`
	Address string      `json:"address"`
	Coors   Coordinates `json:"coors"`
}

type Event struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	Type            string `json:"type"`
	DataContentType string `json:"datacontenttype"`
	Time            string `json:"time"`
	Data            []byte `json:"data"`
}

type TripDriverIDs struct {
	TripID   string `json:"trip_id"`
	DriverID string `json:"driver_id"`
}

type TripReason struct {
	TripID string `json:"trip_id"`
	Reason string `json:"reason"`
}

type Trip struct {
	TripID string `json:"trip_id"`
}

type CancelationReason struct {
	DriverID string `json:"driver_id"`
	TripID   string `json:"trip_id"`
	Reason   string `json:"reason"`
}

type TripPrice struct {
	Currency int `json:"currency"`
	Amount   int `json:"amount"`
}

type Coordinates struct {
	Lat int `json:"lat"`
	Lng int `json:"lng"`
}

type CreatedTripData struct {
	TripID  string      `json:"trip_id"`
	OfferID string      `json:"offer_id"`
	Price   TripPrice   `json:"price"`
	Status  string      `json:"status"`
	From    Coordinates `json:"from"`
	To      Coordinates `json:"to"`
}

type Request struct {
	Lng    int `json:"lng"`
	Lat    int `json:"lat"`
	Radius int `json:"radius"`
}
