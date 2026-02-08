package kit

import (
	"encoding/json"
	"testing"
)

type carrierMsg struct {
	Value string `json:"value"`
}

func TestEnvelopeCarrierMarshalUnmarshal(t *testing.T) {
	msg := carrierMsg{Value: "ok"}
	data := marshalEnvelopeCarrier(msg)

	var out carrierMsg
	unmarshalEnvelopeCarrier(data, &out)
	if out.Value != "ok" {
		t.Fatalf("unexpected unmarshal value: %s", out.Value)
	}

	raw := RawMessage("raw")
	if got := string(marshalEnvelopeCarrier(raw)); got != "raw" {
		t.Fatalf("unexpected raw payload: %s", got)
	}
}

func TestEnvelopeCarrierFillWithEnvelope(t *testing.T) {
	ctx := NewContext(nil)
	conn := newTestConn()
	conn.kv["client"] = "client-1"
	ctx.SetConn(conn)
	ctx.setRoute("route")
	ctx.setServiceName("svc")
	ctx.setContractID("cid")

	env := ctx.Out().
		SetID("env-1").
		SetMsg(&carrierMsg{Value: "out"}).
		SetHdr("hdr", "value")

	carrier := newEnvelopeCarrier(outgoingCarrier, "session", "origin", "target").
		FillWithEnvelope(env)

	if carrier.Data.EnvelopeID != "env-1" {
		t.Fatalf("unexpected envelope id: %s", carrier.Data.EnvelopeID)
	}
	if carrier.Data.Route != "route" {
		t.Fatalf("unexpected route: %s", carrier.Data.Route)
	}
	if carrier.Data.ServiceName != "svc" {
		t.Fatalf("unexpected service name: %s", carrier.Data.ServiceName)
	}
	if carrier.Data.ContractID != "cid" {
		t.Fatalf("unexpected contract id: %s", carrier.Data.ContractID)
	}
	if carrier.Data.ConnHdr["client"] != "client-1" {
		t.Fatalf("unexpected conn header: %s", carrier.Data.ConnHdr["client"])
	}
	if carrier.Data.Hdr["hdr"] != "value" {
		t.Fatalf("unexpected envelope header: %s", carrier.Data.Hdr["hdr"])
	}

	var decoded carrierMsg
	unmarshalEnvelopeCarrier(carrier.Data.Msg, &decoded)
	if decoded.Value != "out" {
		t.Fatalf("unexpected carrier payload: %s", decoded.Value)
	}
}

func TestEnvelopeCarrierFromJSON(t *testing.T) {
	original := &envelopeCarrier{
		SessionID: "sid",
		Kind:      incomingCarrier,
		OriginID:  "origin",
		TargetID:  "target",
		Data: &carrierData{
			EnvelopeID: "env",
			ConnHdr:    map[string]string{"k": "v"},
		},
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded envelopeCarrier
	if err := decoded.FromJSON(payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.SessionID != "sid" || decoded.Data.EnvelopeID != "env" {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestCarrierDataTraceCarrier(t *testing.T) {
	c := carrierData{ConnHdr: map[string]string{}}
	c.Set("k", "v")
	if got := c.Get("k"); got != "v" {
		t.Fatalf("unexpected carrier value: %s", got)
	}
}
