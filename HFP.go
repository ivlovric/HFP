package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

var localAddr *string = flag.String("l", ":9060", "local address")
var remoteAddr *string = flag.String("r", "192.168.2.2:9060", "remote address")

func proxyConn(conn *net.TCPConn) {

	outFile, err := os.OpenFile("HEP/HEP-saved.arch", os.O_RDWR, 0664)
	if err != nil {
		fmt.Println("Open HEP file error", err)
		return
	}

	//Make buffer
	buf := make([]byte, 1024*8)
	//Connect out to backend with strict timeout
	rConn, err := net.DialTimeout("tcp4", *remoteAddr, 5*time.Second)

	if err != nil {
		log.Println("||-->X Dial OUT error", err)
		defer outFile.Close()
		go func() {
			_, errcopyfile := io.Copy(outFile, conn)
			if errcopyfile != nil {
				fmt.Println("Copy to FILE error\n", errcopyfile)
				return
			}
		}()
		log.Printf("-->|| Receiving HEP to LOG")
		//Connection retries
		for range time.Tick(time.Second * 10) {
			conn, err_outreconn := net.DialTimeout("tcp4", *remoteAddr, 5*time.Second)
			if err_outreconn == nil {
				log.Println("||-->V Dial OUT reconnected", err_outreconn)
				break
			}
			log.Println("||-->X Dial OUT reconnect failure - retrying", conn)
		}
		return
	} else {
		log.Println("||--> Connected OUT", rConn.RemoteAddr())
		HEPFileData, HEPFileDataerr := ioutil.ReadFile("HEP/HEP-saved.arch")
		if HEPFileDataerr != nil {
			fmt.Println("Read HEP file error", err)
			return
		}

		//Send Logged HEP upon reconnect out to backend
		hl, err := rConn.Write(HEPFileData)
		if err != nil {
			log.Println("||-->X Send HEP from LOG error", err)
			return
		} else {
			log.Println("||-->V Send HEP from LOG OK -", hl, "bytes")
			log.Println("Clearing HEP file")
			//Recreate file, thus cleaning the content
			os.Create("HEP/HEP-saved.arch")
		}

	}
	defer rConn.Close()

	for {
		//Read incomming packets
		data, err_inconn := conn.Read(buf)
		if err_inconn != nil {
			log.Println("-->X||Read IN packets error:", err_inconn)
			if err_inconn != io.EOF {
				log.Println("-->X||Read IN packets error:", err_inconn)
			}
			break
		}
		//Proxy only HEP
		if string(buf[:4]) == "HEP3" || string(buf[:4]) == "HEP2" || string(buf[:4]) == "HEP" {
			log.Printf("-->|| Received HEP version:%q", string(buf[:4]))
			log.Printf("-->|| Received HEP packet:%q", string(buf[:data]))
			log.Println("-->|| Got", data, "bytes")
			log.Println("-->|| Total buffer size:", len(buf))

			//Send HEP out to backend
			if _, err_HEPout := fmt.Fprint(rConn, string(buf[:data])); err_HEPout != nil {
				log.Println("||--> Sending HEP OUT error:", err)
				rb := bytes.NewReader(buf[:data])
				_, err := io.Copy(outFile, rb)
				if err != nil {
					log.Println("||-->X Send HEP from LOG error", err)
					return
				}
				log.Println("-->|| Received HEP to LOG", buf[:data])
				break
			}

			log.Println("||--> Sending HEP OUT successful", rConn.RemoteAddr())

		} else {
			log.Printf("-->|| Not HEP - C'mon")
		}
	}

	//Incomming data from backend side
	data := make([]byte, 1024*8)
	n, err := rConn.Read(data)
	if err != nil {
		if err != io.EOF {
			log.Println("||<-- Received:", err)
			return
		} else {
			log.Println("||<-- Received:", err, data[:n])
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

	flag.Parse()

	errmkdir := os.Mkdir("HEP", 0755)
	if errmkdir != nil {
		log.Println(errmkdir)
	}

	if _, errhfexist := os.Stat("./HEP/HEP-saved.arch"); errhfexist != nil {
		if os.IsNotExist(errhfexist) {
			fmt.Println("HEP File doesnt exists - Creating", errhfexist)
			_, errhfcreate := os.Create("./HEP/HEP-saved.arch")
			fmt.Println("-->|| Creating HEP file")
			if errhfcreate != nil {
				fmt.Println("Create file error", errhfcreate)
				return
			}
		}
	}

	fmt.Printf("Listening for HEP on: %v\nProxying HEP to: %v\n\n", *localAddr, *remoteAddr)

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
	go closeConn(complete)

	for {
		conn, err := listener.AcceptTCP()
		log.Println("-->|| New connection from", conn.RemoteAddr())
		if err != nil {
			log.Println(err)
			return
		}
		pending <- conn
	}
}
