package main

import (
	"testing"
	"time"
)

func TestBrokerSubscribeReceivesEvents(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe()
	defer unsub()

	event := SSEEvent{Type: "state", Data: []byte(`{"tracking":false}`)}
	b.Broadcast(event)

	select {
	case got := <-ch:
		if got.Type != "state" {
			t.Errorf("expected type 'state', got %s", got.Type)
		}
		if string(got.Data) != `{"tracking":false}` {
			t.Errorf("unexpected data: %s", string(got.Data))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBrokerMultipleClients(t *testing.T) {
	b := NewBroker()
	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	event := SSEEvent{Type: "tracking_started", Data: []byte(`{"taskId":"t1"}`)}
	b.Broadcast(event)

	for i, ch := range []chan SSEEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != "tracking_started" {
				t.Errorf("client %d: expected type 'tracking_started', got %s", i, got.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d: timed out", i)
		}
	}
}

func TestBrokerUnsubscribe(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe()

	if b.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", b.ClientCount())
	}

	unsub()

	if b.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after unsubscribe, got %d", b.ClientCount())
	}

	// Broadcast should not block or panic with no clients
	b.Broadcast(SSEEvent{Type: "state", Data: []byte(`{}`)})

	// Channel should not receive anything after unsubscribe
	select {
	case <-ch:
		t.Error("should not receive events after unsubscribe")
	default:
	}
}

func TestBrokerSlowClientDropsEvents(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the buffer (cap 16)
	for i := 0; i < 20; i++ {
		b.Broadcast(SSEEvent{Type: "state", Data: []byte(`{}`)})
	}

	// Should have exactly 16 (buffer capacity) events
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 16 {
		t.Errorf("expected 16 buffered events, got %d", count)
	}
}
