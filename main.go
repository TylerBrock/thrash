package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const DEFAULT_NUM_REQUESTS = 100
const DEFAULT_CONCURRENCY = 1

type Response struct {
	OK            bool
	Error         error
	StartTime     time.Time
	EndTime       time.Time
	Status        string
	StatusCode    int
	ContentLength int64
	Body          []byte
}

func printHistrogram(times []time.Duration, maxTime time.Duration, minTime time.Duration) {
	scalingFactor := float64(100) / float64(len(times))
	var buckets [5]int64
	bucketLength := float64(maxTime-minTime) / 4
	for _, responseTime := range times {
		bucketTime := time.Duration(responseTime) - minTime
		bucket := int(float64(bucketTime) / bucketLength)
		buckets[bucket]++
	}
	for index, bucket := range buckets {
		bucketStart := minTime + (time.Duration(bucketLength) * time.Duration(index))
		bucketEnd := minTime + (time.Duration(bucketLength) * time.Duration(index+1))
		fmt.Printf("(%3d%%) ", int(float64(bucket)*scalingFactor))
		for i := 0; i < int(float64(bucket)*scalingFactor); i += 2 {
			fmt.Print("∎")
			if i == int(bucket)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Printf("[%v - %v]", bucketStart, bucketEnd)
		fmt.Println()
	}
}

func fetchURL(ack chan<- Response, url string) {
	response := Response{OK: true, StartTime: time.Now()}
	resp, err := http.Get(url)
	response.EndTime = time.Now()

	if err != nil {
		response.OK = false
		response.Error = err
		ack <- response
		return
	}

	response.Status = resp.Status
	response.StatusCode = resp.StatusCode
	response.ContentLength = resp.ContentLength

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		response.Body = body
	} else {
		fmt.Println("Error reading response body", err)
	}

	ack <- response
}

func main() {
	url := os.Args[len(os.Args)-1]
	fmt.Println("Thrashing", url)
	var concurrency int
	var numRequests int
	var histogram bool
	flag.IntVar(&concurrency, "c", DEFAULT_CONCURRENCY, "how much concurrency")
	flag.IntVar(&numRequests, "n", DEFAULT_NUM_REQUESTS, "how many requests")
	flag.BoolVar(&histogram, "h", false, "print response time histogram")
	flag.Parse()
	fmt.Println("Concurrency", concurrency, "Num Requests", numRequests)

	sem := make(chan bool, concurrency)
	ack := make(chan Response, numRequests)

	// Queue up the requests
	for i := 0; i < numRequests; i++ {
		sem <- true
		go func() {
			defer func() { <-sem }()
			fetchURL(ack, url)
		}()
	}

	var numOK int
	var bytesTransferred int64
	var responseTimes = make([]time.Duration, numRequests)

	// Collect the responses
	for i := 0; i < numRequests; i++ {
		response := <-ack
		if response.OK {
			numOK++
			if response.ContentLength != -1 {
				bytesTransferred += response.ContentLength
			}
			responseTime := response.EndTime.Sub(response.StartTime)
			responseTimes[i] = responseTime
		} else {
			fmt.Println("Error:", response.Error)
		}
	}

	var sumResponseTimes time.Duration
	maxResponseTime := responseTimes[0]
	minResponseTime := responseTimes[0]

	for i := 0; i < numRequests; i++ {
		responseTime := responseTimes[i]
		sumResponseTimes += responseTime

		if responseTime > maxResponseTime {
			maxResponseTime = responseTime
		}
		if responseTime < minResponseTime {
			minResponseTime = responseTime
		}
	}

	pctOK := int((float64(numOK) / float64(numRequests)) * 100)
	avgResponseTime := time.Duration(float64(sumResponseTimes) / float64(numOK))
	fmt.Printf("Responses OK: %d%%\n", pctOK)
	fmt.Printf("Bytes Transferred: %d\n", bytesTransferred)
	fmt.Printf("Avg Response Time: %v\n", avgResponseTime)
	fmt.Printf("Min Response Time %v\n", minResponseTime)
	fmt.Printf("Max Response Time %v\n", maxResponseTime)
	if histogram {
		printHistrogram(responseTimes, maxResponseTime, minResponseTime)
	}
}
