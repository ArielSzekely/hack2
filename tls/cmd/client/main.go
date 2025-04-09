//package main
//
//import (
//	"bytes"
//	"crypto/rand"
//	"crypto/rsa"
//	"crypto/tls"
//	"crypto/x509"
//	"crypto/x509/pkix"
//	"encoding/pem"
//	"fmt"
//	"io/ioutil"
//	"log"
//	"math/big"
//	"net"
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"time"
//)
//
//func main() {
//	// get our ca and server certificate
//	serverTLSConf, clientTLSConf, err := certsetup()
//	if err != nil {
//		panic(err)
//	}
//
//	// set up the httptest.Server using our certificate signed by our CA
//	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprintln(w, "success!")
//	}))
//	server.TLS = serverTLSConf
//	server.StartTLS()
//	defer server.Close()
//
//	// communicate with the server using an http.Client configured to trust our CA
//	transport := &http.Transport{
//		TLSClientConfig: clientTLSConf,
//	}
//	http := http.Client{
//		Transport: transport,
//	}
//	start := time.Now()
//	resp, err := http.Get(server.URL)
//	if err != nil {
//		panic(err)
//	}
//	log.Printf("Time HTTP get: %v", time.Since(start))
//
//	// verify the response
//	respBodyBytes, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		panic(err)
//	}
//	body := strings.TrimSpace(string(respBodyBytes[:]))
//	if body == "success!" {
//		fmt.Println(body)
//	} else {
//		panic("not successful!")
//	}
//}
//
//func certsetup() (serverTLSConf *tls.Config, clientTLSConf *tls.Config, err error) {
//	// set up our CA certificate
//	ca := &x509.Certificate{
//		SerialNumber: big.NewInt(2019),
//		Subject: pkix.Name{
//			Organization:  []string{"Company, INC."},
//			Country:       []string{"US"},
//			Province:      []string{""},
//			Locality:      []string{"San Francisco"},
//			StreetAddress: []string{"Golden Gate Bridge"},
//			PostalCode:    []string{"94016"},
//		},
//		NotBefore:             time.Now(),
//		NotAfter:              time.Now().AddDate(10, 0, 0),
//		IsCA:                  true,
//		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
//		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
//		BasicConstraintsValid: true,
//	}
//
//	// create our private and public key
//	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	// create the CA
//	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	// pem encode
//	caPEM := new(bytes.Buffer)
//	pem.Encode(caPEM, &pem.Block{
//		Type:  "CERTIFICATE",
//		Bytes: caBytes,
//	})
//
//	caPrivKeyPEM := new(bytes.Buffer)
//	pem.Encode(caPrivKeyPEM, &pem.Block{
//		Type:  "RSA PRIVATE KEY",
//		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
//	})
//
//	// set up our server certificate
//	cert := &x509.Certificate{
//		SerialNumber: big.NewInt(2019),
//		Subject: pkix.Name{
//			Organization:  []string{"Company, INC."},
//			Country:       []string{"US"},
//			Province:      []string{""},
//			Locality:      []string{"San Francisco"},
//			StreetAddress: []string{"Golden Gate Bridge"},
//			PostalCode:    []string{"94016"},
//		},
//		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
//		NotBefore:    time.Now(),
//		NotAfter:     time.Now().AddDate(10, 0, 0),
//		SubjectKeyId: []byte{1, 2, 3, 4, 6},
//		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
//		KeyUsage:     x509.KeyUsageDigitalSignature,
//	}
//
//	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	certPEM := new(bytes.Buffer)
//	pem.Encode(certPEM, &pem.Block{
//		Type:  "CERTIFICATE",
//		Bytes: certBytes,
//	})
//
//	certPrivKeyPEM := new(bytes.Buffer)
//	pem.Encode(certPrivKeyPEM, &pem.Block{
//		Type:  "RSA PRIVATE KEY",
//		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
//	})
//
//	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
//	if err != nil {
//		return nil, nil, err
//	}
//
//	serverTLSConf = &tls.Config{
//		Certificates: []tls.Certificate{serverCert},
//	}
//
//	certpool := x509.NewCertPool()
//	certpool.AppendCertsFromPEM(caPEM.Bytes())
//	clientTLSConf = &tls.Config{
//		RootCAs: certpool,
//	}
//
//	return
//}

package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"proxy/pipe"
)

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
}

func main() {
	if len(os.Args) != 5 {
		log.Fatalf("Usage: %v <srvaddr> <ndial> <sleep> <noproxy|netproxy|pipeproxy>", os.Args[0])
	}
	cerBytes, err := os.ReadFile("keys/server.crt")
	if err != nil {
		log.Fatalf("Err read cert file: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(cerBytes)
	conf := &tls.Config{
		RootCAs:    roots,
		ServerName: "foo.example.org",
	}
	srvAddr := os.Args[1]
	ndial, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Can't parse ndial: %v", err)
	}
	sl, err := time.ParseDuration(os.Args[3])
	if err != nil {
		log.Fatalf("Can't parse sleep dur: %v", err)
	}
	proxytype := os.Args[4]
	ch := make(chan time.Duration)
	var wg sync.WaitGroup
	wg.Add(ndial)
	for i := 0; i < ndial; i++ {
		go func() {
			start := time.Now()
			log.Printf("Dialing!")
			var conn *tls.Conn
			switch proxytype {
			case "noproxy":
				conn, err = tls.Dial("tcp", srvAddr, conf)
				if err != nil {
					log.Fatalf("Dial srv addr %v: err %v", srvAddr, err)
				}
			case "netproxy":
				conn, err = tls.Dial("tcp", srvAddr, conf)
				if err != nil {
					log.Fatalf("Dial proxy addr %v: err %v", srvAddr, err)
				}
			case "pipeproxy":
				conn, err = tls.Dial("unix", pipe.PATH, conf)
				if err != nil {
					log.Fatalf("Dial pipe %v: err %v", pipe.PATH, err)
				}
			default:
				log.Fatalf("Unknown proxy type: %v", proxytype)
			}
			if err := conn.Handshake(); err != nil {
				log.Fatalf("Err handshake: %v", err)
			}
			log.Printf("Dial latency %v", time.Since(start))
			wg.Done()
			ch <- time.Since(start)
		}()
		time.Sleep(sl)
	}
	wg.Wait()
	var d time.Duration
	for i := 0; i < ndial; i++ {
		d += <-ch
	}
	log.Printf("Average dial time: %v", d/time.Duration(ndial))
}
