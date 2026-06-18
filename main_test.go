package main

import (
	"errors"
	"testing"
)

func TestParser(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Status
		err  error
	}{
		{
			name: "valid",
			in:   "CO2=1012,HUM=35.2,TMP=29.6",
			out: Status{
				co2:  1012,
				hum:  35.2,
				temp: 29.6,
			},
			err: nil,
		},
		{
			name: "valid with line ending",
			in:   "CO2=1012,HUM=35.2,TMP=29.6\r\n",
			out: Status{
				co2:  1012,
				hum:  35.2,
				temp: 29.6,
			},
			err: nil,
		},
		{
			name: "valid with integer temperature",
			in:   "CO2=90,HUM=1.0,TMP=1\r\n",
			out: Status{
				co2:  90,
				hum:  1.0,
				temp: 1,
			},
			err: nil,
		},
		{
			name: "invalid field count",
			in:   "CO2=90",
			out:  Status{},
			err:  ErrInvalidFormat,
		},
		{
			name: "empty input",
			in:   "",
			out:  Status{},
			err:  ErrInvalidFormat,
		},
		{
			name: "invalid value",
			in:   "CO2=1012,HUM=bad,TMP=29.6",
			out:  Status{},
			err:  ErrInvalidFormat,
		},
		{
			name: "unknown key is ignored",
			in:   "CO2=1012,HUM=35.2,XXX=1",
			out: Status{
				co2: 1012,
				hum: 35.2,
			},
			err: nil,
		},
		{
			name: "missing equals is ignored",
			in:   "CO2=1012,HUM=35.2,TMP",
			out: Status{
				co2: 1012,
				hum: 35.2,
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat, err := parser(tt.in)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expect error got %v, want %v", stat, tt.out)
			}
			if stat != tt.out {
				t.Fatalf("got %v, want %v", stat, tt.out)
			}
		})
	}
}
