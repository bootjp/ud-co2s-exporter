package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.bug.st/serial"
)

type Status struct {
	co2  float64
	hum  float64
	temp float64
}

func parser(data []byte) (Status, error) {
	// CO2=1012,HUM=35.2,TMP=29.6
	line := string(data)
	line = strings.Replace(line, "\r", "", -1)
	line = strings.Replace(line, "\n", "", -1)
	splits := strings.Split(line, ",")

	result := Status{}
	for _, split := range splits {
		var err error
		keyValue := strings.Split(split, "=")
		if len(keyValue) != 2 {
			continue
		}

		switch keyValue[0] {
		case "CO2":
			result.co2, err = strconv.ParseFloat(keyValue[1], 64)
		case "HUM":
			result.hum, err = strconv.ParseFloat(keyValue[1], 64)
		case "TMP":
			result.temp, err = strconv.ParseFloat(keyValue[1], 64)
		}
		if err != nil {
			return Status{}, err
		}
	}

	return result, nil
}

func recordMetrics() {
	port, err := serial.Open("/dev/ttyACM0", &serial.Mode{})
	if err != nil {
		log.Fatal(err)
	}
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	if err := port.SetMode(mode); err != nil {
		log.Fatal(err)
	}
	_, err = port.Write([]byte("STA\r\n"))
	if err != nil {
		log.Fatal(err)
	}

	buff := make([]byte, 100)
	for {
		n, err := port.Read(buff)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}

		stat, err := parser(buff[:n])
		if err != nil {
			log.Fatal(err)
		}

		co2ppm.Set(stat.co2)
		temperature.Set(stat.temp)
		humidity.Set(stat.hum)
	}
}

var (
	co2ppm = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "udco2s_co2ppm",
		Help: "The co2 ppm value",
	})
	temperature = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "udco2s_temperature",
		Help: "The temperature value (Celsius)",
	})
	humidity = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "udco2s_humidity",
		Help: "The humidity value",
	})
)

func main() {
	go recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2233", nil)
	if err != nil {
		return
	}
}
