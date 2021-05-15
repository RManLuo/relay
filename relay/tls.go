package relay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func GetCert() {
	_, err := os.Stat(CertFile)
	if err != nil {
		fmt.Println(CertFile, "Not exit, generating ...")
		CreateTLSFile(CertFile, KeyFile)
	}

	_, err = os.Stat(KeyFile)
	if err != nil {
		fmt.Println(KeyFile, "Not exit, generating ...")
		CreateTLSFile(CertFile, KeyFile)
	}
}

func (s *Relay) RunTlsServer() error {
	return nil
}

func Tls_Handle(s *Relay) error {
	return nil
}

func sendRequest(url string, body io.Reader, addHeaders map[string]string, method string) (statuscode int, resp []byte, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")

	if len(addHeaders) > 0 {
		for k, v := range addHeaders {
			req.Header.Add(k, v)
		}
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()

	statuscode = response.StatusCode
	resp, err = ioutil.ReadAll(response.Body)
	return
}

func CreateTLSFile(certFile, keyFile string) {
	var ip string
	os.Remove(certFile)
	os.Remove(keyFile)
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, max)
	subject := pkix.Name{
		Country:            []string{"US"},
		Province:           []string{"WDC"},
		Organization:       []string{"Microsoft Corporation"},
		OrganizationalUnit: []string{"Microsoft Corporation"},
		CommonName:         "www.microstft.com",
	}

	_, resp, err := sendRequest("https://api.ip.sb/ip", nil, nil, "GET")
	if err == nil {
		ip = string(resp)
		ip = strings.Replace(ip, "\n", "", -1)
	} else {
		ip = "127.0.0.1"
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip)},
	}

	pk, _ := rsa.GenerateKey(rand.Reader, 2048)

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk)
	certOut, _ := os.Create(certFile)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, _ := os.Create(keyFile)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
	keyOut.Close()
}
