package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.bug.st/serial"
)

type Status struct {
	co2  float64
	hum  float64
	temp float64
}

var logger = log.Default()
var ErrInvalidFormat = errors.New("invalid format")

func parser(data []byte) (Status, error) {
	line := string(data)
	line = strings.Replace(line, "\r", "", -1)
	line = strings.Replace(line, "\n", "", -1)
	splits := strings.Split(line, ",")
	result := Status{}

	if len(splits) != 3 {
		logger.Println("invalid format", line)
		return Status{}, ErrInvalidFormat
	}

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
			logger.Println(err, keyValue[1])
			return Status{}, ErrInvalidFormat
		}
	}

	return result, nil
}

func recordMetrics() {
	port, err := serial.Open(*deviceName, &serial.Mode{})
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
		logger.Fatal(err)
	}
	_, err = port.Write([]byte("STA\r\n"))
	if err != nil {
		logger.Println(err)
	}

	buff := make([]byte, 256)
	for {
		n, err := port.Read(buff)
		if err != nil {
			logger.Fatalln(err)
		}
		if n == 0 {
			continue
		}

		stat, err := parser(buff[:n])
		if err != nil {
			log.Println(err)
			continue
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

func run() error {
	go recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// language=HTML
		_, err := w.Write([]byte(`<html><head><title>UD-CO2S Exporter</title></head><body><a href="/metrics">metrics</a></body></html>`))
		if err != nil {
			return
		}
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		return err
	}

	return nil
}

var (
	deviceName = kingpin.Flag("device.name", "Specify the UD-CO2S device path.(default /dev/ttyACM0)").Default("/dev/ttyACM0").String()
	addr       = kingpin.Flag("exporter.addr", "Specifies the address on which the exporter listens (default :9233)").Default(":9233").String()
)

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
