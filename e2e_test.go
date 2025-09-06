package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func TestEndToEnd(t *testing.T) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("open pty: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()

	*deviceName = tty.Name()

	srv := httptest.NewServer(promhttp.Handler())
	defer srv.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- recordMetrics()
	}()

	buf := make([]byte, 5)
	if _, err := io.ReadFull(ptmx, buf); err != nil {
		t.Fatalf("read start command: %v", err)
	}

	if string(buf) != "STA\r\n" {
		t.Fatalf("unexpected start command: %q", buf)
	}

	if _, err := ptmx.Write([]byte("OK\r\nOK\r\n")); err != nil {
		t.Fatalf("write response: %v", err)
	}

	if _, err := ptmx.Write([]byte("CO2=1012,HUM=35.2,TMP=29.6\r\n")); err != nil {
		t.Fatalf("write measurement: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	getMetrics := func() map[string]*dto.MetricFamily {
		resp, err := http.Get(srv.URL + "/metrics")
		if err != nil {
			t.Fatalf("get metrics: %v", err)
		}
		defer resp.Body.Close()

		parser := expfmt.NewTextParser(model.LegacyValidation)
		metrics, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			t.Fatalf("parse metrics: %v", err)
		}
		return metrics
	}

	assertGauge := func(metrics map[string]*dto.MetricFamily, name string, want float64) {
		mf, ok := metrics[name]
		if !ok || len(mf.GetMetric()) == 0 {
			t.Fatalf("%s metric not found", name)
		}
		got := mf.GetMetric()[0].GetGauge().GetValue()
		if got != want {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}

	metrics := getMetrics()
	assertGauge(metrics, "udco2s_co2ppm", 1012)
	assertGauge(metrics, "udco2s_temperature", 29.6)
	assertGauge(metrics, "udco2s_humidity", 35.2)

	if _, err := ptmx.Write([]byte("CO2=900,HUM=34.0,TMP=28.0\r\n")); err != nil {
		t.Fatalf("write measurement: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	metrics = getMetrics()
	assertGauge(metrics, "udco2s_co2ppm", 900)
	assertGauge(metrics, "udco2s_temperature", 28.0)
	assertGauge(metrics, "udco2s_humidity", 34.0)

	ptmx.Close()

	if err := <-errCh; err != nil {
		t.Logf("recordMetrics: %v", err)
	}
}
