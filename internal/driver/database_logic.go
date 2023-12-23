package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"project/pkg/structs"
)

func (d *DriverService) NewDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use method http.post", http.StatusMethodNotAllowed)
		return
	}

	var driverID string
	err := json.NewDecoder(r.Body).Decode(&driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	driverFilter := bson.D{{"id", driverID}}

	err = d.DriversDB.FindOne(context.TODO(), driverFilter).Decode(nil)
	if !errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "driver with this ID already exists", http.StatusBadRequest)
		log.Fatal(err)
		return
	}

	driverAddress := r.RemoteAddr
	newDriver := structs.Driver{
		ID:      driverID,
		Status:  Available,
		Address: driverAddress,
	}

	_, err = d.DriversDB.InsertOne(context.TODO(), newDriver)
	if err != nil {
		log.Fatal(err)
		return
	}

	jsonDriver, err := json.Marshal(newDriver)
	_, err = http.Post(d.LocationServiceAddress, "application/json", bytes.NewReader(jsonDriver))
	if err != nil {
		log.Fatal("can't send http.post to locationSVC")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (d *DriverService) GetDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use http.get method", http.StatusMethodNotAllowed)
		return
	}

	driverID := 0
	err := json.NewDecoder(r.Body).Decode(&driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Fatal(err)
		return
	}
	defer r.Body.Close()

	driverFilter := bson.D{{"id", driverID}}

	var driver structs.Driver
	err = d.DriversDB.FindOne(context.TODO(), driverFilter).Decode(&driver)
	if errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "incorrect driver id", http.StatusBadRequest)
		log.Fatal(err)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Fatal(err)
		return
	}

	jsonDriver, err := json.Marshal(driver)
	if err != nil {
		log.Fatal(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jsonDriver)
}
