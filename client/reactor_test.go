package gircclient

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/DanielOaks/girc-go/ircmsg"
)

func TestPlainConnection(t *testing.T) {
	reactor := NewReactor()
	client := reactor.CreateServer("local")

	initialiseServerConnection(client)

	// we mock up a server connection to test the client
	listener, _ := net.Listen("tcp", ":0")

	client.Connect(listener.Addr().String(), false, nil)
	go client.ReceiveLoop()

	testServerConnection(t, reactor, client, listener)
}

func TestTLSConnection(t *testing.T) {
	reactor := NewReactor()
	client := reactor.CreateServer("local")

	initialiseServerConnection(client)

	// generate a test certificate to use
	priv, _ := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)

	duration30Days, _ := time.ParseDuration("-30h")
	notBefore := time.Now().Add(duration30Days) // valid 30 hours ago
	duration1Year, _ := time.ParseDuration("90h")
	notAfter := notBefore.Add(duration1Year) // for 90 hours

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gIRC-Go Co"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA: true,
	}

	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("::"))
	template.DNSNames = append(template.DNSNames, "localhost")

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	c := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	b, _ := x509.MarshalECPrivateKey(priv)
	k := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	// we mock up a server connection to test the client
	listenerKeyPair, _ := tls.X509KeyPair(c, k)

	var listenerTLSConfig tls.Config
	listenerTLSConfig.Certificates = make([]tls.Certificate, 0)
	listenerTLSConfig.Certificates = append(listenerTLSConfig.Certificates, listenerKeyPair)
	listener, _ := tls.Listen("tcp", ":0", &listenerTLSConfig)

	// mock up the client side too
	clientTLSCertPool := x509.NewCertPool()
	clientTLSCertPool.AppendCertsFromPEM(c)

	var clientTLSConfig tls.Config
	clientTLSConfig.RootCAs = clientTLSCertPool
	clientTLSConfig.ServerName = "localhost"
	go client.Connect(listener.Addr().String(), true, &clientTLSConfig)
	go client.ReceiveLoop()

	testServerConnection(t, reactor, client, listener)
}

func sendMessage(conn net.Conn, tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) {
	ircmsg := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := ircmsg.Line()
	if err != nil {
		return
	}
	fmt.Fprintf(conn, line)

	// need to wait for a quick moment here for TLS to process any changes this
	// message has caused
	runtime.Gosched()
	waitTime, _ := time.ParseDuration("10ms")
	time.Sleep(waitTime)
}

func initialiseServerConnection(client *ServerConnection) {
	client.InitialNick = "coolguy"
	client.InitialUser = "c"
	client.InitialRealName = "girc-go Test Client  "
}

func testServerConnection(t *testing.T, reactor Reactor, client *ServerConnection, listener net.Listener) {
	// start our reader
	conn, _ := listener.Accept()
	reader := bufio.NewReader(conn)

	var message string

	// CAP
	message, _ = reader.ReadString('\n')
	if message != "CAP LS 302\r\n" {
		t.Error(
			"Did not receive CAP LS message, received: [",
			message,
			"]",
		)
		return
	}

	sendMessage(conn, nil, "example.com", "CAP", "*", "LS", "*", "multi-prefix userhost-in-names sasl=PLAIN")
	sendMessage(conn, nil, "example.com", "CAP", "*", "LS", "chghost")

	message, _ = reader.ReadString('\n')
	if message != "CAP REQ :chghost multi-prefix sasl userhost-in-names\r\n" {
		t.Error(
			"Did not receive CAP REQ message, received: [",
			message,
			"]",
		)
		return
	}

	// these should be silently ignored
	fmt.Fprintf(conn, "\r\n\r\n\r\n")

	sendMessage(conn, nil, "example.com", "CAP", "*", "ACK", "chghost multi-prefix userhost-in-names sasl")

	message, _ = reader.ReadString('\n')
	if message != "CAP END\r\n" {
		t.Error(
			"Did not receive CAP END message, received: [",
			message,
			"]",
		)
		return
	}

	// NICK/USER
	message, _ = reader.ReadString('\n')
	if message != "NICK coolguy\r\n" {
		t.Error(
			"Did not receive NICK message, received: [",
			message,
			"]",
		)
		return
	}

	message, _ = reader.ReadString('\n')
	if message != "USER c 0 * :girc-go Test Client  \r\n" {
		t.Error(
			"Did not receive USER message, received: [",
			message,
			"]",
		)
		return
	}

	// make sure nick changes properly
	sendMessage(conn, nil, "example.com", "001", "dan", "Welcome to the gIRC-Go Test Network!")

	if client.Nick != "dan" {
		t.Error(
			"Nick was not set with 001, expected",
			"dan",
			"got",
			client.Nick,
		)
		return
	}

	// shutdown client
	reactor.Shutdown(" Get mad!  ")

	message, _ = reader.ReadString('\n')
	if message != "QUIT : Get mad!  \r\n" {
		t.Error(
			"Did not receive QUIT message, received: [",
			message,
			"]",
		)
		return
	}

	// close connection and listener
	conn.Close()
	listener.Close()
}
