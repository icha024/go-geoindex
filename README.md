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

Around 12,000 TPS for norminal conditions.

```
Concurrency Level:      10
Time taken for tests:   4.194 seconds
Complete requests:      50000
Failed requests:        0
Total transferred:      364050000 bytes
HTML transferred:       359200000 bytes
Requests per second:    11923.10 [#/sec] (mean)
Time per request:       0.839 [ms] (mean)
Time per request:       0.084 [ms] (mean, across all concurrent requests)
Transfer rate:          84777.40 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       2
Processing:     0    1   1.6      1      80
Waiting:        0    1   1.6      0      79
Total:          0    1   1.6      1      80

Percentage of the requests served within a certain time (ms)
  50%      1
  66%      1
  75%      1
  80%      1
  90%      1
  95%      1
  98%      2
  99%      3
 100%     80 (longest request)

```

#### (Extreme) Total 443,969 locations in the system, search 10 km bound to return 490 records.

Around 600 TPS for extremely large dataset and high HTTP overhead/traffic.
(When we limit HTTP server to only send the first 250 records, the TPS jumps to around 900)

```
Concurrency Level:      10
Time taken for tests:   10.000 seconds
Complete requests:      5936
Failed requests:        0
Total transferred:      873496000 bytes
HTML transferred:       872919917 bytes
Requests per second:    593.59 [#/sec] (mean)
Time per request:       16.847 [ms] (mean)
Time per request:       1.685 [ms] (mean, across all concurrent requests)
Transfer rate:          85301.50 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       0
Processing:     4   17  15.2     11     119
Waiting:        2   11  13.0      6     110
Total:          4   17  15.2     11     119

Percentage of the requests served within a certain time (ms)
  50%     11
  66%     17
  75%     19
  80%     21
  90%     27
  95%     41
  98%     79
  99%     93
 100%    119 (longest request)

```

## Limitations
* Search bounds are approximate square, and become gradually curved as the area increases.
* [Latitude/Longitude approximation:](http://stackoverflow.com/questions/1253499/simple-calculations-for-working-with-lat-lon-km-distance)
  * 1 deg latitude = 110.574 km
  * 1 deg longitude = 111.320*cos(latitude) km
* This geohash based proximity search does not search across the boundary at the poles (0) and at the international date line (180/-180).
