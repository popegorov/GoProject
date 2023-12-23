package location

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"project/pkg/structs"
	"time"
)

type LocationService struct {
	db *sql.DB
}

func NewLocationService(host string, port int, user string, password string, dbname string, mode string) *LocationService {
	connection := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s mode=%s",
		host, port, user, password, dbname, mode)

	db, err := sql.Open("postgres", connection)
	if err != nil {
		return nil
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return nil
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Drivers (id SERIAL PRIMARY KEY, address VARCHAR(255), lat DOUBLE PRECISION, lng DOUBLE PRECISION)")
	if err != nil {
		return nil
	}

	return &LocationService{db: db}
}

func (l *LocationService) GetDrivers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use http.get method", http.StatusBadRequest)
		return
	}

	var driver structs.Driver
	err := json.NewDecoder(r.Body).Decode(&driver)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	query := "SELECT id, address FROM Drivers WHERE (lat - $1) * (lat - $1) + (lng - $2) * (lng - $2) <= $3 * $3"

	rows, err := l.db.QueryContext(ctx, query, driver.Coors.Lat, driver.Coors.Lng, driver.Coors.Lat)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer rows.Close()

	var drivers []structs.Driver

	for rows.Next() {
		curDriver := structs.Driver{}

		err = rows.Scan(&curDriver.ID, &curDriver.Address)

		if err != nil {
			continue
		}

		drivers = append(drivers, curDriver)
	}

	jsonDrivers, err := json.Marshal(drivers)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jsonDrivers)
}
