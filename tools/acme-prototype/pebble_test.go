//go:build acme_e2e

package acmeprototype

import (
	"context"
	"crypto"
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/letsencrypt/pebble/v2/ca"
	"github.com/letsencrypt/pebble/v2/db"
	"github.com/letsencrypt/pebble/v2/va"
	"github.com/letsencrypt/pebble/v2/wfe"
	"github.com/miekg/dns"
)

type localAccount struct {
	key          crypto.PrivateKey
	registration *registration.Resource
}

func (a *localAccount) GetEmail() string                        { return "" }
func (a *localAccount) GetPrivateKey() crypto.PrivateKey        { return a.key }
func (a *localAccount) GetRegistration() *registration.Resource { return a.registration }

type observedProvider struct {
	*Provider
	presented, cleaned atomic.Int32
}

func (p *observedProvider) Present(domain, token, keyAuth string) error {
	if err := p.Provider.Present(domain, token, keyAuth); err != nil {
		return err
	}
	p.presented.Add(1)
	return nil
}
func (p *observedProvider) CleanUp(domain, token, keyAuth string) error {
	if err := p.Provider.CleanUp(domain, token, keyAuth); err != nil {
		return err
	}
	p.cleaned.Add(1)
	return nil
}

// This explicitly selected test uses reserved .test names, ephemeral loopback
// listeners, real challenge validation and verified local TLS. Keys stay in RAM.
func TestPebbleHTTP01(t *testing.T) {
	t.Setenv("PEBBLE_VA_ALWAYS_VALID", "0")
	t.Setenv("PEBBLE_VA_NOSLEEP", "1")
	t.Setenv("PEBBLE_WFE_NONCEREJECT", "0")
	t.Setenv("PEBBLE_AUTHZREUSE", "0")
	logger := log.New(io.Discard, "", 0)
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "invalid-challenge"}[failed], func(t *testing.T) {
			domain := "valid.acme.test"
			if failed {
				domain = "invalid.acme.test"
			}
			provider, err := New(domain, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			observed := &observedProvider{Provider: provider}
			var challengeRequests atomic.Int32
			challengeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				challengeRequests.Add(1)
				if failed {
					http.NotFound(w, r)
					return
				}
				provider.ServeHTTP(w, r)
			}))
			defer challengeServer.Close()
			httpPort := challengeServer.Listener.Addr().(*net.TCPAddr).Port
			// Pebble uses this TCP resolver; it never forwards a DNS query.
			dnsListener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			dnsServer := &dns.Server{Listener: dnsListener, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, question *dns.Msg) {
				reply := new(dns.Msg)
				reply.SetReply(question)
				for _, q := range question.Question {
					if q.Name == dns.Fqdn(domain) && q.Qtype == dns.TypeA {
						reply.Answer = append(reply.Answer, &dns.A{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}, A: net.ParseIP("127.0.0.1")})
					}
				}
				_ = w.WriteMsg(reply)
			})}
			go func() { _ = dnsServer.ActivateAndServe() }()
			defer dnsServer.Shutdown()
			store := db.NewMemoryStore()
			authority := ca.New(logger, store, "", "ecdsa", 0, 1, map[string]ca.Profile{"default": {Description: "local fixture", ValidityPeriod: 3600}})
			validator := va.New(logger, httpPort, 0, false, dnsListener.Addr().String(), store)
			front := wfe.New(logger, store, validator, authority, []string{"pebble.test"}, false, false, 1, 1)
			acmeServer := httptest.NewTLSServer(front.Handler())
			defer acmeServer.Close()
			serverURL, _ := url.Parse(acmeServer.URL)
			roots := x509.NewCertPool()
			roots.AddCert(acmeServer.Certificate())
			transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				if address != serverURL.Host {
					return nil, errors.New("non-fixture network target refused")
				}
				return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", address)
			}}
			defer transport.CloseIdleConnections()
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			account := &localAccount{key: key}
			config := &lego.Config{CADirURL: acmeServer.URL + wfe.DirectoryPath, User: account, HTTPClient: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
				if r.URL.Host != serverURL.Host || r.URL.Scheme != "https" || len(via) > 3 {
					return errors.New("non-fixture redirect refused")
				}
				return nil
			}}, Certificate: lego.CertificateConfig{KeyType: certcrypto.EC256, Timeout: 20 * time.Second}}
			client, err := lego.NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			account.registration, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err != nil {
				t.Fatal(err)
			}
			if err := client.Challenge.SetHTTP01Provider(observed); err != nil {
				t.Fatal(err)
			}
			resource, err := client.Certificate.Obtain(certificate.ObtainRequest{Domains: []string{domain}, Bundle: true})
			if failed {
				if err == nil || resource != nil {
					t.Fatal("invalid challenge unexpectedly issued a certificate")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				block, remaining := pem.Decode(resource.Certificate)
				if block == nil {
					t.Fatal("missing local fixture certificate")
				}
				leaf, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					t.Fatal(err)
				}
				if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != domain || leaf.VerifyHostname(domain) != nil {
					t.Fatal("wrong certificate SAN")
				}
				pool := x509.NewCertPool()
				pool.AddCert(authority.GetRootCert(0).Cert)
				intermediates := x509.NewCertPool()
				if !intermediates.AppendCertsFromPEM(remaining) {
					t.Fatal("missing intermediate chain")
				}
				if _, err := leaf.Verify(x509.VerifyOptions{DNSName: domain, Roots: pool, Intermediates: intermediates}); err != nil {
					t.Fatal(err)
				}
				if _, err := tls.X509KeyPair(resource.Certificate, resource.PrivateKey); err != nil {
					t.Fatal(err)
				}
			}
			if observed.presented.Load() != 1 || observed.cleaned.Load() != 1 || challengeRequests.Load() == 0 {
				t.Fatal("presentation, actual validation or cleanup missing")
			}
			provider.mu.Lock()
			remainingToken := provider.token
			provider.mu.Unlock()
			if remainingToken != "" {
				t.Fatal("challenge remained after order completion/failure")
			}
		})
	}
}
