package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/clubpay/ronykit/rony"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriageEndpoint(t *testing.T) {
	base := startTriageServer(t)

	in := TicketInput{
		Customer: "Acme Corp",
		Plan:     "enterprise",
		Subject:  "Duplicate charge on March invoice",
		Message:  "We were billed twice for March. Please refund the duplicate today or we cancel.",
		Tags:     []string{"billing", "duplicate"},
	}

	status, raw := postJSON(t, base+"/triage", in)
	require.Equal(t, http.StatusOK, status, "body: %s", raw)

	var out TriageResponse
	require.NoError(t, json.Unmarshal(raw, &out))

	assert.Contains(t, out.RenderedState, "Customer: ACME CORP (Plan: enterprise)")
	assert.Contains(t, out.RenderedState, "Tags:     billing, duplicate")
	assert.Equal(t, DeptBilling, out.Decision.Department)
	assert.Equal(t, Urgency(3), out.Decision.Urgency)
	assert.True(t, out.Decision.Refund)
	assert.InDelta(t, 0.89, out.Decision.ChurnRisk, 1e-6)
	assert.InDelta(t, 0.92, out.Probabilities["department"]["billing"], 1e-6)
}

func TestTriageBatchEndpoint(t *testing.T) {
	base := startTriageServer(t)

	req := BatchTriageRequest{
		Tickets: []TicketInput{
			{
				Customer: "Beta LLC",
				Subject:  "Seat pricing",
				Message:  "Can we get enterprise pricing for 200 seats on our contract?",
			},
			{
				Customer: "Gamma Inc",
				Subject:  "Webhook failure",
				Message:  "Webhook deliveries are timing out in staging.",
			},
		},
	}

	status, raw := postJSON(t, base+"/triage/batch", req)
	require.Equal(t, http.StatusOK, status, "body: %s", raw)

	var out BatchTriageResponse
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Len(t, out.Results, 2)

	assert.Equal(t, DeptSales, out.Results[0].Decision.Department)
	assert.False(t, out.Results[0].Decision.Refund)
	assert.Equal(t, DeptPlatform, out.Results[1].Decision.Department)
	assert.Equal(t, Urgency(1), out.Results[1].Decision.Urgency)
}

func startTriageServer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	svc, err := NewTriageService(newDemoKeywordResolver())
	require.NoError(t, err)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	srv := rony.NewServer(rony.Listen(addr))
	svc.Setup(srv)
	require.NoError(t, srv.Start(ctx))
	t.Cleanup(func() { srv.Stop(ctx) })

	base := "http://" + addr
	require.Eventually(t, func() bool {
		st, _ := postJSON(t, base+"/triage/batch", BatchTriageRequest{})

		return st == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond)

	return base
}

func postJSON(t *testing.T, url string, body any) (int, []byte) {
	t.Helper()

	b, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()

	out, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return res.StatusCode, out
}
