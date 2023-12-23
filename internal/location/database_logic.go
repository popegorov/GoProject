package location

import (
	"context"
	"encoding/json"
	"net/http"
	"project/pkg/structs"
	"time"
)

func (l *LocationService) AddDriverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use http.post method", http.StatusBadRequest)
		return
	}

	var driver structs.Driver
	err := json.NewDecoder(r.Body).Decode(&driver)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	insertQuery := "INSERT INTO Drivers (id, address, lat, lng) VALUES ($1, $2, $3, $4)"

	lat, lng := 0, 0
	_, err = l.db.ExecContext(ctx, insertQuery, driver.ID, driver.Address, lat, lng)

	if err != nil {
		http.Error(w, "can't insert driver", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (l *LocationService) UpdateDriverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use http.post method", http.StatusBadRequest)
		return
	}

	var driver structs.Driver
	err := json.NewDecoder(r.Body).Decode(&driver)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*1))
	defer cancel()

	const query = "UPDATE Drivers SET lat=$1, lng=$2 WHERE id=$3"

	_, err = l.db.ExecContext(ctx, query, driver.Coors.Lat, driver.Coors.Lng, driver.ID)
	w.WriteHeader(http.StatusOK)
}
