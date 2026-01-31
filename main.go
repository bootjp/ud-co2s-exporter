package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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

const (
	expectedDataFields = 3
	keyValueLength     = 2
	defaultBaudRate    = 115200
	defaultDataBits    = 8
	readHeaderTimeout  = 5 * time.Second
	startCommand       = "STA\r\n"
	stopCommand        = "STP\r\n"
	commandHeaderLines = 2
)

func parser(data string) (Status, error) {
	data = strings.TrimSuffix(data, "\r\n")
	splits := strings.Split(data, ",")
	result := Status{}

	if len(splits) != expectedDataFields {
		logger.Println("invalid format", data)
		return Status{}, ErrInvalidFormat
	}

	for _, split := range splits {
		keyValue := strings.Split(split, "=")
		if len(keyValue) != keyValueLength {
			continue
		}

		if err := parseKeyValue(keyValue[0], keyValue[1], &result); err != nil {
			logger.Println(err, split)
			return Status{}, ErrInvalidFormat
		}
	}

	return result, nil
}

func parseKeyValue(key, value string, result *Status) error {
	switch key {
	case "CO2":
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		result.co2 = parsed
	case "HUM":
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		result.hum = parsed
	case "TMP":
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		result.temp = parsed
	}
	return nil
}

func openSerialPort(device string) (serial.Port, error) {
	port, err := serial.Open(device, &serial.Mode{})
	if err != nil {
		return nil, fmt.Errorf("open serial port: %w", err)
	}
	mode := &serial.Mode{
		BaudRate: defaultBaudRate,
		Parity:   serial.NoParity,
		DataBits: defaultDataBits,
		StopBits: serial.OneStopBit,
	}
	if err := port.SetMode(mode); err != nil {
		return nil, fmt.Errorf("set mode: %w", err)
	}
	return port, nil
}

func writeCommand(port serial.Port, command string) error {
	if _, err := port.Write([]byte(command)); err != nil {
		return fmt.Errorf("write command %q: %w", strings.TrimSpace(command), err)
	}
	return nil
}

func recordMetrics() error {
	port, err := openSerialPort(*deviceName)
	if err != nil {
		return err
	}
	if err := writeCommand(port, startCommand); err != nil {
		return err
	}

	defer func() {
		if err := writeCommand(port, stopCommand); err != nil {
			logger.Println("write stop command:", err)
		}
		if err := port.Close(); err != nil {
			logger.Println("close port:", err)
		}
	}()

	scanner := bufio.NewScanner(port)
	// skip first 2 lines for command response
	for i := 0; i < commandHeaderLines; i++ {
		scanner.Scan()
	}

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

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	return nil
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

func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func indexHandler(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// language=HTML
	_, err := w.Write([]byte(`<html><head><title>UD-CO2S Exporter</title></head><body>` +
		`<a href="/metrics">metrics</a></body></html>`))
	if err != nil {
		return
	}
}

func run() error {
	e := errgroup.Group{}
	e.Go(recordMetrics)

	http.Handle("/metrics", metricsHandler())
	http.HandleFunc("/", indexHandler)
	e.Go(func() error {
		server := &http.Server{
			Addr:              *addr,
			ReadHeaderTimeout: readHeaderTimeout,
		}
		if err := server.ListenAndServe(); err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	})

	if err := e.Wait(); err != nil {
		return fmt.Errorf("run error: %w", err)
	}
	return nil
}

var (
	deviceName = kingpin.
			Flag("device.name", "Specify the UD-CO2S device path.(default /dev/ttyACM0)").
			Default("/dev/ttyACM0").
			String()
	addr = kingpin.
		Flag("exporter.addr", "Specifies the address on which the exporter listens (default :9233)").
		Default(":9233").
		String()
)

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
