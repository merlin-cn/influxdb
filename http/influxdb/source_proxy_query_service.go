package influxdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/csv"
	"github.com/influxdata/flux/lang"
	platform "github.com/influxdata/influxdb"
	platformhttp "github.com/influxdata/influxdb/http"
	"github.com/influxdata/influxdb/query"
	"github.com/influxdata/influxdb/query/influxql"
)

type SourceProxyQueryService struct {
	InsecureSkipVerify bool
	URL                string
	OrganizationID     platform.ID
	platform.SourceFields
	platform.V1SourceFields
}

func (s *SourceProxyQueryService) Query(ctx context.Context, w io.Writer, req *query.ProxyRequest) (flux.Statistics, error) {
	switch req.Request.Compiler.CompilerType() {
	case influxql.CompilerType:
		return s.influxQuery(ctx, w, req)
	case lang.FluxCompilerType:
		return s.fluxQuery(ctx, w, req)
	}

	return flux.Statistics{}, fmt.Errorf("compiler type not supported")
}

func (s *SourceProxyQueryService) fluxQuery(ctx context.Context, w io.Writer, req *query.ProxyRequest) (flux.Statistics, error) {
	request := struct {
		Spec    *flux.Spec   `json:"spec"`
		Query   string       `json:"query"`
		Type    string       `json:"type"`
		Dialect flux.Dialect `json:"dialect"`
	}{}

	switch c := req.Request.Compiler.(type) {
	case lang.FluxCompiler:
		request.Query = c.Query
		request.Type = lang.FluxCompilerType
	case lang.SpecCompiler:
		request.Spec = c.Spec
		request.Type = lang.SpecCompilerType
	default:
		return flux.Statistics{}, fmt.Errorf("compiler type not supported: %s", c.CompilerType())
	}

	request.Dialect = req.Dialect
	if request.Dialect == nil {
		request.Dialect = &csv.Dialect{
			ResultEncoderConfig: csv.ResultEncoderConfig{
				Annotations: nil,
				NoHeader:    false,
				Delimiter:   ',',
			},
		}
	}

	u, err := newURL(s.URL, "/api/v2/query")
	if err != nil {
		return flux.Statistics{}, err
	}

	qp := u.Query()
	qp.Set("organizationID", req.Request.OrganizationID.String())
	u.RawQuery = qp.Encode()

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(request); err != nil {
		return flux.Statistics{}, err
	}

	hreq, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return flux.Statistics{}, err
	}
	hreq.Header.Set("Authorization", s.Token)
	hreq.Header.Set("Content-Type", "application/json")
	hreq = hreq.WithContext(ctx)

	hc := newTraceClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(hreq)
	if err != nil {
		return flux.Statistics{}, err
	}
	defer resp.Body.Close()
	if err := platformhttp.CheckError(resp); err != nil {
		return flux.Statistics{}, err
	}

	var n int64
	var stats flux.Statistics
	defer func() {
		query.SetResponseBytes(&stats, n)
	}()

	n, err = io.Copy(w, resp.Body)
	if err != nil {
		return stats, err
	}

	data := []byte(resp.Trailer.Get(platformhttp.QueryStatsTrailer))
	if err := json.Unmarshal(data, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

func (s *SourceProxyQueryService) influxQuery(ctx context.Context, w io.Writer, req *query.ProxyRequest) (flux.Statistics, error) {
	if len(s.URL) == 0 {
		return flux.Statistics{}, fmt.Errorf("URL from source cannot be empty if the compiler type is influxql")
	}

	u, err := newURL(s.URL, "/query")
	if err != nil {
		return flux.Statistics{}, err
	}

	hreq, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return flux.Statistics{}, err
	}

	// TODO(fntlnz): configure authentication methods username/password and stuff
	hreq = hreq.WithContext(ctx)

	params := hreq.URL.Query()
	compiler, ok := req.Request.Compiler.(*influxql.Compiler)
	if !ok {
		return flux.Statistics{}, fmt.Errorf("passed compiler is not of type 'influxql'")
	}
	params.Set("q", compiler.Query)
	params.Set("db", compiler.DB)
	params.Set("rp", compiler.RP)

	hreq.URL.RawQuery = params.Encode()

	hc := newTraceClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(hreq)
	if err != nil {
		return flux.Statistics{}, err
	}
	defer resp.Body.Close()
	if err := platformhttp.CheckError(resp); err != nil {
		return flux.Statistics{}, err
	}

	res := &influxql.Response{}
	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		return flux.Statistics{}, err
	}

	csvDialect, ok := req.Dialect.(csv.Dialect)
	if !ok {
		return flux.Statistics{}, fmt.Errorf("unsupported dialect %T", req.Dialect)
	}

	var stats flux.Statistics
	var n int64
	defer func() {
		query.SetResponseBytes(&stats, n)
	}()

	stats.Metadata = make(flux.Metadata)
	n, err = csv.NewMultiResultEncoder(csvDialect.ResultEncoderConfig).Encode(w, influxql.NewResponseIterator(res))
	if err != nil {
		return stats, err
	}

	data := []byte(resp.Trailer.Get(platformhttp.QueryStatsTrailer))
	if err := json.Unmarshal(data, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}
