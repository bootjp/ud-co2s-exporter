package main

import "testing"

func TestParser(t *testing.T) {
	stat, err := parser([]byte("CO2=1012,HUM=35.2,TMP=29.6"))

	if err != nil {
		t.Error(err)
	}

	if stat.co2 != 1012 {
		t.Errorf("CO2: %f", stat.co2)
	}

	if stat.hum != 35.2 {
		t.Errorf("HUM: %f", stat.hum)

	}
	if stat.temp != 29.6 {
		t.Errorf("TMP: %f", stat.temp)
	}
}
