# trotter-api

This repo will act as the backend for trotter. All api will be maintained here
<br/>
<br/>

# Prereqs:
1. Set the environment variables `PORT_TROTTER` to what ever port desired.  We typically use 3002.
  - This link https://www.computerhope.com/issues/ch000549.htm#windows10 shows you how to do it in windows

# To run:

1. `go run main.go`

API we are using to get Country Info

# Triposo - for city info, tour info

https://www.triposo.com/api/console/20180627
login: asqwrd@gmail.com
pw: trotter@world

# Sherpa Api - for Visa, immunization for countries

http://apidocsv2.joinsherpa.com/
API access:
{
"key": "nIXaxALFPV0IiwNOvBEBrDCNSw3SCv67R4UEvD9r",
"username": "VDLQLCbMmugvsOEtihQ9kfc6nQoeGd"
}

# Docker
`docker run -d \
  --name watchtower \
  -e REPO_USER=username \
  -e REPO_PASS=password \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower $(docker ps -a -q --filter ancestor=asqwrd/trotter-api --format="{{.ID}}") --debug`
  
`docker run -p 3002:3002 -dit --restart unless-stopped asqwrd/trotter-api`
