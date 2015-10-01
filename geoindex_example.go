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
*/
import "C"
import (
	"flag"
	"log"
)

func main() {
	log.Println("Start Go-Geoindex")
	flag.Parse() // Parse debug flag
	var hash C.GeoHashBits

	C.geohashEncodeWGS84(-32.1, 120.3, 15, &hash)
	Debugf("Encoded hash in %v steps, bits: %d", hash.step, uint64(hash.bits))
	Debugf("Shift this by %v --> %v", uint((MAX_STEPS-hash.step)*2), hash.bits<<uint((MAX_STEPS-hash.step)*2))
	Debugf("test geohash encode function: %d", uint64(geohashEncodeMax(-32.1, 120.3)))
	Debugf("+1 Encoded hash in %v steps, bits: %d", hash.step, uint64(hash.bits+1))
	Debugf("+ 1Shift this by %v --> %v", uint((MAX_STEPS-hash.step)*2), (hash.bits+1)<<uint((MAX_STEPS-hash.step)*2))

	var area C.GeoHashArea
	C.geohashDecodeWGS84(hash, &area)
	Debugf("Decoding area lat: %v %v, long: %v %v", area.latitude.min, area.latitude.max, area.longitude.min, area.longitude.max)

	nArr := getNeighbours(uint64(hash.bits), uint8(hash.step))

	for i, v := range nArr {
		Debugf("-- nArr %v : %v", i, v)
	}

	// Test add
	prop := []string{"k", "vv"}
	AddCoord(&GeoData{Latitude: -32.1, Longitude: 120.3, Name: "0", Properties: &prop})
	AddCoord(&GeoData{Latitude: -33.1, Longitude: 121.3})
	AddCoord(&GeoData{Latitude: -32.1, Longitude: 120.3})

	// Check it
	for _, ele := range geoDataStore {
		Debugf("Geo store keys: %v", ele)
	}

	// Test find locations
	locations := SearchBound(-32.2, 120.3, 12)
	Debugf("There are %v locations found", len(locations))
	for _, loc := range locations {
		Debugf("Found location: %v", loc)
	}

	geo0, err := GetLocation("3139639761105107-0")
	if err != nil {
		Debugf("Error getting details")
	}
	geo1, err := GetLocation("3139639761105107-1")
	if err != nil {
		Debugf("Error getting details")
	}
	Debugf("Find by location id: %v -> %v (prop: %v=%v)", "3139639761105107[0]", geo0,
		&(*geo0.Properties)[0], &(*geo0.Properties)[1])
	Debugf("Find by location id: %v -> %v", "3139639761105107[1]", geo1)
}
