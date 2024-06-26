package main

import (
	"bufio"
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
	"golang.org/x/sync/errgroup"
)

type Status struct {
	co2  float64
	hum  float64
	temp float64
}

var logger = log.Default()
var ErrInvalidFormat = errors.New("invalid format")

func parser(data string) (Status, error) {
	data = strings.TrimSuffix(data, "\r\n")
	splits := strings.Split(data, ",")
	result := Status{}

	if len(splits) != 3 {
		logger.Println("invalid format", data)
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
			logger.Println(err, split)
			return Status{}, ErrInvalidFormat
		}
	}

	return result, nil
}

func recordMetrics() error {
	port, err := serial.Open(*deviceName, &serial.Mode{})
	if err != nil {
		return err
	}
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	if err := port.SetMode(mode); err != nil {
		return err
	}
	_, err = port.Write([]byte("STA\r\n"))
	if err != nil {
		return err
	}

	defer func() {
		if _, err = port.Write([]byte("STP\r\n")); err != nil {
			logger.Fatalln(err)
		}
		if err := port.Close(); err != nil {
			logger.Fatalln(err)
		}
	}()

	scanner := bufio.NewScanner(port)
	// skip first 2 lines for command response
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		stat, err := parser(scanner.Text())
		if err != nil {
			logger.Println(err)
			continue
		}

		co2ppm.Set(stat.co2)
		temperature.Set(stat.temp)
		humidity.Set(stat.hum)
	}

	return err
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
	e := errgroup.Group{}
	e.Go(recordMetrics)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// language=HTML
		_, err := w.Write([]byte(`<html><head><title>UD-CO2S Exporter</title></head><body><a href="/metrics">metrics</a></body></html>`))
		if err != nil {
			return
		}
	})
	e.Go(func() error {
		return http.ListenAndServe(*addr, nil)
	})

	return e.Wait()
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
