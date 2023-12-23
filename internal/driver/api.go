package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"project/pkg/structs"
	"syscall"
	"time"
)

const (
	Available int = iota
	NotAvailable
)

type DriverService struct {
	ResChannels            map[string]chan *sarama.ConsumerMessage
	LocationServiceAddress string
	Producer               *sarama.SyncProducer
	PartConsumer           *sarama.PartitionConsumer
	Client                 *mongo.Client
	DriversDB              *mongo.Collection
}

func NewDriverService(locationServiceAddress string) *DriverService {
	resChan := make(map[string]chan *sarama.ConsumerMessage)

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Can't create new mongo client %v\n", err)
		return nil
	}

	driversDB := client.Database("project").Collection("driversDB")

	return &DriverService{
		ResChannels:  resChan,
		Producer:     nil,
		PartConsumer: nil,
		Client:       client,
		DriversDB:    driversDB,
	}
}

func (d *DriverService) Start() {
	producer, err := sarama.NewSyncProducer([]string{"kafka:9092"}, nil)
	defer producer.Close()
	if err != nil {
		log.Fatalf("Can't create new producer %v\n", err)
	}

	consumer, err := sarama.NewConsumer([]string{"kafka:9092"}, nil)
	defer consumer.Close()
	if err != nil {
		log.Fatalf("Can't create new consumer %v\n", err)
	}

	partConsumer, err := consumer.ConsumePartition("created-trip", 0, sarama.OffsetNewest)
	defer partConsumer.Close()
	if err != nil {
		log.Fatalf("Can't create new partConsumer %v\n", err)
	}

	d.Producer = &producer
	d.PartConsumer = &partConsumer

	ctx := context.Background()
	go d.CreatedTripEventHandler(ctx)
	<-ctx.Done()
}

func (d *DriverService) CreatedTripEventHandler(ctx context.Context) {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	defer cancel()

	for {
		select {
		case msg, ok := <-(*d.PartConsumer).Messages():
			if !ok {
				log.Fatal("Channel closed, exiting goroutine")
				return
			}

			var event structs.Event
			err := json.Unmarshal(msg.Value, &event)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			var eventData structs.CreatedTripData
			err = json.Unmarshal(event.Data, &eventData)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			radius := 5
			jsonReq, err := json.Marshal(structs.Request{Lng: eventData.From.Lng,
				Lat:    eventData.From.Lat,
				Radius: radius})
			if err != nil {
				log.Println(err.Error())
				continue
			}

			response, err := http.NewRequest(http.MethodGet, d.LocationServiceAddress+"/drivers", bytes.NewReader(jsonReq))
			defer response.Body.Close()

			if err != nil {
				log.Println(err.Error())
				continue
			}

			rawDrivers, err := io.ReadAll(response.Body)
			var drivers []structs.Driver
			err = json.Unmarshal(rawDrivers, &drivers)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			var confirmedDriver structs.Driver
			indexDriver := -1
			tripID := ""
			for cntDriver, curDriver := range drivers {
				for i := 0; i < 10; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
					defer cancel()
					resp, _ := http.NewRequestWithContext(ctx, http.MethodGet, curDriver.Address, nil)

					if err != nil {
						log.Println(err.Error())
						break
					}

					if resp.Response.StatusCode == http.StatusGatewayTimeout {
						continue
					} else if resp.Response.StatusCode == http.StatusOK {
						indexDriver = cntDriver
						vars := mux.Vars(resp)
						tripID = vars["trip_id"]
						break
					}
				}
				if indexDriver != -1 {
					confirmedDriver = drivers[indexDriver]
					filter := bson.D{{"id", curDriver.ID}}
					updateBD := bson.D{{"set", bson.D{{"trip_id", tripID}, {"status", NotAvailable}}}}
					_, err = d.DriversDB.UpdateOne(context.TODO(), filter, updateBD)
					if err != nil {
						log.Fatal(err)
					}
					break
				}
			}
			opID := "id"

			if indexDriver != -1 {
				dataToSend := structs.TripDriverIDs{
					TripID:   tripID,
					DriverID: confirmedDriver.ID,
				}
				dataJson, err := json.Marshal(dataToSend)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				toSend := structs.Event{
					ID:              opID,
					Source:          "/driver",
					Type:            "trip.command.accept",
					DataContentType: "application/json",
					Time:            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					Data:            dataJson,
				}
				toSendJson, err := json.Marshal(toSend)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				resp := &sarama.ProducerMessage{
					Topic: "accept-trip",
					Value: sarama.StringEncoder(toSendJson),
				}
				_, _, err = (*d.Producer).SendMessage(resp)
				if err != nil {
					log.Println(err.Error())
				}
			} else {
				dataToSend := structs.TripReason{
					TripID: eventData.TripID,
					Reason: "No available drivers",
				}
				dataJson, err := json.Marshal(dataToSend)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				toSend := structs.Event{
					ID:              opID,
					Source:          "/driver",
					Type:            "trip.command.cancel",
					DataContentType: "application/json",
					Time:            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					Data:            dataJson,
				}
				toSendJson, err := json.Marshal(toSend)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				resp := &sarama.ProducerMessage{
					Topic: "cancel-trip",
					Value: sarama.StringEncoder(toSendJson),
				}
				_, _, err = (*d.Producer).SendMessage(resp)
				if err != nil {
					log.Println(err.Error())
				}
			}
		}
	}
}

func (d *DriverService) DriverCancelTripHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use method http.post", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	tripID := vars["trip_id"]

	var cancelationReason structs.CancelationReason
	err := json.NewDecoder(r.Body).Decode(&cancelationReason)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.D{{"id", cancelationReason.DriverID}}
	updateBD := bson.D{{"set", bson.D{{"trip_id", "none"}, {"status", Available}}}}
	_, err = d.DriversDB.UpdateOne(context.TODO(), filter, updateBD)
	if err != nil {
		log.Fatal(err)
	}

	tripReason := structs.TripReason{TripID: tripID, Reason: cancelationReason.Reason}
	dataJson, err := json.Marshal(tripReason)
	if err != nil {
		log.Println(err.Error())
		return
	}

	opID := "id"
	toSend := structs.Event{
		ID:              opID,
		Source:          "/driver",
		Type:            "trip.command.cancel",
		DataContentType: "application/json",
		Time:            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Data:            dataJson,
	}
	toSendJson, err := json.Marshal(toSend)
	if err != nil {
		log.Println(err.Error())
		return
	}
	resp := &sarama.ProducerMessage{
		Topic: "cancel-trip",
		Value: sarama.StringEncoder(toSendJson),
	}
	_, _, err = (*d.Producer).SendMessage(resp)
	if err != nil {
		log.Println(err.Error())
	}
}

func (d *DriverService) DriverStartTripHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use method http.post", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	tripID := vars["trip_id"]

	var ids structs.TripDriverIDs
	err := json.NewDecoder(r.Body).Decode(&ids)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	trip := structs.Trip{TripID: tripID}
	dataJson, err := json.Marshal(trip)
	if err != nil {
		log.Println(err.Error())
		return
	}

	opID := "id"
	toSend := structs.Event{
		ID:              opID,
		Source:          "/driver",
		Type:            "trip.command.start",
		DataContentType: "application/json",
		Time:            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Data:            dataJson,
	}
	toSendJson, err := json.Marshal(toSend)
	if err != nil {
		log.Println(err.Error())
		return
	}
	resp := &sarama.ProducerMessage{
		Topic: "start-trip",
		Value: sarama.StringEncoder(toSendJson),
	}
	_, _, err = (*d.Producer).SendMessage(resp)
	if err != nil {
		log.Println(err.Error())
	}
}

func (d *DriverService) DriverEndTripHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use method http.post", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	tripID := vars["trip_id"]

	var ids structs.TripDriverIDs
	err := json.NewDecoder(r.Body).Decode(&ids)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.D{{"id", ids.DriverID}}
	updateBD := bson.D{{"set", bson.D{{"trip_id", "none"}, {"status", Available}}}}
	_, err = d.DriversDB.UpdateOne(context.TODO(), filter, updateBD)
	if err != nil {
		log.Fatal(err)
	}

	tripReason := structs.Trip{TripID: tripID}
	dataJson, err := json.Marshal(tripReason)
	if err != nil {
		log.Println(err.Error())
		return
	}

	opID := "id"
	toSend := structs.Event{
		ID:              opID,
		Source:          "/driver",
		Type:            "trip.command.end",
		DataContentType: "application/json",
		Time:            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Data:            dataJson,
	}
	toSendJson, err := json.Marshal(toSend)
	if err != nil {
		log.Println(err.Error())
		return
	}
	resp := &sarama.ProducerMessage{
		Topic: "end-trip",
		Value: sarama.StringEncoder(toSendJson),
	}
	_, _, err = (*d.Producer).SendMessage(resp)
	if err != nil {
		log.Println(err.Error())
	}
}
