package outputs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubearmor/sidekick/types"
)

var TestInput = `
{
  "Timestamp": 1631542902,
  "UpdatedTime": "2023-09-13T15:35:02Z",
  "ClusterName": "default",
  "HostName": "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r",
  "EventType": "MatchedPolicy",
  "Detail": {
    "OwnerRef": "Deployment",
    "OwnerName": "wordpress",
    "OwnerNamespace": "wordpress-mysql",
    "Timestamp": "1631542902",
    "UpdatedTime": "2023-09-13T15:35:02Z",
    "ClusterName": "default",
    "Hostname": "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r",
    "NamespaceName": "wordpress-mysql",
    "PodName": "wordpress-7c966b5d85-xvsrl",
    "Labels": "app=wordpress",
    "ContainerID": "80eead8fb840e9f3f3b1bea94bb202a798b92ad8ba4e0c92f52c4027dab98e73",
    "ContainerName": "wordpress",
    "ContainerImage": "docker.io/library/wordpress:4.8-apache@sha256:6216f64ab88fc51d311e38c7f69ca3f9aaba621492b4f1fa93ddf63093768845",
    "HostPPID": "102114",
    "HostPID": "102947",
    "PPID": "203",
    "PID": "217",
    "UID": "1001",
    "ParentProcessName": "/bin/bash",
    "ProcessName": "/bin/ls",
    "Source": "/bin/ls",
    "Operation": "File",
    "Resource": "/var",
    "Data": "syscall=SYS_OPENAT fd=-100 flags=O_RDONLY|O_NONBLOCK|O_DIRECTORY|O_CLOEXEC",
    "Result": "Passed",
    "PolicyName": "DefaultPosture",
    "Severity": "Medium",
    "Tags": "Tag1,Tag2",
    "ATags": "ATag1,ATag2",
    "Message": "Policy Matched",
    "Enforcer": "eBPF Monitor"
  }
}`

func TestNewClient(t *testing.T) {
	u, _ := url.Parse("http://localhost")

	config := &types.Configuration{}
	stats := &types.Statistics{}
	promStats := &types.PromStatistics{}

	testClientOutput := Client{OutputType: "test", EndpointURL: u, MutualTLSEnabled: false, CheckCert: true, HeaderList: []Header{}, ContentType: "application/json; charset=utf-8", Config: config, Stats: stats, PromStats: promStats}
	_, err := NewClient("test", "localhost/%*$¨^!/:;", false, true, config, stats, promStats, nil, nil)
	require.NotNil(t, err)

	nc, err := NewClient("test", "http://localhost", false, true, config, stats, promStats, nil, nil)
	require.Nil(t, err)
	require.Equal(t, &testClientOutput, nc)
}

func TestPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected method : POST, got %s\n", r.Method)
		}
		switch r.URL.EscapedPath() {
		case "/200":
			w.WriteHeader(http.StatusOK)
		case "/400":
			w.WriteHeader(http.StatusBadRequest)
		case "/401":
			w.WriteHeader(http.StatusUnauthorized)
		case "/403":
			w.WriteHeader(http.StatusForbidden)
		case "/404":
			w.WriteHeader(http.StatusNotFound)
		case "/422":
			w.WriteHeader(http.StatusUnprocessableEntity)
		case "/429":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/502":
			w.WriteHeader(http.StatusBadGateway)
		}
	}))

	for i, j := range map[string]error{
		"/200": nil, "/400": ErrHeaderMissing,
		"/401": ErrClientAuthenticationError,
		"/403": ErrForbidden,
		"/404": ErrNotFound,
		"/422": ErrUnprocessableEntityError,
		"/429": ErrTooManyRequest,
		"/502": ErrBadGateway,
	} {
		nc, err := NewClient("", ts.URL+i, false, true, &types.Configuration{}, &types.Statistics{}, &types.PromStatistics{}, nil, nil)
		require.Nil(t, err)
		require.NotEmpty(t, nc)

		errPost := nc.Post("")
		require.Equal(t, errPost, j)
	}
}

func TestAddHeader(t *testing.T) {
	headerKey, headerVal := "key", "val"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		passedVal := r.Header.Get(headerKey)
		require.Equal(t, passedVal, headerVal)
	}))
	nc, err := NewClient("", ts.URL, false, true, &types.Configuration{}, &types.Statistics{}, &types.PromStatistics{}, nil, nil)
	require.Nil(t, err)
	require.NotEmpty(t, nc)

	nc.AddHeader(headerKey, headerVal)

	nc.Post("")
}

func TestAddBasicAuth(t *testing.T) {
	username, password := "user", "pass"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// I'm not comfortable using the constants here - seems too easy to fat-finger a change in
		// one location and break auth across a bunch of apps. Seems like the solution will be
		// more robust if we check against the _actual string_ "Authorization".
		passedVal := r.Header.Get("Authorization")
		// We have to have content here
		if passedVal == "" {
			t.Fatalf("Input Authorization header was empty")
		}

		splitVal := strings.Split(passedVal, " ")

		if len(splitVal) != 2 {
			t.Fatalf("Basic Authorization header value must be able to be split by a space into \"Basic\" and a digest")
		}

		basicDeclarator := splitVal[0]
		digest := splitVal[1]

		require.Equal(t, basicDeclarator, "Basic")

		decodedDigestBytes, err := base64.StdEncoding.DecodeString(digest)
		require.Nil(t, err)
		decodedDigest := string(decodedDigestBytes)

		splitDigest := strings.Split(decodedDigest, ":")

		if len(splitDigest) != 2 {
			t.Fatalf("Decoded digest split on a colon must have two elements - user and password.")
		}

		passedUsername := splitDigest[0]
		passedPassword := splitDigest[1]

		require.Equal(t, passedUsername, username)
		require.Equal(t, passedPassword, password)
		// I used https://www.base64encode.org/ to encode "user:pass" in base64,
		// and that should be the provided value.
		require.Equal(t, digest, "dXNlcjpwYXNz")
	}))
	nc, err := NewClient("", ts.URL, false, true, &types.Configuration{}, &types.Statistics{}, &types.PromStatistics{}, nil, nil)
	require.Nil(t, err)
	require.NotEmpty(t, nc)

	nc.BasicAuth(username, password)

	nc.Post("")
}

func TestHeadersResetAfterReq(t *testing.T) {
	headerKey, headerVal := http.CanonicalHeaderKey("key"), "val"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		passedList := r.Header[headerKey]
		require.Equal(t, 1, len(passedList), "Expected %v to have 1 element", passedList)
	}))

	nc, err := NewClient("", ts.URL, false, true, &types.Configuration{}, &types.Statistics{}, &types.PromStatistics{}, nil, nil)
	require.Nil(t, err)
	require.NotEmpty(t, nc)

	nc.AddHeader(headerKey, headerVal)

	nc.Post("")

	nc.AddHeader(headerKey, headerVal)

	nc.Post("")
}

func TestMutualTlsPost(t *testing.T) {
	config := &types.Configuration{}
	config.MutualTLSFilesPath = "/tmp/kubearmor/client"
	config.MutualTLSClient.CertFile = "/tmp/kubearmor/client/client.crt"
	config.MutualTLSClient.KeyFile = "/tmp/kubearmor/client/client.key"
	config.MutualTLSClient.CaCertFile = "/tmp/kubearmor/client/ca.crt"
	// delete folder to avoid makedir failure
	os.RemoveAll(config.MutualTLSFilesPath)

	serverTLSConf, err := certsetup(config)
	if err != nil {
		require.Nil(t, err)
	}

	tlsURL := "127.0.0.1:5443"

	// set up the httptest.Server using our certificate signed by our CA
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected method : POST, got %s\n", r.Method)
		}
		if r.URL.EscapedPath() == "/200" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	// This Listen config is required since server.URL generates a "Server already started" Panic error
	// Check https://golang.org/src/net/http/httptest/server.go#:~:text=s.URL
	l, _ := net.Listen("tcp", tlsURL)
	server.Listener = l
	server.TLS = serverTLSConf
	server.StartTLS()
	defer server.Close()

	nc, err := NewClient("", server.URL+"/200", true, true, config, &types.Statistics{}, &types.PromStatistics{}, nil, nil)
	require.Nil(t, err)
	require.NotEmpty(t, nc)

	errPost := nc.Post("")
	require.Nil(t, errPost)

}

func certsetup(config *types.Configuration) (serverTLSConf *tls.Config, err error) {
	err = os.MkdirAll(config.MutualTLSFilesPath, 0755)
	if err != nil {
		return nil, err
	}

	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Kubearmor"},
			Country:       []string{"Bharat"},
			Province:      []string{""},
			Locality:      []string{"New Delhi"},
			StreetAddress: []string{"Kubearmor nagar"},
			PostalCode:    []string{"110074"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// save ca to ca.crt file (it will be used by Client)
	err = ioutil.WriteFile(config.MutualTLSClient.CaCertFile, caPEM.Bytes(), 0600)
	if err != nil {
		return nil, err
	}

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Falco"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Falcosidekick st"},
			PostalCode:    []string{"94016"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// create server private key
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// sign server certificate with CA key
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	// create server TLS config
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caPEM.Bytes())
	serverTLSConf = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}

	// create client certificate
	clientCert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Falcosidekick"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Falcosidekickclient st"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// create client private key
	clientCertPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// sign client certificate with CA key
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientCert, ca, &clientCertPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	clientCertPEM := new(bytes.Buffer)
	pem.Encode(clientCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCertBytes,
	})

	// save client cert and key to client.crt and client.key
	err = ioutil.WriteFile(config.MutualTLSClient.CertFile, clientCertPEM.Bytes(), 0600)
	if err != nil {
		return nil, err
	}
	clientCertPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(clientCertPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientCertPrivKey),
	})
	err = ioutil.WriteFile(config.MutualTLSClient.KeyFile, clientCertPrivKeyPEM.Bytes(), 0600)
	if err != nil {
		return nil, err
	}
	return serverTLSConf, nil
}
