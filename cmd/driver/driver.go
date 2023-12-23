package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"project/internal/driver"
)

func main() {
	err := godotenv.Load(".env.dev")
	if err != nil {
		log.Fatal("Не удалось загрузить файл .env.dev")
	}

	port := os.Getenv("DRIVER_PORT")

	driverSVC := driver.NewDriverService("localhost:" + port)

	http.HandleFunc("/api/v1/trip/{trip_id}/cancel", driverSVC.DriverCancelTripHandler)
	http.HandleFunc("/api/v1/trip/{trip_id}/start", driverSVC.DriverStartTripHandler)
	http.HandleFunc("/api/v1/trip/{trip_id}/end", driverSVC.DriverEndTripHandler)

	err = http.ListenAndServe("localhost:"+port, nil)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	driverSVC.Start()
}
