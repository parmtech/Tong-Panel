//go:build acme_e2e

package acmeprototype

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/letsencrypt/pebble/v2/ca"
	"github.com/letsencrypt/pebble/v2/db"
	"github.com/letsencrypt/pebble/v2/va"
	"github.com/letsencrypt/pebble/v2/wfe"
	"github.com/miekg/dns"
)

// These paths belong only to the pinned Pebble fixture, not to a CA profile or
// a production endpoint policy. No test helper is built into the panel.
const (
	pebbleNewOrderPath = "/order-plz"
	pebbleNewAccount   = "/sign-me-up"
	pebbleOrderPath    = "/my-order/"
	pebbleFinalize     = "/finalize-order/"
	pebbleAccountPath  = "/my-account/"
	pebbleCertPath     = "/certZ/"
)

type pebbleFixtureOptions struct {
	force404             bool
	afterPresent         func()
	holdFinalizeResponse bool
}

type pebbleFixture struct {
	domain, directoryURL string
	origin               *url.URL
	provider             *observedProvider
	authority            *ca.CAImpl
	transport            *http.Transport
	dnsServer            *dns.Server
	challengeServer      *httptest.Server
	acmeServer           *httptest.Server
	shutdown             chan struct{}
	finalizeAccepted     chan struct{}
	finalizeDisconnected chan struct{}
	acceptedOnce         sync.Once
	disconnectedOnce     sync.Once

	challengeRequests, rejectedRequests             atomic.Int32
	requests, newAccounts, newOrders, finalizations atomic.Int32
}

func newPebbleFixture(t *testing.T, providerContext context.Context, domain string, options pebbleFixtureOptions) *pebbleFixture {
	t.Helper()
	t.Setenv("PEBBLE_VA_ALWAYS_VALID", "0")
	t.Setenv("PEBBLE_VA_NOSLEEP", "1")
	t.Setenv("PEBBLE_WFE_NONCEREJECT", "0")
	t.Setenv("PEBBLE_AUTHZREUSE", "0")
	f := &pebbleFixture{domain: domain, shutdown: make(chan struct{}), finalizeAccepted: make(chan struct{}), finalizeDisconnected: make(chan struct{})}
	t.Cleanup(func() {
		// Release intentionally withheld responses before closing listeners,
		// including when an earlier assertion aborts the test.
		close(f.shutdown)
		if f.provider != nil {
			f.provider.Close()
		}
		if f.transport != nil {
			f.transport.CloseIdleConnections()
		}
		if f.acmeServer != nil {
			f.acmeServer.Close()
		}
		if f.challengeServer != nil {
			f.challengeServer.Close()
		}
		if f.dnsServer != nil {
			_ = f.dnsServer.Shutdown()
		}
	})
	provider, err := NewForAttempt(providerContext, domain, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	f.provider = &observedProvider{Provider: provider, afterPresent: options.afterPresent}
	f.challengeServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.challengeRequests.Add(1)
		status := &observedStatusWriter{ResponseWriter: w, status: http.StatusOK}
		if options.force404 {
			http.NotFound(status, r)
		} else {
			provider.ServeHTTP(status, r)
		}
		if status.status == http.StatusNotFound {
			f.rejectedRequests.Add(1)
		}
	}))
	httpPort := f.challengeServer.Listener.Addr().(*net.TCPAddr).Port
	// This private TCP resolver returns only the selected .test host and never
	// forwards queries. No system resolver or hosts-file changes are needed.
	dnsListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	f.dnsServer = &dns.Server{Listener: dnsListener, NotifyStartedFunc: func() { close(started) }, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, question *dns.Msg) {
		reply := new(dns.Msg)
		reply.SetReply(question)
		for _, q := range question.Question {
			if q.Name == dns.Fqdn(domain) && q.Qtype == dns.TypeA {
				reply.Answer = append(reply.Answer, &dns.A{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}, A: net.ParseIP("127.0.0.1")})
			}
		}
		_ = w.WriteMsg(reply)
	})}
	dnsResult := make(chan error, 1)
	go func() { dnsResult <- f.dnsServer.ActivateAndServe() }()
	select {
	case <-started:
	case <-dnsResult:
		t.Fatal("local DNS fixture stopped before startup")
	case <-time.After(2 * time.Second):
		t.Fatal("local DNS fixture did not start")
	}
	logger := log.New(io.Discard, "", 0)
	store := db.NewMemoryStore()
	f.authority = ca.New(logger, store, "", "ecdsa", 0, 1, map[string]ca.Profile{"default": {Description: "local fixture", ValidityPeriod: 3600}})
	validator := va.New(logger, httpPort, 0, false, dnsListener.Addr().String(), store)
	frontEnd := wfe.New(logger, store, validator, f.authority, []string{"pebble.test"}, false, false, 1, 1)
	front := frontEnd.Handler()
	f.acmeServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		if r.Method == http.MethodPost && r.URL.Path == pebbleNewAccount {
			f.newAccounts.Add(1)
		}
		if r.Method == http.MethodPost && r.URL.Path == pebbleNewOrderPath {
			f.newOrders.Add(1)
		}
		if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, pebbleFinalize) {
			front.ServeHTTP(w, r)
			return
		}
		f.finalizations.Add(1)
		if !options.holdFinalizeResponse {
			front.ServeHTTP(w, r)
			return
		}
		// Let the real CA accept and process the signed CSR, but never deliver
		// its successful finalize response to the interrupted client.
		buffered := httptest.NewRecorder()
		front.ServeHTTP(buffered, r)
		if buffered.Code != http.StatusOK {
			for key, values := range buffered.Header() {
				w.Header()[key] = values
			}
			w.WriteHeader(buffered.Code)
			_, _ = io.Copy(w, buffered.Body)
			return
		}
		f.acceptedOnce.Do(func() { close(f.finalizeAccepted) })
		select {
		case <-r.Context().Done():
			f.disconnectedOnce.Do(func() { close(f.finalizeDisconnected) })
		case <-f.shutdown:
		}
	}))
	f.origin, err = url.Parse(f.acmeServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	f.directoryURL = f.acmeServer.URL + wfe.DirectoryPath
	roots := x509.NewCertPool()
	roots.AddCert(f.acmeServer.Certificate())
	f.transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != f.origin.Host {
			return nil, errors.New("non-fixture network target refused")
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", address)
	}}
	return f
}

func (f *pebbleFixture) httpClient(ctx context.Context, timeout time.Duration) (*http.Client, error) {
	if timeout <= 0 {
		return nil, ErrInput
	}
	transport, err := newAttemptTransport(ctx, f.transport)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Host != f.origin.Host || r.URL.Scheme != "https" || r.URL.User != nil || len(via) > 3 {
			return errors.New("non-fixture redirect refused")
		}
		return nil
	}}, nil
}

func (f *pebbleFixture) newClient(t *testing.T, ctx context.Context, timeout time.Duration, receipt *localOrderReceipt) (*lego.Client, *localAccount) {
	t.Helper()
	httpClient, err := f.httpClient(ctx, timeout)
	if err != nil {
		t.Fatal(err)
	}
	if receipt != nil {
		httpClient.Transport = &orderObservingTransport{next: httpClient.Transport, fixture: f, receipt: receipt}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	account := &localAccount{key: key}
	config := &lego.Config{CADirURL: f.directoryURL, User: account, HTTPClient: httpClient,
		Certificate: lego.CertificateConfig{KeyType: certcrypto.EC256, Timeout: 20 * time.Second}}
	// Lego's internal polling sleeps do not accept the attempt context. The
	// transport fences network work; this explicit polling bound remains needed.
	client, err := lego.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	account.registration, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Challenge.SetHTTP01Provider(f.provider); err != nil {
		t.Fatal(err)
	}
	return client, account
}

func (f *pebbleFixture) validResourceURL(raw, prefix string) bool {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Scheme != "https" || u.Host != f.origin.Host || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || u.Opaque != "" || !strings.HasPrefix(u.Path, prefix) {
		return false
	}
	id := strings.TrimPrefix(u.Path, prefix)
	return id != "" && len(id) <= 256 && base64url.MatchString(id)
}

func (f *pebbleFixture) verifyCertificate(chain, privateKey []byte) (*x509.Certificate, error) {
	block, remaining := pem.Decode(chain)
	if block == nil {
		return nil, errors.New("missing local fixture certificate")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("invalid local fixture certificate")
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != f.domain || leaf.VerifyHostname(f.domain) != nil {
		return nil, errors.New("wrong certificate SAN")
	}
	roots := x509.NewCertPool()
	roots.AddCert(f.authority.GetRootCert(0).Cert)
	intermediates := x509.NewCertPool()
	if !intermediates.AppendCertsFromPEM(remaining) {
		return nil, errors.New("missing intermediate chain")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{DNSName: f.domain, Roots: roots, Intermediates: intermediates}); err != nil {
		return nil, errors.New("certificate chain verification failed")
	}
	if _, err := tls.X509KeyPair(chain, privateKey); err != nil {
		return nil, errors.New("certificate does not match retained key")
	}
	return leaf, nil
}

func (f *pebbleFixture) assertClosed(t *testing.T) {
	t.Helper()
	p := f.provider.Provider
	p.mu.Lock()
	closed := p.closed && p.stop == nil && p.token == "" && p.keyAuth == "" && p.expires.IsZero()
	p.mu.Unlock()
	if !closed {
		t.Fatal("interrupted provider retained state or cancellation registration")
	}
	value := strings.Repeat("a", 32)
	keyAuth := value + "." + strings.Repeat("b", 43)
	if err := p.Present(f.domain, value, keyAuth); !errors.Is(err, ErrClosed) {
		t.Fatal("interrupted provider admitted another challenge")
	}
	if err := p.CleanUp(f.domain, value, keyAuth); err != nil {
		t.Fatal("interrupted provider cleanup was not idempotent")
	}
	if got := request(p, http.MethodGet, f.domain, http01.ChallengePath(value)); got.Code != http.StatusNotFound {
		t.Fatal("interrupted provider admitted another HTTP response")
	}
}
