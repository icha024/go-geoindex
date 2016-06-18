/*
 * Copyright (c) 2015, Ian Chan <icha024@gmail.com>.
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *  * Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *  * Neither the name of Redis nor the names of its contributors may be used
 *    to endorse or promote products derived from this software without
 *    specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS
 * BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF
 * THE POSSIBILITY OF SUCH DAMAGE.
 */

package geoindex

/*
#include "geohash.h"
#include "geohash_helper.h"
#include <math.h>
#cgo LDFLAGS: -lm
*/
import "C"
import (
	"errors"
	"flag"
	"github.com/cznic/sortutil"
	"log"
	"math"
	"sort"
	"sync"
)

// GeoData location representation
type GeoData struct {
	// Generated automatically, must be unique.
	ID int
	// Generated automatically
	GeoHash uint64
	// User must specify these
	Latitude, Longitude float64
	Properties          *[]string
}

// GeoHashSlice sorted by GeoHash
type geoHashSlice []*GeoData

func (s geoHashSlice) Len() int           { return len(s) }
func (s geoHashSlice) Less(i, j int) bool { return s[i].GeoHash < s[j].GeoHash }
func (s geoHashSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GeoIDSlice sorted by ID
type geoIDSlice []*GeoData

const maxSteps uint8 = 26

var geoHashStore geoHashSlice
var geoIDStore geoIDSlice
var mutex = &sync.Mutex{}
var searchReady = false

// Debug logger. Remember to init flag.Parse() in main!!!
var debug = flag.Bool("debugLib", false, "enable debug logging")

func debugf(format string, args ...interface{}) {
	if *debug {
		log.Printf("DEBUG "+format, args...)
	}
}

// AddLocation data to search index, returns the location ID assigned.
func AddLocation(geoData *GeoData) (id int, err error) {
	if geoData.ID != 0 || geoData.GeoHash != 0 {
		return 0, errors.New("GeoHash and ID field should not be specified, it will be generated internally.")
	}
	hash := geohashEncodeMax(geoData.Latitude, geoData.Longitude)
	geoData.GeoHash = hash
	mutex.Lock()
	geoData.ID = len(geoIDStore)
	geoIDStore = append(geoIDStore, geoData)
	geoHashStore = append(geoHashStore, geoData)
	mutex.Unlock()
	searchReady = false
	return geoData.ID, nil
}

// GetLocation data for a location ID.
func GetLocation(id int) (geodata *GeoData, err error) {
	if id >= len(geoIDStore) {
		return nil, errors.New("index out of range")
	}
	return geoIDStore[id], nil
}

const mercatorMax float64 = 20037726.37

func geohashEstimateStepsByRadius(rangeMeter float64) uint8 {
	var step uint8 = 1
	for i := 0; i < 26; i++ {
		rangeMeter *= 2
		step++
		if rangeMeter < 0 || rangeMeter > mercatorMax {
			break
		}
	}
	step--
	if step == 0 || step > 26 {
		return 26
	}
	return step
}

// SearchLocations around latitude/longitude in bounded area (km) for known location points.
func SearchLocations(latitude, longitude, bound float64) []*GeoData {
	if !searchReady {
		InitSearch()
	}
	hashSteps := geohashEstimateStepsByRadius(bound * 1000)
	debugf("Hash step: %v, for radius: %v km", hashSteps, bound)

	var hash C.GeoHashBits
	C.geohashEncodeWGS84(C.double(latitude), C.double(longitude), C.uint8_t(hashSteps), &hash)
	neighbours := getNeighbours(uint64(hash.bits), uint8(hashSteps))
	box := boundingBox(latitude, longitude, bound)

	initialSize := 128
	var locationsFound = make([]*GeoData, initialSize) // TODO initial size????
	locFoundCount := 0
	geoStoreKeysLen := len(geoHashStore)
	for nIdx := range neighbours {
		neighboursUpperLimit := (neighbours[nIdx] + 1) << uint((maxSteps-hashSteps)*2)
		neighbours[nIdx] = neighbours[nIdx] << uint((maxSteps-hashSteps)*2)
		debugf("Normalized Neighbours Hash: %v to %v", neighbours[nIdx], neighboursUpperLimit)
		searchIdx := sort.Search(geoStoreKeysLen, func(i int) bool { return geoHashStore[i].GeoHash >= neighbours[nIdx] })
		if searchIdx < geoStoreKeysLen { // Not found would turn index=N
			debugf("found location?")
			// found location
			for i := searchIdx; i < geoStoreKeysLen; i++ {
				if geoHashStore[i].GeoHash < neighboursUpperLimit {
					//var data *GeoData
					data := geoHashStore[i]
					debugf("filtering by lat/long: %v %v", data.Latitude, data.Longitude)
					debugf("filtering by bounding box: %v %v %v %v", box[0], box[1], box[2], box[3])
					// filter by strict bounding box
					if ((data.Latitude >= box[0] && data.Latitude <= box[1]) || (data.Latitude <= box[0] && data.Latitude >= box[1])) &&
						((data.Longitude >= box[2] && data.Longitude <= box[3]) || (data.Longitude <= box[2] && data.Longitude >= box[3])) {
						if locFoundCount < initialSize {
							locationsFound[locFoundCount] = data
						} else {
							locationsFound = append(locationsFound, data)
						}
						locFoundCount++
						debugf("Search found location in geoHashStore: %v", geoHashStore[searchIdx])
					}
				} else {
					break
				}
			}
		}
	}
	return locationsFound[:locFoundCount]
}

// InitSearch should be call after all locations are added. This sort the list of geo so binary search can be used.
func InitSearch() {
	sort.Sort(geoHashStore)
	searchReady = true // might cause race cond if data are added and search at the same time
}

func getNeighbours(hashBits uint64, steps uint8) []uint64 {
	var neighbours C.GeoHashNeighbors
	var hash C.GeoHashBits
	hash.bits = C.uint64_t(hashBits)
	hash.step = C.uint8_t(steps)
	C.geohashNeighbors(&hash, &neighbours)

	neighbourArr := sortutil.Uint64Slice{
		uint64(hashBits),
		uint64(neighbours.north.bits),
		uint64(neighbours.east.bits),
		uint64(neighbours.west.bits),
		uint64(neighbours.south.bits),
		uint64(neighbours.north_east.bits),
		uint64(neighbours.south_east.bits),
		uint64(neighbours.north_west.bits),
		uint64(neighbours.south_west.bits),
	}
	//	sort.Sort(neighbourArr)
	if steps <= 6 {
		sortutil.Dedupe(neighbourArr) // Can have duplicates if search range is large (>~5000km)
	}
	return neighbourArr
}

// Encode a geo hash to MAX(26) steps
func geohashEncodeMax(latitude, longitude float64) uint64 {
	var hash C.GeoHashBits
	C.geohashEncodeWGS84(C.double(latitude), C.double(longitude), C.uint8_t(maxSteps), &hash)
	return uint64(hash.bits)
}

// The approximate conversions are (doesn't fully correct for the Earth's polar flattening):
// Latitude: 1 deg = 110.574 km
// Longitude: 1 deg = 111.320*cos(latitude) km
// See: http://stackoverflow.com/questions/1253499/simple-calculations-for-working-with-lat-lon-km-distance
// Returns: min/max latitude, min/max longitude.
func boundingBox(latitude, longitude, boundKm float64) []float64 {
	latDiff := boundKm / 110.574
	longDiff := boundKm / (111.320 * math.Cos(latitude))
	minLatitude := latitude - latDiff
	maxLatitude := latitude + latDiff
	minLongitude := longitude - longDiff
	maxLongitude := longitude + longDiff

	debugf("min lat: %v", minLatitude)
	debugf("max lat: %v", maxLatitude)
	debugf("min long: %v", minLongitude)
	debugf("max long: %v", maxLongitude)

	return []float64{
		minLatitude,
		maxLatitude,
		minLongitude,
		maxLongitude,
	}
}
