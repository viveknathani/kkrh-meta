package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var (
	errorRate                    float64
	averageRequestsPerDay        float64
	averageProcessingTime        float64
	endPointDistribution         map[string]float64
	averageTimeSpentPerComponent map[string]float64
)

func computeErrorRate(requests map[string][]interface{}) float64 {

	var result float64 = 0
	errorCount := 0
	totalCount := len(requests)

	for _, arr := range requests {
		for _, store := range arr {
			temp := store.(map[string]interface{})
			if temp != nil && temp["level"] == "ERROR" {
				errorCount++
			}
		}
	}

	result = float64(errorCount) / float64(totalCount)
	result = result * 100.0
	return result
}

func computeAvgProcessingTime(requests map[string][]interface{}) float64 {

	var result float64 = 0
	for _, arr := range requests {

		var startTime float64 = math.MaxFloat64
		var endTime float64 = 0
		for _, store := range arr {
			temp := store.(map[string]interface{})
			if temp == nil {
				continue
			}
			ts := temp["ts"]
			if ts != nil {
				tss := ts.(float64)

				res1 := big.NewFloat(startTime).Cmp(big.NewFloat(tss))
				res2 := big.NewFloat(endTime).Cmp(big.NewFloat(tss))
				if res1 == 1 {
					startTime = tss
				}
				if res2 == -1 {
					endTime = tss
				}
			}
		}
		responseTime := endTime - startTime
		result += (responseTime / float64(len(requests)))
	}

	return result
}

func computeAvgRequestsPerDay(requests map[string][]interface{}) float64 {

	daily := make(map[string]int)
	for _, arr := range requests {

		var earliest float64 = math.MaxFloat64
		for _, store := range arr {
			temp := store.(map[string]interface{})
			if temp == nil {
				continue
			}
			ts := temp["ts"]
			if ts != nil {
				tss := ts.(float64)
				res := big.NewFloat(earliest).Cmp(big.NewFloat(tss))
				if res == 1 {
					earliest = tss
				}
			}
		}

		y, m, d := time.Unix(int64(earliest/1000), 0).Date()
		date := formADate(y, int(m), d)
		daily[date]++
	}

	var result float64 = 0
	for k, v := range daily {
		fmt.Println(k, v)
		result += (float64(v) / float64(len(daily)))
	}

	return result
}

func computeEndpointDistribution(requests map[string][]interface{}) map[string]int {

	endpoint := make(map[string]int)

	for _, arr := range requests {
		for _, store := range arr {
			temp := store.(map[string]interface{})
			if temp == nil {
				continue
			}
			path := temp["path"]
			if path != nil {
				endpoint[path.(string)]++
				break
			}
		}
	}

	return endpoint
}

func formADate(year int, month int, day int) string {

	res := ""
	if year < 10 {
		res += "0"
	}
	res += strconv.Itoa(year)

	if month < 10 {
		res += "0"
	}
	res += strconv.Itoa(month)

	if day < 10 {
		res += "0"
	}
	res += strconv.Itoa(day)
	return res
}

func computeTimeSpentPerComponent(requests map[string][]interface{}) (float64, float64, float64) {

	var one float64 = 0
	var two float64 = 0
	var three float64 = 0
	for _, arr := range requests {

		var databaseStartTime float64 = math.MaxFloat64
		var databaseStopTime float64 = 0
		var cacheStartTime float64 = math.MaxFloat64
		var cacheStopTime float64 = 0
		var appStartTime float64 = math.MaxFloat64
		var appStopTime float64 = 0

		for _, store := range arr {
			temp := store.(map[string]interface{})
			if temp == nil {
				continue
			}

			msg := temp["message"].(string)
			ts := temp["ts"]
			if ts != nil {
				tss := ts.(float64)

				if strings.HasPrefix(msg, "cache:") {
					res1 := big.NewFloat(cacheStartTime).Cmp(big.NewFloat(tss))
					res2 := big.NewFloat(cacheStopTime).Cmp(big.NewFloat(tss))
					if res1 == 1 {
						cacheStartTime = tss
					}
					if res2 == -1 {
						cacheStopTime = tss
					}
				} else if strings.HasPrefix(msg, "database:") {
					res1 := big.NewFloat(databaseStartTime).Cmp(big.NewFloat(tss))
					res2 := big.NewFloat(databaseStopTime).Cmp(big.NewFloat(tss))
					if res1 == 1 {
						databaseStartTime = tss
					}
					if res2 == -1 {
						databaseStopTime = tss
					}
				} else {
					res1 := big.NewFloat(appStartTime).Cmp(big.NewFloat(tss))
					res2 := big.NewFloat(appStopTime).Cmp(big.NewFloat(tss))
					if res1 == 1 {
						appStartTime = tss
					}
					if res2 == -1 {
						appStopTime = tss
					}
				}
			}
		}

		app := appStopTime - appStartTime
		db := databaseStopTime - databaseStartTime
		cache := cacheStopTime - cacheStartTime

		if app >= 0 {
			one += app / float64(len(requests))
		}
		if db >= 0 {
			two += db / float64(len(requests))
		}
		if cache >= 0 {
			three += cache / float64(len(requests))
		}
	}

	return one, two, three
}

func main() {

	content, err := ioutil.ReadFile("./data/data.json")
	if err != nil {
		log.Fatal(err)
	}

	var f map[string]interface{}
	err = json.Unmarshal(content, &f)
	if err != nil {
		log.Fatal(err)
		return
	}

	store := make(map[string][]interface{})
	for _, v := range f["kkrh"].(map[string]interface{}) {
		temp := v.(map[string]interface{})

		t := temp["requestID"]
		if t != nil {
			store[t.(string)] = append(store[t.(string)], temp)
		}
	}

	fmt.Println(computeErrorRate(store))
	fmt.Println(computeAvgProcessingTime(store))
	fmt.Println(computeAvgRequestsPerDay(store))
	fmt.Println(computeEndpointDistribution(store))
	fmt.Println(computeTimeSpentPerComponent(store))
}
