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
)

type GeoData struct {
	// Must be unique
	Id int
	// Generated automatically
	GeoHash uint64
	// User must specify these
	Latitude, Longitude float64
	Properties          *[]string
}

// GeoSlice sorted by GeoHash
type GeoSlice []*GeoData

func (s GeoSlice) Len() int           { return len(s) }
func (s GeoSlice) Less(i, j int) bool { return s[i].GeoHash < s[j].GeoHash }
func (s GeoSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GeoIdSlice sorted by ID
type GeoIdSlice []*GeoData

func (s GeoIdSlice) Len() int           { return len(s) }
func (s GeoIdSlice) Less(i, j int) bool { return s[i].Id < s[j].Id }
func (s GeoIdSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

var searchReady bool = false
var geoStore = make(map[string]GeoSlice)
var geoIdStore GeoIdSlice

const MAX_STEPS C.uint8_t = 26

// Debug logger. Remember to init flag.Parse() in main!!!
var debug *bool = flag.Bool("debugLib", false, "enable debug logging")

func Debugf(format string, args ...interface{}) {
	if *debug {
		log.Printf("DEBUG "+format, args...)
	}
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

func GetLocation(id int) (geodata *GeoData, err error) {
	if id == 0 || id >= len(geoIdStore) {
		return nil, errors.New("index out of range")
	}
	return geoIdStore[id-1], nil
}

// Add geo location data to search index.
func AddCoord(provider string, geoData *GeoData) {
	hash := geohashEncodeMax(geoData.Latitude, geoData.Longitude)
	geoData.GeoHash = hash
	geoStore[provider] = append(geoStore[provider], geoData)
	geoStore["default"] = geoStore[provider]
	geoIdStore = append(geoIdStore, geoData)
	searchReady = false
}

// Encode a geo hash to MAX(26) steps
func geohashEncodeMax(latitude, longitude float64) uint64 {
	var hash C.GeoHashBits
	C.geohashEncodeWGS84(C.double(latitude), C.double(longitude), MAX_STEPS, &hash)
	return uint64(hash.bits)
}

// Sort the list of geo so search can happen. Normally automatically trigger by search.
func initSearch() {
	for provider := range geoStore {
		sort.Sort(geoStore[provider])
	}
	sort.Sort(geoIdStore)
	searchReady = true // might cause race cond, but assume add doesn't happen often.
}

// Search a latitude/longitude in an area bounded (km) for known location data.
func SearchBound(provider string, latitude, longitude, bound float64) []*GeoData {
	if !searchReady {
		initSearch()
	}
	hashSteps := C.geohashEstimateStepsByRadius(C.double(bound * 1000))
	Debugf("Hash step: %v, for radius: %v km", hashSteps, bound)

	var hash C.GeoHashBits
	C.geohashEncodeWGS84(C.double(latitude), C.double(longitude), C.uint8_t(hashSteps), &hash)
	neighbours := getNeighbours(uint64(hash.bits), uint8(hashSteps))
	box := boundingBox(latitude, longitude, bound)

	locationsFound := make([]*GeoData, 0)
	geoStoreKeysLen := len(geoStore[provider])
	for nIdx := range neighbours {
		neighboursUpperLimit := (neighbours[nIdx] + 1) << uint((MAX_STEPS-hashSteps)*2)
		neighbours[nIdx] = neighbours[nIdx] << uint((MAX_STEPS-hashSteps)*2)
		Debugf("Normalized Neighbours Hash: %v to %v", neighbours[nIdx], neighboursUpperLimit)
		searchIdx := sort.Search(geoStoreKeysLen, func(i int) bool { return geoStore[provider][i].GeoHash >= neighbours[nIdx] })
		if searchIdx < geoStoreKeysLen { // Not found would turn index=N
			Debugf("found location?")
			// found location
			for i := searchIdx; i < geoStoreKeysLen; i++ {
				if geoStore[provider][i].GeoHash < neighboursUpperLimit {
					data := geoStore[provider][i]
					Debugf("filtering by lat/long: %v %v", data.Latitude, data.Longitude)
					Debugf("filtering by bounding box: %v %v %v %v", box[0], box[1], box[2], box[3])
					// filter by strict bounding box
					if ((data.Latitude >= box[0] && data.Latitude <= box[1]) || (data.Latitude <= box[0] && data.Latitude >= box[1])) &&
						((data.Longitude >= box[2] && data.Longitude <= box[3]) || (data.Longitude <= box[2] && data.Longitude >= box[3])) {
						locationsFound = append(locationsFound, data)
						Debugf("Search found location in GeoStore: %v", geoStore[provider][searchIdx])
					}
				} else {
					break
				}
			}
		}
	}
	return locationsFound
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

	Debugf("min lat: %v", minLatitude)
	Debugf("max lat: %v", maxLatitude)
	Debugf("min long: %v", minLongitude)
	Debugf("max long: %v", maxLongitude)

	return []float64{
		minLatitude,
		maxLatitude,
		minLongitude,
		maxLongitude,
	}
}
