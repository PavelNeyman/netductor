package main

import "testing"

func TestNormalizeBand(t *testing.T) {
	if normalizeBand("2g") != "2g" || normalizeBand("5G") != "5g" {
		t.Fatal("band")
	}
}

func TestBandFromHwmode(t *testing.T) {
	if bandFromHwmode("11g") != "2g" {
		t.Fatal("11g")
	}
	if bandFromHwmode("11a") != "5g" {
		t.Fatal("11a")
	}
}
