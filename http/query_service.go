package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/csv"
	"github.com/influxdata/influxdb/query"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// TODO(cwolff): This file can move to idpe because all this code is specific to queryd,
//   and is unused in OSS.

const (
	queryPath = "/api/v2/querysvc"

	queryStatisticsTrailer = "Influx-Query-Statistics"
)

type QueryHandler struct {
	*httprouter.Router

	Logger *zap.Logger

	csvDialect csv.Dialect

	ProxyQueryService query.ProxyQueryService
	CompilerMappings  flux.CompilerMappings
	DialectMappings   flux.DialectMappings
}

// NewQueryHandler returns a new instance of QueryHandler.
func NewQueryHandler() *QueryHandler {
	h := &QueryHandler{
		Router: NewRouter(),
		csvDialect: csv.Dialect{
			ResultEncoderConfig: csv.DefaultEncoderConfig(),
		},
	}

	h.HandlerFunc("GET", "/ping", h.handlePing)
	h.HandlerFunc("POST", queryPath, h.handlePostQuery)
	return h
}

// handlePing returns a simple response to let the client know the server is running.
func (h *QueryHandler) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

// handlePostQuery is the HTTP handler for the POST /api/v2/query route.
func (h *QueryHandler) handlePostQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req query.ProxyRequest
	req.WithCompilerMappings(h.CompilerMappings)
	req.WithDialectMappings(h.DialectMappings)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		EncodeError(ctx, err, w)
		return
	}

	_, err := h.ProxyQueryService.Query(ctx, w, &req)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	// TODO(cwolff): Add stats to the trailer here
}

// PrometheusCollectors satisifies the prom.PrometheusCollector interface.
func (h *QueryHandler) PrometheusCollectors() []prometheus.Collector {
	// TODO: gather and return relevant metrics.
	return nil
}

// QueryService implements query.ProxyQueryService and executes queries
// by communicating with queryd over http.
type QueryService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
}

// Ping checks to see if the server is responding to a ping request.
func (s *QueryService) Ping(ctx context.Context) error {
	u, err := newURL(s.Addr, "/ping")
	if err != nil {
		return err
	}

	hreq, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	SetToken(s.Token, hreq)
	hreq = hreq.WithContext(ctx)

	hc := newClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(hreq)
	if err != nil {
		return err
	}

	return CheckError(resp)
}

// Query calls the query route with the requested query and writes the result to the given writer.
// Returned are the number of bytes written (independent of any error returned).
func (s *QueryService) Query(ctx context.Context, w io.Writer, req *query.ProxyRequest) (int64, error) {
	u, err := newURL(s.Addr, queryPath)
	if err != nil {
		return 0, err
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(req); err != nil {
		return 0, err
	}

	hreq, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return 0, err
	}
	SetToken(s.Token, hreq)
	hreq = hreq.WithContext(ctx)

	hc := newClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(hreq)
	if err != nil {
		return 0, err
	}
	if err := CheckError(resp); err != nil {
		return 0, err
	}

	return io.Copy(w, resp.Body)
}

// statsResultIterator implements flux.ResultIterator and flux.Statisticser by reading the HTTP trailers.
type statsResultIterator struct {
	results    flux.ResultIterator
	resp       *http.Response
	statisitcs flux.Statistics
	err        error
}

func (s *statsResultIterator) More() bool {
	return s.results.More()
}

func (s *statsResultIterator) Next() flux.Result {
	return s.results.Next()
}

func (s *statsResultIterator) Release() {
	s.results.Release()
	s.readStats()
}

func (s *statsResultIterator) Err() error {
	err := s.results.Err()
	if err != nil {
		return err
	}
	return s.err
}

func (s *statsResultIterator) Statistics() flux.Statistics {
	return s.statisitcs
}

// readStats reads the query statisitcs off the response trailers.
func (s *statsResultIterator) readStats() {
	data := s.resp.Trailer.Get(queryStatisticsTrailer)
	if data != "" {
		s.err = json.Unmarshal([]byte(data), &s.statisitcs)
	}
}
