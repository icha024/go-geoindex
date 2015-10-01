cmake_minimum_required (VERSION 2.6) 
project(GEOHASH)

set(CMAKE_C_FLAGS, "${CMAKE_C_FLAGS} -O2 -g -Wall -pedantic -std=c99")

# Create a geohash_test file including main
add_executable(geohash_test geohash_main.c geohash_helper.c)

# Build a shared and static library
add_library(geohash SHARED geohash.c)
add_library(geohash_static STATIC geohash.c)

# Tell CMake we want a .so and not .dylib
set_target_properties(geohash PROPERTIES SUFFIX .so) 

# Link libgeohash.so to our main function
target_link_libraries(geohash_test geohash)
