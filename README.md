# go-geoindex

### A simple in-memory geoindex for Go, based on the geohash-int library.

Proximity search largely based on the StackExchange discussion:
- http://gis.stackexchange.com/questions/18330/would-it-be-possible-to-use-geohash-for-proximity-searches/92331#92331

Original geohash-int library:
- https://github.com/yinqiwen/geohash-int

The modified fork of the geohash-int:
- https://github.com/mattsta/geohash-int

Some more documentations:
- https://github.com/yinqiwen/ardb/blob/master/doc/spatial-index.md
- https://matt.sh/redis-geo#_how-it-works


## Usage
### Add location:
```go
prop := []string{"property1", "property2"}
locationID, err := geoindex.AddLocation(&GeoData{Latitude: latitude, Longitude: longitude, Properties: &prop})
```

**Initialize search index** (optional) - Otherwise it will trigger an indexed on the first search.
```go
geoindex.InitSearch()
```

###Search locations:
Search at latitude/longitude (-32.1, 120.3) within a 12 km bound.
```go
locations := geoindex.SearchBound(-32.2, 120.3, 12)
```

###Get location details:
Get details by ID.
```go
locationID := 12345 // Either from add operation, or from search results.
geoindex.GetLocation(locationID)
```

## Performance
Tested with a simple local HTTP server (no cache) in plain Go lang 1.6 and Apache Bench. Using a HTTP GET search operation that returns data in GeoJson format, on my 6x vCore i7 CPU (3.2Ghz Haswell) Ubuntu 16.04 desktop.

#### (Basic) Total 30 locations in the system, search 10 km bound to return 2 records.

Over 26,000 TPS for basic dataset when exposed as a HTTP services

```
Concurrency Level:      10
Time taken for tests:   1.871 seconds
Complete requests:      50000
Failed requests:        0
Total transferred:      23150000 bytes
HTML transferred:       17250000 bytes
Requests per second:    26719.59 [#/sec] (mean)
Time per request:       0.374 [ms] (mean)
Time per request:       0.037 [ms] (mean, across all concurrent requests)
Transfer rate:          12081.22 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       1
Processing:     0    0   0.1      0       3
Waiting:        0    0   0.1      0       3
Total:          0    0   0.1      0       3

Percentage of the requests served within a certain time (ms)
  50%      0
  66%      0
  75%      0
  80%      0
  90%      0
  95%      1
  98%      1
  99%      1
 100%      3 (longest request)
```

#### (Normal) Total 443,969 locations in the system, search 1 km bound to return 42 records.

Around 14,000 TPS for norminal conditions.

```
Concurrency Level:      10
Time taken for tests:   3.591 seconds
Complete requests:      50000
Failed requests:        0
Total transferred:      324150000 bytes
HTML transferred:       319300000 bytes
Requests per second:    13923.68 [#/sec] (mean)
Time per request:       0.718 [ms] (mean)
Time per request:       0.072 [ms] (mean, across all concurrent requests)
Transfer rate:          88151.60 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       3
Processing:     0    1   1.4      1      79
Waiting:        0    1   1.4      0      79
Total:          0    1   1.4      1      79

Percentage of the requests served within a certain time (ms)
  50%      1
  66%      1
  75%      1
  80%      1
  90%      1
  95%      1
 100%     79 (longest request)
```

#### (Extreme) Total 443,969 locations in the system, search 10 km bound to return 490 records.

Close to 1000 TPS for extremely large dataset and high HTTP overhead/traffic.
(When we limit HTTP server to only send the first 250 records, the TPS jumps to around 900)

```
Concurrency Level:      10
Time taken for tests:   10.000 seconds
Complete requests:      9728
Failed requests:        0
Total transferred:      375577600 bytes
HTML transferred:       374633790 bytes
Requests per second:    972.75 [#/sec] (mean)
Time per request:       10.280 [ms] (mean)
Time per request:       1.028 [ms] (mean, across all concurrent requests)
Transfer rate:          36675.73 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       0
Processing:     2   10  12.7      6     106
Waiting:        2    9  12.0      5     105
Total:          2   10  12.7      6     106

Percentage of the requests served within a certain time (ms)
  50%      6
  66%      9
  75%     11
  80%     12
  90%     15
  95%     23
  98%     66
  99%     81
 100%    106 (longest request)
```

## Limitations
* Search bounds are approximate square, and become gradually curved as the area increases.
* [Latitude/Longitude approximation:](http://stackoverflow.com/questions/1253499/simple-calculations-for-working-with-lat-lon-km-distance)
  * 1 deg latitude = 110.574 km
  * 1 deg longitude = 111.320*cos(latitude) km
* This geohash based proximity search does not search across the boundary at the poles (0) and at the international date line (180/-180).
