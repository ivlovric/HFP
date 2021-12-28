package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const AppVersion = "0.5"

var localAddr *string = flag.String("l", ":9060", "Local HEP listening address")
var remoteAddr *string = flag.String("r", "192.168.2.2:9060", "Remote HEP address")
var IPfilter *string = flag.String("ipf", "", "IP filter address from HEP SRC or DST chunks. Option can use multiple IP as comma sepeated values. Default is no filter without processing HEP acting as high performance HEP proxy")
var IPfilterAction *string = flag.String("ipfa", "pass", "IP filter Action. Options are pass or reject")
var Debug *string = flag.String("d", "off", "Debug options are off or on")
var PrometheusPort *string = flag.String("prom", "8090", "Prometheus metrics port")

var (
	AppLogger   *log.Logger
	filterIPs   []string
	HFPlog      string = "HFP.log"
	HEPsavefile string = "HEP/HEP-saved.arch"
)

func copyHEPbufftoFile(inbytes []byte, file string) (int64, error) {

	destination, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Open HEP file error", err)
	}

	defer destination.Close()
	nBytes, err := destination.Write(inbytes)

	if err != nil {
		log.Println("||-->XFile Send HEP from buffer to file error", err)
	} else {
		log.Println("||-->VFile Send HEP from buffer to file OK")
		go hepBytesInFile.Add(float64(nBytes))

	}

	return int64(nBytes), err

}

func copyHEPFileOut(file string, outnet net.Conn) (int, error) {

	HEPFileData, HEPFileDataerr := ioutil.ReadFile(HEPsavefile)
	if HEPFileDataerr != nil {
		fmt.Println("Read HEP file error", HEPFileDataerr)
	}

	rConn, err := net.DialTimeout("tcp4", *remoteAddr, 5*time.Second)

	//Send Logged HEP upon reconnect out to backend
	hl, err := rConn.Write(HEPFileData)
	if err != nil {
		log.Println("||-->X Send HEP from LOG error", err)
		AppLogger.Println("||-->X Send HEP from LOG error", err)
		hepFileFlushesError.Inc()
	} else {

		fi, err := os.Stat(HEPsavefile)
		if err != nil {
			log.Println("Cannot stat HEP log file", err)
			AppLogger.Println("Cannot stat HEP log file", err)
		}

		if fi.Size() > 0 {
			log.Println("||-->V Send HEP from LOG OK -", hl, "bytes")
			log.Println("Clearing HEP file")
			AppLogger.Println("||-->V Send HEP from LOG OK -", hl, "bytes")
			AppLogger.Println("Clearing HEP file")
			//Recreate file, thus cleaning the content
			os.Create(HEPsavefile)
			hepFileFlushesSuccess.Inc()
		}
	}

	defer rConn.Close()
	//	nBytes, err := io.Copy(destination, innet)

	return hl, err
}

func proxyConn(conn *net.TCPConn) {

	// Create buffer per "HEP3 Network Protocol Specification rev. 32"
	// The HEP3 header consists of a 4-octet protocol identifier with the fixed value 0x48455033
	// (ASCII „HEP3“) and a two-octet length value (network byte order). The length value specifies
	// the total packet length including the HEP3 or EEP3 ID, and the length field itself and the
	// payload. It has a possible range of values between 6 and 65535

	buf := make([]byte, 65535)

	//Connect out to backend with strict timeout
	rConn, err := net.DialTimeout("tcp4", *remoteAddr, 5*time.Second)

	if err != nil {
		log.Println("||-->X Dial OUT error", err)
		AppLogger.Println("|| -->X Dial OUT error", err)
		connectionStatus.Set(0)
		data, err_inconn := conn.Read(buf)
		if err_inconn != nil {
			log.Println("-->X||Read IN packets error:", err_inconn)
			AppLogger.Println("-->X||Read IN packets error:", err_inconn)
			if err_inconn != io.EOF {
				log.Println("-->X||Read IN packets error:", err_inconn)
				AppLogger.Println("-->X||Read IN packets error:", err_inconn)
				connectionStatus.Set(0)
			}
			return
		}

		connectionStatus.Set(1)
		hepPkt, err := DecodeHEP(buf[:data])
		if err != nil {
			log.Println("Error decoding HEP", err)
		}

		if *Debug == "on" {
			//log.Println("HEP decoded ", hepPkt)
			log.Println("HEP decoded SRC IP", hepPkt.SrcIP)
			log.Println("HEP decoded DST IP", hepPkt.DstIP)
		}

		for _, ipf := range filterIPs {
			if ((hepPkt.SrcIP == string(ipf) || hepPkt.DstIP == string(ipf)) && *IPfilter != "" && *IPfilterAction == "pass") || (*IPfilter == "" || (hepPkt.SrcIP != string(ipf) || hepPkt.DstIP != string(ipf)) && *IPfilter != "" && *IPfilterAction == "reject") {

				go copyHEPbufftoFile(buf[:data], HEPsavefile)
			}
		}

		log.Printf("-->|| Receiving HEP to LOG")
		AppLogger.Println("-->|| Receiving HEP to LOG")
		connectionStatus.Set(0)

		//Connection retries
		for range time.Tick(time.Second * 10) {
			conn, err_outreconn := net.DialTimeout("tcp4", *remoteAddr, 5*time.Second)
			if err_outreconn == nil {
				log.Println("||-->V Dial OUT reconnected", err_outreconn)

				break
			}
			log.Println("||-->X Dial OUT reconnect failure - retrying", conn)
			AppLogger.Println("||-->X Dial OUT reconnect failure - retrying")
		}
		return
	} else {
		log.Println("||--> Connected OUT", rConn.RemoteAddr())
		AppLogger.Println("||--> Connected OUT", rConn.RemoteAddr())
		connectionStatus.Set(1)
		copyHEPFileOut(HEPsavefile, rConn)
	}

	defer rConn.Close()

	reader := bufio.NewReader(conn)

	for {
		//Read incoming packets
		data, err_inconn := reader.Read(buf)
		if err_inconn != nil {
			log.Println("-->X||Read IN packets error:", err_inconn)
			if err_inconn != io.EOF {
				log.Println("-->X||Read IN packets error:", err_inconn)
			}
			return
		}

		if *Debug == "on" {
			log.Println("-->|| Got", data, "bytes on wire -- Total buffer size:", len(buf))
		}

		//Prometheus timestamp metric of incoming packet to detect lack of inbound HEP traffic
		clientLastMetricTimestamp.SetToCurrentTime()

		if *IPfilter != "" && *IPfilterAction == "pass" {
			hepPkt, err := DecodeHEP(buf[:data])
			if err != nil {
				log.Println("Error decoding HEP", err)
			}

			if *Debug == "on" {
				//log.Println("HEP decoded ", hepPkt)
				log.Println("HEP decoded SRC IP", hepPkt.SrcIP)
				log.Println("HEP decoded DST IP", hepPkt.DstIP)
			}

			var accepted bool = false
			for _, ipf := range filterIPs {
				if hepPkt.SrcIP == string(ipf) || hepPkt.DstIP == string(ipf) {

					//Send HEP out to backend
					if _, err_HEPout := fmt.Fprint(rConn, string(buf[:data])); err_HEPout != nil {
						log.Println("||--> Sending HEP OUT error:", err_HEPout)
						//	rb := bytes.NewReader(buf[:data])
						go copyHEPbufftoFile(buf[:data], HEPsavefile)
						accepted = true
						return
					} else {
						if *Debug == "on" {
							log.Println("||--> Sending HEP OUT successful with filter for", string(ipf), "to", rConn.RemoteAddr())
						}
						accepted = true

					}
				}
			}

			if accepted == false {
				if *Debug == "on" {
					log.Println("-->X|| HEP filter not matched with source or destination IP in HEP packet", hepPkt.SrcIP, "or", hepPkt.DstIP)
				}
			}

		} else if *IPfilter != "" && *IPfilterAction == "reject" {
			hepPkt, err := DecodeHEP(buf[:data])
			if err != nil {
				log.Println("Error decoding HEP", err)
			}

			if *Debug == "on" {
				//log.Println("HEP decoded ", hepPkt)
				log.Println("HEP decoded SRC IP", hepPkt.SrcIP)
				log.Println("HEP decoded DST IP", hepPkt.DstIP)
			}

			var rejected bool = false
			for _, ipf := range filterIPs {
				if hepPkt.SrcIP == string(ipf) || hepPkt.DstIP == string(ipf) {
					conn.Write([]byte("Rejecting IP"))
					if *Debug == "on" {
						log.Printf("-->X|| Rejecting IP:%q", ipf)
					}
					rejected = true
					break
				}
			}

			if rejected == false {
				//Send HEP out to backend
				if _, err_HEPout := fmt.Fprint(rConn, string(buf[:data])); err_HEPout != nil {
					log.Println("||--> Sending HEP OUT error:", err_HEPout)
					//rb := bytes.NewReader(buf[:data])
					go copyHEPbufftoFile(buf[:data], HEPsavefile)
					return
				} else {
					if *Debug == "on" {
						log.Println("||--> Sending HEP OUT successful with filter to", rConn.RemoteAddr())
					}
				}
			}

		} else {
			//Send HEP out to backend
			if _, err_HEPout := fmt.Fprint(rConn, string(buf[:data])); err_HEPout != nil {
				log.Println("||--> Sending HEP OUT error:", err_HEPout)
				// rb := bytes.NewReader(buf[:data])
				go copyHEPbufftoFile(buf[:data], HEPsavefile)
				return
			} else {
				if *Debug == "on" {
					log.Println("||--> Sending HEP OUT successful without filters to", rConn.RemoteAddr())
				}
			}
		}
	}

	//Incoming data from backend side
	data := make([]byte, 1024*8)
	n, err := rConn.Read(data)
	if err != nil {
		if err != io.EOF {
			log.Println("||<-- Received:", err)
			AppLogger.Println("||<-- Received:", err)
			return
		} else {
			if *Debug == "on" {
				log.Println("||<-- Received:", err, data[:n])
				AppLogger.Println("||<-- Received:", err, data[:n])
			}
		}
	}
}

func handleConn(in <-chan *net.TCPConn, out chan<- *net.TCPConn) {
	for conn := range in {
		proxyConn(conn)
		out <- conn
	}
}

func closeConn(in <-chan *net.TCPConn) {
	for conn := range in {
		conn.Close()
	}
}

func main() {

	var wg sync.WaitGroup

	version := flag.Bool("v", false, "Prints current HFP version")
	flag.Parse()

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	filterIPs = strings.Split(*IPfilter, ",")

	errmkdir := os.Mkdir("HEP", 0755)
	if errmkdir != nil {
		log.Println(errmkdir)
	}

	if _, errhfexist := os.Stat(HEPsavefile); errhfexist != nil {
		if os.IsNotExist(errhfexist) {
			fmt.Println("HEP File doesnt exists - Creating", errhfexist)
			_, errhfcreate := os.Create(HEPsavefile)
			fmt.Println("-->|| Creating HEP file")
			if errhfcreate != nil {
				fmt.Println("Create file error", errhfcreate)
				return
			}
		}
	}

	applog, err := os.OpenFile(HFPlog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	AppLogger = log.New(applog, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	fi, err := os.Stat(HEPsavefile)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("Saved HEP file is %d bytes long\n", fi.Size())

	fmt.Printf("Listening for HEP on: %v\nProxying HEP to: %v\nIPFilter: %v\nIPFilterAction: %v\nPrometheus metrics: %v\n\n", *localAddr, *remoteAddr, *IPfilter, *IPfilterAction, *PrometheusPort)
	AppLogger.Println("Listening for HEP on:", *localAddr, "\n", "Proxying HEP to:", *remoteAddr, "\n", "IPFilter:", *IPfilter, "\n", "IPFilterAction:", *IPfilterAction, "\n", "Prometheus metrics:", *PrometheusPort, "\n")

	if *IPfilter == "" {
		fmt.Printf("HFP started in proxy high performance mode\n__________________________________________\n")
		AppLogger.Println("HFP started in proxy high performance mode\n__________________________________________\n")
	} else {
		fmt.Printf("HFP started in proxy processing mode\n_____________________________________\n")
		AppLogger.Println("HFP started in proxy processing mode\n_____________________________________\n")
	}

	addr, err := net.ResolveTCPAddr("tcp", *localAddr)
	if err != nil {
		log.Println(err)
		return
	}

	listener, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		fmt.Println("X|| Server starting error", err)
		os.Exit(1)
	}

	pending, complete := make(chan *net.TCPConn), make(chan *net.TCPConn)

	for i := 1; i <= 4; i++ {
		go handleConn(pending, complete)
	}
	go startMetrics(&wg)
	wg.Wait()
	go closeConn(complete)

	for {
		conn, err := listener.AcceptTCP()
		log.Println("-->|| New connection from", conn.RemoteAddr())
		AppLogger.Println("-->|| New connection from", conn.RemoteAddr())
		connectedClients.Inc()

		if err != nil {
			log.Println(err)
			return
		}
		pending <- conn
	}
}
