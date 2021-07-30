package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const mqttTopic = "device-stats"
const mqttBrokerEnvVar = "MQTT_BROKER"

//const defaultMqttBroker = "tcp://localhost:1883"
const defaultMqttBroker = "tcp://broker.hivemq.com:1883"

func main() {
	mqttBroker := os.Getenv(mqttBrokerEnvVar)
	if mqttBroker == "" {
		mqttBroker = defaultMqttBroker
	}

	client := mqtt.NewClient(mqtt.NewClientOptions().AddBroker(mqttBroker))
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	log.Println("connected to broker at", mqttBroker)

	exit := make(chan os.Signal)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Println("starting producer loop")

		for {
			location := "location-" + strconv.Itoa(rand.Intn(9)+1)
			sensor := "sensor-" + strconv.Itoa(rand.Intn(100)+1)
			temp := rand.Intn(50) + 1
			pressure := rand.Intn(80) + 1

			data := Data{Location: location, Sensor: sensor, Temp: temp, Pressure: pressure, Ts: time.Now().Unix()}
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Fatal(err)
			}

			log.Println(string(jsonData))

			token = client.Publish(mqttTopic, 0, false, jsonData)

			if token.Wait() && token.Error() != nil {
				log.Fatal("failed to publish", token.Error())
			}

			time.Sleep(2 * time.Second)
		}
	}()

	<-exit
	log.Println("exit requested")
}

type Data struct {
	Location string `json:"location"`
	Sensor   string `json:"sensor"`
	Temp     int    `json:"temp"`
	Pressure int    `json:"pressure"`
	Ts       int64  `json:"ts"`
}

const data = `{
    "temp": 24,
    "sensor": "sensor-24",
    "location": "location-4",
    "id": "a0ea8b99-e4f8-4c47-a01c-75343d2fe573",
    "pressure": 24,
    "ts": "2021-07-02 08:34:02.433",
    "_rid": "K1prAPie2N8CAAAAAAAAAA==",
    "_self": "dbs/K1prAA==/colls/K1prAPie2N8=/docs/K1prAPie2N8CAAAAAAAAAA==/",
    "_etag": "\"0100da1d-0000-0100-0000-60df01500000\"",
    "_attachments": "attachments/",
    "_ts": 1625227600
}`
