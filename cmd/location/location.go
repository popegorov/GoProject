package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"project/internal/location"
	"strconv"
)

func main() {
	err := godotenv.Load(".env.dev")
	if err != nil {
		log.Fatal("Не удалось загрузить файл .env.dev")
	}

	port := os.Getenv("LOCATION_PORT")

	numPort, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	locationSVC := location.NewLocationService("host", numPort, "user", "password", "Drivers", "mode")

	http.HandleFunc("/api/v1/add", locationSVC.AddDriverHandler)
	http.HandleFunc("/api/v1/update", locationSVC.UpdateDriverHandler)
	http.HandleFunc("/api/v1/drivers", locationSVC.GetDrivers)

	err = http.ListenAndServe("localhost:"+port, nil)

	if err != nil {
		log.Fatal(err.Error())
		return
	}
}
