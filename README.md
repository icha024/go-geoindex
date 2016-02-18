#go-geoindex
======

###A simple in-memory geoindex for Go, based on the geohash-int library.

Proximity search largely based on the StackExchange discussion:
- http://gis.stackexchange.com/questions/18330/would-it-be-possible-to-use-geohash-for-proximity-searches/92331#92331

Original geohash-int library:
- https://github.com/yinqiwen/geohash-int

The modified fork of the geohash-int:
- https://github.com/mattsta/geohash-int

Some more documentations:
- https://github.com/yinqiwen/ardb/blob/master/doc/spatial-index.md
- https://matt.sh/redis-geo#_how-it-works


##Usage
------
###Add location:
Locations can be added to a provider (optional). This is useful for categorizing location types, eg. Italian food, Asian restaurant, French cruisine, etc
```go
locationId := 12345
prop := []string{"property1", "property2"}
AddLocation(providerId, &GeoData{Latitude: latitude, Longitude: longitude, Id: locationId, Properties: &prop})
```
###Search locations:
Search at lat/long (-32.1, 120.3) within a 12 km bound.
```go
locations := SearchBound(-32.2, 120.3, 12)
```

###Get location details:
Get details by ID.
```go
locationId := 12345
GetLocation(locationId)
```

##Limitations
------
* Search bounds are approximate square, and become gradually curved as the area increases.
* Bound distance are approximate and does not take the flat polar region into account.
* [Latitude/Longitude approximation:](http://stackoverflow.com/questions/1253499/simple-calculations-for-working-with-lat-lon-km-distance)
  * 1 deg latitude = 110.574 km
  * 1 deg longitude = 111.320*cos(latitude) km
* This geohash based proximity search does not search across the boundary at the poles (0) and at the international date line (180/-180).
