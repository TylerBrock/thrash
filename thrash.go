package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"golang.org/x/text/message"
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

func fetchURL(ack chan<- *Response, url string, client *http.Client) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating new request")
	}

	response := &Response{OK: true, StartTime: time.Now()}
	resp, err := client.Do(req)
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
	_, err = io.Copy(ioutil.Discard, resp.Body)

	if err != nil {
		response.OK = false
		response.Error = err
		fmt.Println("Error reading response body", err)
	}

	ack <- response
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	url := os.Args[len(os.Args)-1]
	fmt.Println("Thrashing", url)
	var concurrency int
	var numRequests int
	var histogram bool
	flag.IntVar(&concurrency, "c", DEFAULT_CONCURRENCY, "how much concurrency")
	flag.IntVar(&numRequests, "n", DEFAULT_NUM_REQUESTS, "how many requests")
	flag.BoolVar(&histogram, "h", false, "print response time histogram")
	flag.Parse()

	p := message.NewPrinter(message.MatchLanguage("en"))
	p.Println("Concurrency", concurrency, "Num Requests", numRequests)

	sem := make(chan bool, concurrency)
	ack := make(chan *Response, numRequests)

	numClients := concurrency

	clients := make([]*http.Client, numClients)
	tr := &http.Transport{
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 1000,
	}
	for i := 0; i < numClients; i++ {
		clients[i] = &http.Client{Transport: tr}
	}

	bar := pb.StartNew(numRequests)

	// Queue up the requests
	for i := 0; i < numRequests; i++ {
		sem <- true
		go func() {
			defer func() { <-sem }()
			clientNum := i % len(clients)
			fetchURL(ack, url, clients[clientNum])
			bar.Increment()
		}()
	}

	bar.Finish()

	var numOK int
	var bytesTransferred int64
	var responseTimes = make([]time.Duration, numRequests)
	statusCounts := map[int]int{}

	// Collect the responses
	for i := 0; i < numRequests; i++ {
		response := <-ack
		if response.OK {
			numOK++
			statusCounts[response.StatusCode]++
			if response.ContentLength != -1 {
				bytesTransferred += response.ContentLength
			}
			responseTime := response.EndTime.Sub(response.StartTime)
			responseTimes[i] = responseTime
		} else {
			p.Println("Error:", response.Error)
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
	statusCountsString, _ := json.Marshal(statusCounts)

	pctOK := int((float64(numOK) / float64(numRequests)) * 100)
	avgResponseTime := time.Duration(float64(sumResponseTimes) / float64(numOK))
	p.Printf("Responses OK: %d%% (%d/%d)\n", pctOK, numOK, numRequests)
	p.Printf("Status Codes: %s\n", statusCountsString)
	p.Printf("Bytes Transferred: %d\n", bytesTransferred)
	p.Printf("Avg Response Time: %v\n", avgResponseTime)
	p.Printf("Min Response Time %v\n", minResponseTime)
	p.Printf("Max Response Time %v\n", maxResponseTime)
	if histogram {
		printHistrogram(responseTimes, maxResponseTime, minResponseTime)
	}
}
