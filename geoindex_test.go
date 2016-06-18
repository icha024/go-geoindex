package geoindex

import (
	"log"
	"math/rand"
	"testing"
)

func TestAddAndSearch(t *testing.T) {
	// Add locations
	lng0 := 13.361389
	lat0 := 38.115556
	prop0 := []string{"Palermo"}
	locationID0, err := AddLocation(&GeoData{Latitude: lat0, Longitude: lng0, Properties: &prop0})
	if err != nil {
		t.Error("Error adding location for Palermo")
	}
	if locationID0 != 0 {
		t.Error("Location ID for Palermo incorrect")
	}

	lng1 := 15.087269
	lat1 := 37.502669
	prop1 := []string{"Catania"}
	locationID1, err := AddLocation(&GeoData{Latitude: lat1, Longitude: lng1, Properties: &prop1})
	if err != nil {
		t.Error("Error adding location for Catania")
	}
	if locationID1 != 1 {
		t.Error("Location ID for Catania incorrect")
	}

	// Check details added
	geoData0, err := GetLocation(locationID0)
	if err != nil {
		t.Error("Error getting location for Palermo")
	}
	if geoData0.Latitude != lat0 || geoData0.Longitude != lng0 || (*geoData0.Properties)[0] != "Palermo" {
		t.Error("Incorrect details for Palermo")
	}

	geoData1, err := GetLocation(locationID1)
	if err != nil {
		t.Error("Error getting location for Catania")
	}
	if geoData1.Latitude != lat1 || geoData1.Longitude != lng1 || (*geoData1.Properties)[0] != "Catania" {
		t.Error("Incorrect details for Catania")
	}

	// Check search - distance beween them is 166274.15156960039 KM
	// The calculation round this to (175KM - FIXME)
	resGeoData := SearchLocations(lat0, lng0, 165)
	// log.Printf("Locations found: %v", len(resGeoData))
	if len(resGeoData) != 1 || (*resGeoData[0].Properties)[0] != "Palermo" {
		t.Error("Expected self location not found")
	}
	resGeoData = SearchLocations(lat0, lng0, 180)
	// log.Printf("Locations found: %v", len(resGeoData))
	if len(resGeoData) != 2 {
		t.Error("Expected second location not found")
	}
	log.Println("Test completed")
}

func BenchmarkAdd(b *testing.B) {
	for n := 0; n < b.N; n++ {
		lng := 5.0 + rand.Float64()*5.0
		lat := 5.0 + rand.Float64()*5.0
		prop := []string{"property1", "property2"}
		//locationID, err :=
		AddLocation(&GeoData{Latitude: lat, Longitude: lng, Properties: &prop})
	}
}

func BenchmarkSearch(b *testing.B) {
	for n := 0; n < 500000; n++ {
		lng := 5.0 + rand.Float64()*5.0
		lat := 5.0 + rand.Float64()*5.0
		prop := []string{"property1", "property2"}
		//locationID, err :=
		AddLocation(&GeoData{Latitude: lat, Longitude: lng, Properties: &prop})
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		lat := 5.0 + rand.Float64()*5.0
		lng := 5.0 + rand.Float64()*5.0
		bound := 5.0 // KM
		SearchLocations(lat, lng, bound)
	}
}
