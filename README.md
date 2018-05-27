# thrash
Golang HTTP Micro Benchmarker

## Usage

```
Usage: ./thrash [flags] url
  -c int
    	how much concurrency (default 1)
  -e	print errors
  -h	print response time histogram
  -n int
    	how many requests (default 100)
  -p	start the profile server on port 6060
  -t duration
    	request timeout in MS (default 1m0s)
```

## Example and Output

```sh
thrash -c 10 -h https://fakedomainzthatdonotexist.com/ping
Thrashing https://fakedomainzthatdonotexist.com/ping
Concurrency 10 Num Requests 100
100 / 100 [-------------------------------------------------------------------------] 100.00% 71 p/s
Responses OK: 100% (100/100), Errors: 0
Status Codes: {"200":100}
Bytes Transferred: 1,300
Avg Response Time: 156.734008ms
Min Response Time 87.21594ms
Max Response Time 653.075597ms
( 81%) ∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎ [87.21594ms - 228.680854ms]
(  9%) ∎∎∎∎∎ [228.680854ms - 370.145768ms]
(  7%) ∎∎∎∎ [370.145768ms - 511.610682ms]
(  2%) ∎[511.610682ms - 653.075596ms]
(  1%) ∎ [653.075596ms - 794.54051ms]
```
