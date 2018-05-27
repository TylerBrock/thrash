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
$ thrash -c 10 -h https://fakedomainzthatdonotexist.com/ping
Thrashing https://fakedomainzthatdonotexist.com/ping
Concurrency 10 Num Requests 100
100 / 100 [--------------------------------------------------------------------------------------] 100.00% 60 p/s
Responses OK: 100% (100/100), Errors: 0
Status Codes: {"200":100}
Bytes Transferred: 1,300
Avg Response Time: 152.603877ms
Min Response Time 86.801591ms
Max Response Time 531.204362ms
( 83%) ∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎ [86.801591ms - 197.902283ms]
(  1%) [197.902283ms - 309.002975ms]
(  8%) ∎∎∎∎ [309.002975ms - 420.103667ms]
(  7%) ∎∎∎ [420.103667ms - 531.204359ms]
(  1%) [531.204359ms - 642.305051ms]
```
