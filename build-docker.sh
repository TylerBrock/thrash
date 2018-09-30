CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o thrash .
docker build -t tbrock/thrash:latest .

