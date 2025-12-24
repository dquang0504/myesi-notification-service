package kafka

import "testing"

func TestParseEvent_StrictPayload(t *testing.T) {
	raw := []byte(`{"type":"payment.success","organization_id":1,"payload":{"a":1},"severity":"high"}`)
	evt, err := parseEvent(raw)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if evt.EventType != "payment.success" {
		t.Fatalf("expected type payment.success got %s", evt.EventType)
	}
	if evt.OrganizationID != 1 {
		t.Fatalf("expected org 1 got %d", evt.OrganizationID)
	}
	if evt.Payload == nil {
		t.Fatalf("expected payload not nil")
	}
}

func TestParseEvent_LoosePayload_Backfill(t *testing.T) {
	raw := []byte(`{"event_type":"sbom.scan.completed","organization_id":2,"severity":"medium","foo":"bar"}`)
	evt, err := parseEvent(raw)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if evt.EventType != "sbom.scan.completed" {
		t.Fatalf("expected sbom.scan.completed got %s", evt.EventType)
	}
	if evt.OrganizationID != 2 {
		t.Fatalf("expected org 2 got %d", evt.OrganizationID)
	}
	if evt.Severity != "medium" {
		t.Fatalf("expected medium got %s", evt.Severity)
	}
	if evt.Payload["foo"] != "bar" {
		t.Fatalf("expected foo=bar got %v", evt.Payload["foo"])
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	raw := []byte(`{bad`)
	_, err := parseEvent(raw)
	if err == nil {
		t.Fatalf("expected error")
	}
}
