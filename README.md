# HFP
#### HEP Fidelity Proxy

Reliable way of relaying all your [HEP](http://hep.sipcapture.org) to any HEP remote server that is behind unreliable networks.

It is buffered TCP proxy with option of storing HEP locally in cases of backend HEP server unavailability and replaying of that HEP after HEP server becomes reachable again. It can be beneficial in highly distributed voice networks to reliably deliver your HEP to its destination without additional infrastructure.
It can be deployed locally to every HEP generating node within one premesis/DC/location acting as addon (1:1) approach or HEP generating nodes can connect to one HFP that will   reliably proxy HEP generated within one premesis/DC/location (N:1)

From version 0.2 two modes of operation are supported.
- Strict reliable HEP proxy mode without processing for high-performance
- Reliable processing HEP proxy mode for filtering purposes to filter HEP by IP whether it is from HEP source or destination fields. It is engaged when "ipf" switch is configured


### Usage
```
./HFP -l :9060 -r (HEP TCP server we want to reliably proxy HEP)

Options:
  -l string
    	Local HEP listening address (default ":9060")
  -r string
    	Remote HEP address (default "192.168.2.2:9060")
  -ipf string
    	IP filter address from HEP SRC or DST chunks. Option can use multiple IP as comma sepeated values. Default is no filter without processing HEP acting as high performance HEP proxy
  -ipfa string
    	IP filter Action. Options are pass or reject (default "pass")
  -d string
    	Debug options are off or on (default "off")
  -prom string
    	Prometheus metrics port (default "8090")
```

### Build
##### Manual
Building HFP requires go 1.15+
```
make
```
###### Docker
```
docker build -t sipcapture/HFP .
docker run -ti --rm sipcapture/HFP -l :9062 -r 1.2.3.4:9062 (optional: -ipf <comma separated IP addresses> -ipfa <action for "ipf" list> -d <on> -prom <Prometheus port>)
```

### Flow Diagrams


<img width="794" alt="image" src="https://user-images.githubusercontent.com/37185376/127317842-3e65c362-8cc3-4666-9cd2-6495a5122a62.png">


### Metrics
Metrics are accessable on port 8090 unless port is changed by option flag. Example: http://HFP:8090/metrics

Prometheus metrics in grafana

<img width="777" alt="Screenshot 2021-09-08 at 12 42 45" src="https://user-images.githubusercontent.com/37185376/132495818-358ea147-f4ac-4c23-92a0-5e8e6cff336c.png">
<img width="777" alt="Screenshot 2021-09-08 at 12 44 29" src="https://user-images.githubusercontent.com/37185376/132495959-f48bc102-0bcb-4863-bcf0-52c129ad831d.png">



### Note

HEP parser and decoder used from https://github.com/sipcapture/heplify-server Heplify-server project
