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
	"net/url"
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"golang.org/x/text/message"
)

const DEFAULT_NUM_REQUESTS = 100
const DEFAULT_CONCURRENCY = 1
const DEFAULT_TIMEOUT = "60s"

type Configuration struct {
	Concurrency int
	NumRequests int
	Timeout     time.Duration
	Histogram   bool
	PrintErrors bool
	Profile     bool
	Url         string
}

type Response struct {
	OK            bool
	Error         error
	StartTime     time.Time
	EndTime       time.Time
	Status        string
	StatusCode    int
	ContentLength int64
}

type ResponseSummary struct {
	NumResponses     int
	NumOK            int
	BytesTransferred int64
	SumResponseTimes time.Duration
	MaxResponseTime  time.Duration
	MinResponseTime  time.Duration
	ResponseTimes    []time.Duration
	StatusCounts     map[int]int
	Errors           []error
}

func (s *ResponseSummary) addResponse(r *Response) {
	s.NumResponses++

	if r.OK == false {
		s.Errors = append(s.Errors, r.Error)
		return
	}

	s.NumOK++

	if s.StatusCounts == nil {
		s.StatusCounts = map[int]int{}
	}
	s.StatusCounts[r.StatusCode]++

	if r.ContentLength != -1 {
		s.BytesTransferred += r.ContentLength
	}

	responseTime := r.EndTime.Sub(r.StartTime)
	s.ResponseTimes = append(s.ResponseTimes, responseTime)

	s.SumResponseTimes += responseTime

	if s.MaxResponseTime == 0 {
		s.MaxResponseTime = responseTime
	}

	if s.MinResponseTime == 0 {
		s.MinResponseTime = responseTime
	}

	if responseTime > s.MaxResponseTime {
		s.MaxResponseTime = responseTime
	}

	if responseTime < s.MinResponseTime {
		s.MinResponseTime = responseTime
	}
}

func (s *ResponseSummary) print() {
	statusCountsString, _ := json.Marshal(s.StatusCounts)

	pctOK := int((float64(s.NumOK) / float64(s.NumResponses)) * 100)
	avgResponseTime := time.Duration(0)
	if s.NumOK != 0 {
		avgResponseTime = time.Duration(float64(s.SumResponseTimes) / float64(s.NumOK))
	}
	p := message.NewPrinter(message.MatchLanguage("en"))
	p.Printf("Responses OK: %d%% (%d/%d), Errors: %d\n", pctOK, s.NumOK, s.NumResponses, len(s.Errors))
	p.Printf("Status Codes: %s\n", statusCountsString)
	p.Printf("Bytes Transferred: %d\n", s.BytesTransferred)
	p.Printf("Avg Response Time: %v\n", avgResponseTime)
	p.Printf("Min Response Time %v\n", s.MinResponseTime)
	p.Printf("Max Response Time %v\n", s.MaxResponseTime)
}

func (s *ResponseSummary) printErrors() {
	for _, err := range s.Errors {
		fmt.Println(err)
	}
}

func (s *ResponseSummary) printHistogram() {
	scalingFactor := float64(100) / float64(len(s.ResponseTimes))
	var buckets [5]int64
	bucketLength := float64(s.MaxResponseTime-s.MinResponseTime) / 4
	for _, responseTime := range s.ResponseTimes {
		bucketTime := time.Duration(responseTime) - s.MinResponseTime
		bucket := int(float64(bucketTime) / bucketLength)
		buckets[bucket]++
	}
	for index, bucket := range buckets {
		bucketStart := s.MinResponseTime + (time.Duration(bucketLength) * time.Duration(index))
		bucketEnd := s.MinResponseTime + (time.Duration(bucketLength) * time.Duration(index+1))
		fmt.Printf("(%3d%%) ", int(float64(bucket)*scalingFactor))
		bricks := int(float64(bucket)*scalingFactor) / 2
		for i := 0; i < bricks; i += 1 {
			fmt.Print("∎")
			if i == bricks-1 {
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

func startProfiler() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

func configure() Configuration {
	config := Configuration{}
	defaultTimeoutDuration, _ := time.ParseDuration(DEFAULT_TIMEOUT)
	flag.IntVar(&config.Concurrency, "c", DEFAULT_CONCURRENCY, "how much concurrency")
	flag.IntVar(&config.NumRequests, "n", DEFAULT_NUM_REQUESTS, "how many requests")
	flag.DurationVar(&config.Timeout, "t", defaultTimeoutDuration, "request timeout in MS")
	flag.BoolVar(&config.Histogram, "h", false, "print response time histogram")
	flag.BoolVar(&config.PrintErrors, "e", false, "print errors")
	flag.BoolVar(&config.Profile, "p", false, "start the profile server on port 6060")
	flag.Parse()

	flag.Usage = func() {
		printUsage()
	}

	if len(os.Args) < 2 || os.Args[1] == "-help" {
		printUsage()
		os.Exit(1)
	}

	urlArg := os.Args[len(os.Args)-1]

	_, err := url.ParseRequestURI(urlArg)
	if err != nil {
		fmt.Printf("Error: \"%s\" does not look like a valid url!\n", urlArg)
		printUsage()
		os.Exit(2)
	} else {
		config.Url = urlArg
	}

	return config
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] url\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	config := configure()

	fmt.Println("Thrashing", config.Url)

	p := message.NewPrinter(message.MatchLanguage("en"))
	p.Println("Concurrency", config.Concurrency, "Num Requests", config.NumRequests)

	if config.Profile {
		startProfiler()
	}

	sem := make(chan bool, config.Concurrency)
	ack := make(chan *Response, config.NumRequests)

	tr := &http.Transport{
		MaxIdleConns:        config.Concurrency,
		MaxIdleConnsPerHost: config.Concurrency,
	}
	client := http.Client{Transport: tr, Timeout: config.Timeout}

	bar := pb.StartNew(config.NumRequests)

	// Queue up the requests
	for i := 0; i < config.NumRequests; i++ {
		sem <- true
		go func() {
			defer func() { <-sem }()
			fetchURL(ack, config.Url, &client)
			bar.Increment()
		}()
	}

	summary := ResponseSummary{}

	// Collect the responses
	for i := 0; i < config.NumRequests; i++ {
		response := <-ack
		summary.addResponse(response)
		if response.OK != true && config.PrintErrors {
			fmt.Println(response.Error)
		}
	}

	bar.Finish()

	summary.print()
	if config.Histogram {
		summary.printHistogram()
	}
}
