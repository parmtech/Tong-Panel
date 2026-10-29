package sites

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"
)

func TestNetTXTLookupUsesConfiguredDNSResolver(t *testing.T) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	serverErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 1500)
		n, addr, err := server.ReadFrom(buf)
		if err != nil {
			serverErr <- err
			return
		}
		response, err := txtDNSResponse(buf[:n], "dns-proof-from-loopback")
		if err == nil {
			_, err = server.WriteTo(response, addr)
		}
		serverErr <- err
	}()

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "udp", server.LocalAddr().String())
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	values, err := (NetTXTLookup{Resolver: resolver}).LookupTXT(ctx, "_tong-panel-challenge.example.test.")
	if err != nil {
		t.Fatalf("LookupTXT() error = %v", err)
	}
	if len(values) != 1 || values[0] != "dns-proof-from-loopback" {
		t.Fatalf("LookupTXT() = %#v, want loopback TXT value", values)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("loopback DNS responder error = %v", err)
	}
}

func TestNetTXTLookupHonorsContextCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			select {
			case started <- struct{}{}:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := (NetTXTLookup{Resolver: resolver}).LookupTXT(ctx, "_tong-panel-challenge.example.test.")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("LookupTXT() error = %v, want context deadline", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("resolver did not use the configured cancellable dialer")
	}
}

func TestDomainVerificationRejectsWrongLoopbackDNSProof(t *testing.T) {
	catalog := newVerificationCatalog(t)
	instruction, err := catalog.BeginDomainVerification(context.Background(), "tenant-a", "site-a", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	serverErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 1500)
		n, addr, err := server.ReadFrom(buf)
		if err == nil {
			var response []byte
			response, err = txtDNSResponse(buf[:n], "attacker-controlled-value")
			if err == nil {
				_, err = server.WriteTo(response, addr)
			}
		}
		serverErr <- err
	}()
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "udp", server.LocalAddr().String())
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := catalog.CompleteDomainVerification(ctx, "tenant-a", "site-a", "example.com", instruction.Value, NetTXTLookup{Resolver: resolver}); !errors.Is(err, ErrDomainProofMismatch) {
		t.Fatalf("wrong loopback TXT proof error = %v, want ErrDomainProofMismatch", err)
	}
	record, err := catalog.Get(context.Background(), "tenant-a", "site-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.Domains[0].OwnershipState != "pending" {
		t.Fatalf("wrong DNS proof changed ownership state to %q", record.Domains[0].OwnershipState)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("loopback DNS responder error = %v", err)
	}
}

func txtDNSResponse(query []byte, value string) ([]byte, error) {
	if len(query) < 12 || binary.BigEndian.Uint16(query[4:6]) != 1 {
		return nil, errors.New("expected one DNS question")
	}
	questionEnd, err := dnsQuestionEnd(query, 12)
	if err != nil || questionEnd+4 > len(query) {
		return nil, errors.New("malformed DNS question")
	}
	if binary.BigEndian.Uint16(query[questionEnd:questionEnd+2]) != 16 || binary.BigEndian.Uint16(query[questionEnd+2:questionEnd+4]) != 1 {
		return nil, errors.New("expected TXT/IN DNS question")
	}
	if len(value) > 255 {
		return nil, errors.New("TXT test value exceeds one character-string")
	}

	question := query[12 : questionEnd+4]
	response := make([]byte, 12+len(question)+2+2+2+4+2+1+len(value))
	copy(response[:2], query[:2])
	binary.BigEndian.PutUint16(response[2:4], 0x8180) // response, recursion desired/available, no error
	binary.BigEndian.PutUint16(response[4:6], 1)
	binary.BigEndian.PutUint16(response[6:8], 1)
	copy(response[12:], question)
	answer := 12 + len(question)
	response[answer], response[answer+1] = 0xc0, 0x0c // owner name points to question name
	binary.BigEndian.PutUint16(response[answer+2:answer+4], 16)
	binary.BigEndian.PutUint16(response[answer+4:answer+6], 1)
	binary.BigEndian.PutUint32(response[answer+6:answer+10], 60)
	binary.BigEndian.PutUint16(response[answer+10:answer+12], uint16(1+len(value)))
	response[answer+12] = byte(len(value))
	copy(response[answer+13:], value)
	return response, nil
}

func dnsQuestionEnd(message []byte, offset int) (int, error) {
	for {
		if offset >= len(message) {
			return 0, errors.New("DNS name is truncated")
		}
		labelLen := int(message[offset])
		switch {
		case labelLen == 0:
			return offset + 1, nil
		case labelLen&0xc0 == 0xc0:
			if offset+1 >= len(message) {
				return 0, errors.New("DNS compression pointer is truncated")
			}
			return offset + 2, nil
		case labelLen&0xc0 != 0 || offset+1+labelLen > len(message):
			return 0, errors.New("invalid or truncated DNS label")
		default:
			offset += labelLen + 1
		}
	}
}
