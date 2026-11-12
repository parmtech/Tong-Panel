//go:build acme_e2e

package acmeprototype

import (
	"context"
	"crypto"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/registration"
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
	afterPresent       func()
}

func (p *observedProvider) Present(domain, token, keyAuth string) error {
	if err := p.Provider.Present(domain, token, keyAuth); err != nil {
		return err
	}
	p.presented.Add(1)
	if p.afterPresent != nil {
		p.afterPresent()
	}
	return nil
}

type observedStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *observedStatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
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
	for _, scenario := range []struct {
		name, domain       string
		force404, canceled bool
	}{
		{name: "valid", domain: "valid.acme.test"},
		{name: "invalid-challenge", domain: "invalid.acme.test", force404: true},
		{name: "canceled-attempt", domain: "canceled.acme.test", canceled: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			domain := scenario.domain
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			options := pebbleFixtureOptions{force404: scenario.force404}
			if scenario.canceled {
				// End only the provider attempt, after its first successful Present.
				// Lego continues the real local validation and must observe a 404;
				// this is not cancellation of the whole client order or transport.
				options.afterPresent = cancel
			}
			fixture := newPebbleFixture(t, ctx, domain, options)
			// Keep the original provider-only cancellation scenario: its HTTP
			// client has an independent context and observes actual validation.
			client, _ := fixture.newClient(t, context.Background(), 10*time.Second, nil)
			resource, err := client.Certificate.Obtain(certificate.ObtainRequest{Domains: []string{domain}, Bundle: true})
			if scenario.force404 || scenario.canceled {
				if err == nil || resource != nil {
					t.Fatal("failed or canceled challenge unexpectedly issued a certificate")
				}
				if fixture.rejectedRequests.Load() == 0 {
					t.Fatal("failed or canceled challenge did not produce a real HTTP 404")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := fixture.verifyCertificate(resource.Certificate, resource.PrivateKey); err != nil {
					t.Fatal(err)
				}
			}
			if fixture.provider.presented.Load() != 1 || fixture.provider.cleaned.Load() != 1 || fixture.challengeRequests.Load() == 0 {
				t.Fatal("presentation, actual validation or cleanup missing")
			}
			provider := fixture.provider.Provider
			provider.mu.Lock()
			empty := provider.token == "" && provider.keyAuth == "" && provider.expires.IsZero()
			closed := provider.closed && provider.stop == nil
			provider.mu.Unlock()
			if !empty {
				t.Fatal("challenge remained after order completion/failure")
			}
			if scenario.canceled && (!errors.Is(ctx.Err(), context.Canceled) || !closed) {
				t.Fatal("canceled provider attempt did not close permanently")
			}
		})
	}
}
