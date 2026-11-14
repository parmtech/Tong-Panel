//go:build acme_e2e

package acmeprototype

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/acme"
	"github.com/go-acme/lego/v4/acme/api"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
)

var (
	errLocalOrderUncertain = errors.New("local remote order outcome remains uncertain")
	errLocalOrderArtifact  = errors.New("recovered certificate does not match original request")
)

// The observer retains only a validated order Location, never a JWS, CSR,
// challenge value or key. Its state is discarded with the short-lived test.
type localOrderReceipt struct {
	mu       sync.Mutex
	orderURL string
}

func (r *localOrderReceipt) snapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.orderURL
}

type orderObservingTransport struct {
	next    http.RoundTripper
	fixture *pebbleFixture
	receipt *localOrderReceipt
}

func (o *orderObservingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := o.next.RoundTrip(r)
	if err != nil || r.Method != http.MethodPost || r.URL.Path != pebbleNewOrderPath || response.StatusCode != http.StatusCreated {
		return response, err
	}
	location := response.Header.Get("Location")
	o.receipt.mu.Lock()
	valid := o.fixture.validResourceURL(location, pebbleOrderPath) && o.receipt.orderURL == ""
	if valid {
		o.receipt.orderURL = location
	}
	o.receipt.mu.Unlock()
	if !valid {
		_ = response.Body.Close()
		return nil, ErrInput
	}
	return response, nil
}

type localOrderReference struct {
	directoryURL, accountURI, orderURL, domain string
}

// reconcileLocalOrder is a tagged lab helper, not an issuer or recovery API.
// It requires an explicit bounded fresh context and only reads the saved order
// through the same CA/account. It cannot create orders or repeat finalization.
// Keys and references are RAM-only: this does not prove process-crash recovery.
func (f *pebbleFixture) reconcileLocalOrder(ctx context.Context, reference localOrderReference, account *localAccount, certificateKey *ecdsa.PrivateKey) ([]byte, error) {
	if ctx == nil {
		return nil, ErrInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, bounded := ctx.Deadline(); !bounded || reference.directoryURL != f.directoryURL || reference.domain != f.domain ||
		account == nil || account.key == nil || account.registration == nil || certificateKey == nil ||
		reference.accountURI != account.registration.URI || !f.validResourceURL(reference.accountURI, pebbleAccountPath) ||
		!f.validResourceURL(reference.orderURL, pebbleOrderPath) {
		return nil, ErrInput
	}
	client, err := f.httpClient(ctx, time.Second)
	if err != nil {
		return nil, err
	}
	// api.New mutates its supplied client's transport. Use a fresh client and
	// nonce/JWS state, retaining the original account URI and signing key.
	core, err := api.New(client, "tong-panel-local-reconciliation-test", reference.directoryURL, reference.accountURI, account.key)
	if err != nil {
		return nil, localRecoveryFailure(ctx)
	}
	// These are explicit fixture limits, not a production retry policy. Lego
	// may still finish an internal nonce/poll sleep before observing transport
	// cancellation; no goroutine is abandoned to pretend the call has stopped.
	for remaining := 32; remaining > 0; remaining-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		order, err := core.Orders.Get(reference.orderURL)
		if err != nil {
			return nil, localRecoveryFailure(ctx)
		}
		if len(order.Identifiers) != 1 || order.Identifiers[0].Type != "dns" || order.Identifiers[0].Value != reference.domain {
			return nil, errLocalOrderArtifact
		}
		switch order.Status {
		case acme.StatusValid:
			if !f.validResourceURL(order.Certificate, pebbleCertPath) {
				return nil, errLocalOrderArtifact
			}
			chain, _, err := core.Certificates.Get(order.Certificate, true)
			if err != nil {
				return nil, localRecoveryFailure(ctx)
			}
			if _, err := f.verifyCertificate(chain, certcrypto.PEMEncode(certificateKey)); err != nil {
				return nil, errLocalOrderArtifact
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return chain, nil
		case acme.StatusPending, acme.StatusReady, acme.StatusProcessing:
			// An unresolved order stays uncertain. There is no finalize/retry
			// side effect here, even if a server reports it as ready.
		default:
			return nil, errLocalOrderUncertain
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, errLocalOrderUncertain
}

func localRecoveryFailure(ctx context.Context) error {
	// Avoid exposing upstream response bodies or arbitrary error strings.
	if err := ctx.Err(); err != nil {
		return errors.Join(errLocalOrderUncertain, err)
	}
	return errLocalOrderUncertain
}

func TestPebbleLostFinalizeResponseReconcilesOriginalOrder(t *testing.T) {
	for _, mode := range []string{"attempt-canceled", "http-deadline"} {
		t.Run(mode, func(t *testing.T) {
			attempt, cancel := context.WithCancel(context.Background())
			defer cancel()
			fixture := newPebbleFixture(t, attempt, mode+".recovery.acme.test", pebbleFixtureOptions{holdFinalizeResponse: true})
			receipt := &localOrderReceipt{}
			client, account := fixture.newClient(t, attempt, 2*time.Second, receipt)
			// Preserve a distinct caller-generated certificate key. Lego's
			// default generated key is inaccessible after a lost finalize reply.
			certificateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			if account.key == certificateKey {
				t.Fatal("account and certificate keys are not distinct")
			}
			type obtainResult struct {
				resource *certificate.Resource
				err      error
			}
			result := make(chan obtainResult, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				resource, err := client.Certificate.Obtain(certificate.ObtainRequest{Domains: []string{fixture.domain}, PrivateKey: certificateKey, Bundle: true})
				result <- obtainResult{resource: resource, err: err}
			}()
			t.Cleanup(func() {
				// This cleanup runs before the fixture closes its listeners.
				// Join the client even after an assertion failure; cancellation
				// cannot skip lego's bounded internal sleeps by abandoning it.
				cancel()
				select {
				case <-finished:
				case <-time.After(25 * time.Second):
					t.Error("canceled lego call did not stop within its fixture bounds")
				}
			})
			select {
			case <-fixture.finalizeAccepted:
			case <-result:
				t.Fatal("issuance returned before the real CA accepted finalization")
			case <-time.After(15 * time.Second):
				t.Fatal("issuance did not reach the finalize-response barrier")
			}
			if mode == "attempt-canceled" {
				cancel()
			}
			var obtained obtainResult
			select {
			case obtained = <-result:
			case <-time.After(4 * time.Second):
				t.Fatal("withheld finalize response did not honor cancellation or HTTP timeout")
			}
			want := error(context.Canceled)
			if mode == "http-deadline" {
				want = context.DeadlineExceeded
			}
			if !errors.Is(obtained.err, want) || obtained.resource != nil {
				t.Fatal("interrupted finalize did not return the expected local error and no artifact")
			}
			cancel()
			select {
			case <-fixture.finalizeDisconnected:
			case <-time.After(2 * time.Second):
				t.Fatal("local server did not observe canceled finalize connection")
			}
			fixture.provider.Close()
			fixture.assertClosed(t)
			if fixture.provider.presented.Load() != 1 || fixture.provider.cleaned.Load() != 1 || fixture.challengeRequests.Load() == 0 {
				t.Fatal("finalization did not follow real validation and challenge cleanup")
			}
			reference := localOrderReference{directoryURL: fixture.directoryURL, accountURI: account.registration.URI, orderURL: receipt.snapshot(), domain: fixture.domain}
			if !fixture.validResourceURL(reference.orderURL, pebbleOrderPath) {
				t.Fatal("client did not retain the validated original order Location")
			}
			// A local error cannot establish absence of a remote certificate.
			// Independently recover the same order with a fresh bounded context
			// and the retained test account/key, without reviving the attempt.
			recovery, stopRecovery := context.WithTimeout(context.Background(), 5*time.Second)
			defer stopRecovery()
			chain, err := fixture.reconcileLocalOrder(recovery, reference, account, certificateKey)
			if err != nil || len(chain) == 0 {
				t.Fatal("accepted but unacknowledged finalization could not be reconciled")
			}
			if _, err := fixture.verifyCertificate(chain, certcrypto.PEMEncode(certificateKey)); err != nil {
				t.Fatal(err)
			}
			assertSingleLocalIssuance(t, fixture)
			fixture.assertClosed(t)
			if mode == "attempt-canceled" {
				testLocalRecoveryDenials(t, fixture, reference, account, certificateKey)
			}
		})
	}
}

func assertSingleLocalIssuance(t *testing.T, f *pebbleFixture) {
	t.Helper()
	if f.newAccounts.Load() != 1 || f.newOrders.Load() != 1 || f.finalizations.Load() != 1 {
		t.Fatal("recovery created another account/order or replayed finalization")
	}
}

func testLocalRecoveryDenials(t *testing.T, f *pebbleFixture, original localOrderReference, account *localAccount, certificateKey *ecdsa.PrivateKey) {
	t.Helper()
	for _, mode := range []string{"empty order", "foreign order", "HTTP order", "userinfo", "fragment", "query", "encoded path", "foreign directory", "changed account", "changed domain", "missing account", "missing certificate key", "nil context", "unbounded context", "canceled context", "expired context", "wrong account key", "wrong certificate key", "unknown order"} {
		t.Run(mode, func(t *testing.T) {
			reference := original
			owner, retainedKey := account, certificateKey
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			preflightDenial := true
			want := ErrInput
			switch mode {
			case "empty order":
				reference.orderURL = ""
			case "foreign order":
				reference.orderURL = "https://foreign.invalid/my-order/foreign"
			case "HTTP order":
				reference.orderURL = "http://" + f.origin.Host + pebbleOrderPath + "foreign"
			case "userinfo":
				reference.orderURL = "https://user@" + f.origin.Host + pebbleOrderPath + "foreign"
			case "fragment":
				reference.orderURL += "#foreign"
			case "query":
				reference.orderURL += "?foreign=1"
			case "encoded path":
				reference.orderURL = f.acmeServer.URL + pebbleOrderPath + "%2e%2e"
			case "foreign directory":
				reference.directoryURL = "https://foreign.invalid/directory"
			case "changed account":
				reference.accountURI = f.acmeServer.URL + pebbleAccountPath + "another"
			case "changed domain":
				reference.domain = "another.acme.test"
			case "missing account":
				owner = nil
			case "missing certificate key":
				retainedKey = nil
			case "nil context":
				ctx = nil
			case "unbounded context":
				ctx = context.Background()
			case "canceled context":
				cancel()
				want = context.Canceled
			case "expired context":
				var stop context.CancelFunc
				ctx, stop = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer stop()
				want = context.DeadlineExceeded
			case "wrong account key", "wrong certificate key":
				key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				preflightDenial = false
				if mode == "wrong account key" {
					owner = &localAccount{key: key, registration: account.registration}
					want = errLocalOrderUncertain
				} else {
					retainedKey = key
					want = errLocalOrderArtifact
				}
			case "unknown order":
				reference.orderURL = f.acmeServer.URL + pebbleOrderPath + "unknown"
				preflightDenial = false
				want = errLocalOrderUncertain
			}
			before := f.requests.Load()
			chain, err := f.reconcileLocalOrder(ctx, reference, owner, retainedKey)
			if !errors.Is(err, want) || len(chain) != 0 {
				t.Fatal("invalid recovery input returned an artifact or wrong status")
			}
			if preflightDenial && f.requests.Load() != before {
				t.Fatal("invalid recovery input caused network traffic")
			}
			assertSingleLocalIssuance(t, f)
		})
	}
}
