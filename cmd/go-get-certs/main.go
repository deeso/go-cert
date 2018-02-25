package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Row struct {
	Popularity int
	Hostname   string
}

type NameJson struct {
	Country            []string `json:"Country"`
	Organization       []string `json:"Organization"`
	OrganizationalUnit []string `json:"OrganizationalUnit"`
	Locality           []string `json:"Locality"`
	Province           []string `json:"Province"`
	StreetAddress      []string `json:"StreetAddress"`
	PostalCode         []string `json:"PostalCode"`
}

type CertificateJson struct {
	Signature                   string       `json:"Signature"`
	SignatureAlgorithm          string       `json:"SignatureAlgorithm"`
	PublicKeyAlgorithm          string       `json:"PublicKeyAlgorithm"`
	Version                     int          `json:"Version"`
	SerialNumber                string       `json:"SerialNumber"`
	Issuer                      NameJson     `json:"Issuer"`
	Subject                     NameJson     `json:"Subject"`
	NotBefore                   string       `json:"NotBefore"`
	NotAfter                    string       `json:"NotAfter"`
	KeyUsage                    int          `json:"KeyUsage"`
	OCSPServer                  []string     `json:"OCSPServer"`
	IssuingCertificateURL       []string     `json:"IssuingCertificateURL"`
	DNSNames                    []string     `json:"DNSNames"`
	EmailAddresses              []string     `json:"EmailAddresses"`
	IPAddresses                 []net.IP     `json:"IPAddresses"`
	URIs                        []*url.URL   `json:"URIs"`
	PermittedDNSDomainsCritical bool         `json:"PermittedDNSDomainsCritical"`
	PermittedDNSDomains         []string     `json:"PermittedDNSDomains"`
	ExcludedDNSDomains          []string     `json:"ExcludedDNSDomains"`
	PermittedIPRanges           []*net.IPNet `json:"PermittedIPRanges"`
	ExcludedIPRanges            []*net.IPNet `json:"ExcludedIPRanges"`
	PermittedEmailAddresses     []string     `json:"PermittedEmailAddresses"`
	ExcludedEmailAddresses      []string     `json:"ExcludedEmailAddresses"`
	PermittedURIDomains         []string     `json:"PermittedURIDomains"`
	ExcludedURIDomains          []string     `json:"ExcludedURIDomains"`
	BasicConstraintsValid       bool         `json:"BasicConstraintsValid"`
	IsCA                        bool         `json:"IsCA"`
}

type ConnectionStateJson struct {
	Version          uint16              `json:"Version"`
	ServerName       string              `json:"ServerName"`
	PeerCertificates []CertificateJson   `json:"PeerCertificates"`
	VerifiedChains   [][]CertificateJson `json:"VerifiedChains"`
}

func NameToJson(name *pkix.Name) *NameJson {
	nameJson := NameJson{
		Country:            name.Country,
		Organization:       name.Organization,
		OrganizationalUnit: name.OrganizationalUnit,
		Locality:           name.Locality,
		Province:           name.Province,
		StreetAddress:      name.StreetAddress,
		PostalCode:         name.PostalCode,
	}
	return &nameJson
}

func FormatTime(t time.Time) string {
	s := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	return s
}

func ConnectionStateToJson(cstate *tls.ConnectionState) *ConnectionStateJson {
	_pcs := make([]CertificateJson, len(cstate.PeerCertificates))
	_vchains := make([][]CertificateJson, len(cstate.VerifiedChains))

	num := 0
	for _, pc := range cstate.PeerCertificates {
		if pc == nil {
			continue
		}
		x := CertificateToJson(pc)
		_pcs[num] = *x
		num += 1
	}

	pcs := make([]CertificateJson, num)
	i := 0
	for i < num {
		pcs[i] = _pcs[i]
		i += 1
	}

	num = 0
	for _, pcp := range cstate.VerifiedChains {
		if pcp == nil || len(pcp) == 0 {
			continue
		}
		_vpcs := make([]CertificateJson, len(pcp))
		num2 := 0
		for k, pc := range pcp {
			if pc == nil {
				continue
			}
			x := CertificateToJson(pc)
			_vpcs[k] = *x
			num2 += 1
		}

		vpcs := make([]CertificateJson, num2)
		i = 0
		for i < num2 {
			vpcs[i] = _vpcs[i]
			i += 1
		}

		_vchains[num] = pcs
		num += 1
	}

	vchains := make([][]CertificateJson, num)
	i = 0
	for i < num {
		vchains[i] = _vchains[i]
		i += 1
	}

	csj := ConnectionStateJson{
		Version:          cstate.Version,
		ServerName:       cstate.ServerName,
		PeerCertificates: pcs,
		VerifiedChains:   vchains,
	}
	return &csj
}

func CertificateToJson(cert *x509.Certificate) *CertificateJson {

	sn, _ := cert.SerialNumber.GobEncode()

	cj := CertificateJson{
		Signature:          hex.EncodeToString(cert.Signature),
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		// PublicKeyAlgorithm: cert.PublicKeyAlgorithm.String(),
		Version:               cert.Version,
		SerialNumber:          hex.EncodeToString(sn),
		Issuer:                *NameToJson(&cert.Issuer),
		Subject:               *NameToJson(&cert.Subject),
		NotBefore:             FormatTime(cert.NotBefore),
		NotAfter:              FormatTime(cert.NotAfter),
		KeyUsage:              int(cert.KeyUsage),
		OCSPServer:            cert.OCSPServer,
		IssuingCertificateURL: cert.IssuingCertificateURL,
		DNSNames:              cert.DNSNames,
		EmailAddresses:        cert.EmailAddresses,
		IPAddresses:           cert.IPAddresses,
		// URIs:                  cert.URIs,
		PermittedDNSDomainsCritical: cert.PermittedDNSDomainsCritical,
		PermittedDNSDomains:         cert.PermittedDNSDomains,
		// ExcludedDNSDomains:          cert.ExcludedDNSDomains,
		// PermittedIPRanges:           cert.PermittedIPRanges,
		// ExcludedIPRanges:            cert.ExcludedIPRanges,
		// PermittedEmailAddresses: cert.PermittedEmailAddresses,
		// ExcludedEmailAddresses:  cert.ExcludedEmailAddresses,
		// PermittedURIDomains:     cert.PermittedURIDomains,
		// ExcludedURIDomains:      cert.ExcludedURIDomains,
		BasicConstraintsValid: cert.BasicConstraintsValid,
		IsCA: cert.IsCA,
	}
	return &cj

}

func SslConnect(hostnamePtr *string, hostPortPtr *int) (*ConnectionStateJson, error) {
	hostPortStr := strconv.Itoa(*hostPortPtr)
	var b bytes.Buffer
	b.WriteString(*hostnamePtr)
	b.WriteString(":")
	b.WriteString(hostPortStr)

	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	dialer := net.Dialer{}
	seconds := 3
	timeOut := time.Duration(seconds) * time.Second
	dialer.Timeout = timeOut
	conn, err := tls.DialWithDialer(&dialer, "tcp", b.String(), conf)
	if err != nil {
		// log.Println(err)
		return nil, err
	}
	// err : = conn.Handshake()
	if err != nil {
		// log.Println(err)
		return nil, err
	}
	defer conn.Close()
	err = conn.Handshake()
	if err != nil {
		// log.Println(err)
		return nil, err
	}

	cstate := conn.ConnectionState()
	csj := ConnectionStateToJson(&cstate)
	// njIssuerB, _ := json.Marshal(csj)
	// fmt.Println(string(njIssuerB))

	return csj, nil

}

func RunAndPrint(pos int, hostname string) {

	hostPort := 443
	// log.Println("Processing: ", hostname)
	csj, err := SslConnect(&hostname, &hostPort)
	if err != nil {
		hostname2 := "www." + hostname
		csj, err = SslConnect(&hostname2, &hostPort)
	}
	if csj != nil {
		njIssuerB, _ := json.Marshal(csj)
		fmt.Println(string(njIssuerB))
	} else if err != nil {
		out := fmt.Sprintf("[% 8s:%s] %s", strconv.Itoa(pos), hostname, err.Error())
		log.Println(out)
	}

}

func ProcessCsv(csvFileName *string) {
	csvFile, _ := os.Open(*csvFileName)
	reader := csv.NewReader(bufio.NewReader(csvFile))
	var rows []Row
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		num, _ := strconv.Atoi(line[0])
		rows = append(rows, Row{num, line[1]})
	}
	maxGoroutines := 100
	guard := make(chan struct{}, maxGoroutines)
	// hostPort := 443
	// pos := 0
	for pos, row := range rows {
		hostname := row.Hostname
		guard <- struct{}{}
		go func(pos int, hostname string) {
			hostPort := 443
			// log.Println("Processing: ", hostname)
			csj, err := SslConnect(&hostname, &hostPort)
			if err != nil {
				hostname2 := "www." + hostname
				csj, err = SslConnect(&hostname2, &hostPort)
			}
			if csj != nil {
				njIssuerB, _ := json.Marshal(csj)
				fmt.Println(string(njIssuerB))
			} else if err != nil {
				out := fmt.Sprintf("[% 8s:%s] %s", strconv.Itoa(pos), hostname, err.Error())
				log.Println(out)
			}
			<-guard
		}(pos, hostname)
		// // log.Println("Processing: ", hostname)
		// csj, err := SslConnect(&hostname, &hostPort)
		// if err != nil {
		// 	hostname2 := "www." + hostname
		// 	csj, err = SslConnect(&hostname2, &hostPort)
		// }
		// if csj != nil {
		// 	njIssuerB, _ := json.Marshal(csj)
		// 	fmt.Println(string(njIssuerB))
		// } else if err != nil {
		// 	out := fmt.Sprintf("[% 8s:%s] %s", strconv.Itoa(pos), hostname, err.Error())
		// 	log.Println(out)
		// }
	}
	var input string
	fmt.Scanln(&input)
}

func main() {
	hostnamePtr := flag.String("host", "127.0.0.1", "hostname dial")
	hostPortPtr := flag.Int("port", 443, "hostport dial")
	csvFilePtr := flag.String("csvFile", "", "csvFile")

	flag.Parse()

	if len(*csvFilePtr) > 0 {
		ProcessCsv(csvFilePtr)
		return
	}

	csj, err := SslConnect(hostnamePtr, hostPortPtr)
	if err == nil {
		njIssuerB, _ := json.Marshal(csj)
		fmt.Println(string(njIssuerB))
	} else {
		log.Println(err)
	}

	return
}
