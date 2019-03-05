package query

import (
	"context"
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/csv"
	platform "github.com/influxdata/influxdb"
	"io"
)

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
	stats.Metadata["influxdb/response-bytes"] = []interface{}{n}
	if err != nil {
		return stats, err
	}

	stats = results.Statistics()
	if stats.Metadata == nil {
		stats.Metadata = make(flux.Metadata)
	}
	stats.Metadata["influxdb/response-bytes"] = []interface{}{n}
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

	go func() {
		_, err := b.ProxyQueryService.Query(ctx, w, preq)
		// propagate the stats here (need to wrap the result iterator)
		// Make the Release method attach the stats
		w.CloseWithError(err)
	}()

	dec := csv.NewMultiResultDecoder(csv.ResultDecoderConfig{})
	return dec.Decode(r)
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
