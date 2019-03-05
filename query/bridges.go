package query

import (
	"context"
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/csv"
	platform "github.com/influxdata/influxdb"
	"io"
)

const (
	ResponseBytesMetadata = "influxdb/response-bytes"
)

func GetResponseBytes(stats *flux.Statistics) int64 {
	if stats == nil {
		return 0
	}

	md := stats.Metadata
	if md == nil {
		return 0
	}

	if vs, ok := md[ResponseBytesMetadata]; ok && len(vs) > 0 {
		if v, ok := vs[0].(int64); ok {
			return v
		}
	}

	return 0
}

func SetResponseBytes(stats *flux.Statistics, bytes int64) {
	if stats == nil {
		return
	}

	if stats.Metadata == nil {
		stats.Metadata = make(flux.Metadata)
	}

	stats.Metadata[ResponseBytesMetadata] = []interface{}{bytes}
}

func StatisticsWithResponseBytes(bytes int64) flux.Statistics {
	stats := new(flux.Statistics)
	stats.Metadata = make(flux.Metadata)
	stats.Metadata[ResponseBytesMetadata] = []interface{}{bytes}
	return *stats
}

// QueryServiceBridge implements the QueryService interface while consuming the AsyncQueryService interface.
type QueryServiceBridge struct {
	AsyncQueryService AsyncQueryService
}

func (b QueryServiceBridge) Query(ctx context.Context, req *Request) (flux.ResultIterator, error) {
	query, err := b.AsyncQueryService.Query(ctx, req)
	if err != nil {
		return nil, err
	}
	return flux.NewResultIteratorFromQuery(query), nil
}

// ProxyQueryServiceBridge implements ProxyQueryService while consuming a QueryService interface.
type ProxyQueryServiceBridge struct {
	QueryService QueryService
}

func (b ProxyQueryServiceBridge) Query(ctx context.Context, w io.Writer, req *ProxyRequest) (flux.Statistics, error) {
	results, err := b.QueryService.Query(ctx, &req.Request)
	if err != nil {
		return flux.Statistics{}, err
	}
	defer results.Release()

	var stats flux.Statistics
	stats.Metadata = make(flux.Metadata)
	encoder := req.Dialect.Encoder()
	n, err := encoder.Encode(w, results)
	stats.Metadata[ResponseBytesMetadata] = []interface{}{n}
	if err != nil {
		return stats, err
	}

	stats = results.Statistics()
	if stats.Metadata == nil {
		stats.Metadata = make(flux.Metadata)
	}
	stats.Metadata[ResponseBytesMetadata] = []interface{}{n}
	return stats, nil
}

// QueryServiceProxyBridge implements QueryService while consuming a ProxyQueryService interface.
type QueryServiceProxyBridge struct {
	ProxyQueryService ProxyQueryService
}

func (b QueryServiceProxyBridge) Query(ctx context.Context, req *Request) (flux.ResultIterator, error) {
	d := csv.Dialect{ResultEncoderConfig: csv.DefaultEncoderConfig()}
	preq := &ProxyRequest{
		Request: *req,
		Dialect: d,
	}

	r, w := io.Pipe()
	statsChan := make(chan flux.Statistics, 1)

	go func() {
		stats, err := b.ProxyQueryService.Query(ctx, w, preq)
		defer func() {
			w.CloseWithError(err)
			statsChan <- stats
		}()
	}()

	dec := csv.NewMultiResultDecoder(csv.ResultDecoderConfig{})
	ri, err := dec.Decode(r)
	return asyncStatsResultIterator{
		ResultIterator: ri,
		stats:          statsChan,
	}, err
}

type asyncStatsResultIterator struct {
	flux.ResultIterator
	stats chan flux.Statistics
}

func (i asyncStatsResultIterator) Statistics() flux.Statistics {
	return <-i.stats
}

// REPLQuerier implements the repl.Querier interface while consuming a QueryService
type REPLQuerier struct {
	// Authorization is the authorization to provide for all requests
	Authorization *platform.Authorization
	// OrganizationID is the ID to provide for all requests
	OrganizationID platform.ID
	QueryService   QueryService
}

func (q *REPLQuerier) Query(ctx context.Context, compiler flux.Compiler) (flux.ResultIterator, error) {
	req := &Request{
		Authorization:  q.Authorization,
		OrganizationID: q.OrganizationID,
		Compiler:       compiler,
	}
	return q.QueryService.Query(ctx, req)
}
