package main

import "testing"

func TestParser(t *testing.T) {
	tests := []struct {
		in  []byte
		out Status
	}{
		{
			[]byte("CO2=1012,HUM=35.2,TMP=29.6"),
			Status{
				co2:  1012,
				hum:  35.2,
				temp: 29.6,
			},
		},
		{
			[]byte("CO2=1012,HUM=35.2,TMP=29.6\n\r"),
			Status{
				co2:  1012,
				hum:  35.2,
				temp: 29.6,
			},
		},
		{
			[]byte("CO2=1012,HUM=35.2,TMP=29.6\r\n"),
			Status{
				co2:  1012,
				hum:  35.2,
				temp: 29.6,
			},
		},
		{
			[]byte("CO2=90,HUM=1.0,TMP=1\r\n"),
			Status{
				co2:  90,
				hum:  1.0,
				temp: 1,
			},
		},
	}

	for _, tt := range tests {
		stat, err := parser(tt.in)
		if err != nil {
			t.Error(err)
		}
		if stat != tt.out {
			t.Errorf("got %v, want %v", stat, tt.out)
		}
	}
}