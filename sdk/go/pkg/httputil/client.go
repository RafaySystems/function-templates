package httputil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type retriableHTTPOptions struct {
	proxy                 func(req *http.Request) (*url.URL, error)
	dialTimeout           time.Duration
	keepAlive             time.Duration
	maxIdleConns          int
	idleConnTimeout       time.Duration
	tlsHandShakeTimeout   time.Duration
	expectContinueTimeout time.Duration
	tlsInsecureSkipVerify bool
	backoff               retryablehttp.Backoff
	checkRetry            retryablehttp.CheckRetry
	minRetryWait          time.Duration
	maxRetryWait          time.Duration
	maxRetryCount         int
	caCert                string
	dialer                func(ctx context.Context, network, addr string) (net.Conn, error)
}

func (o *retriableHTTPOptions) setDefaults() {
	if o.proxy == nil {
		o.proxy = http.ProxyFromEnvironment
	}
	if o.dialTimeout == 0 {
		o.dialTimeout = 30 * time.Second
	}
	if o.keepAlive == 0 {
		o.keepAlive = 120 * time.Second
	}
	if o.maxIdleConns == 0 {
		o.maxIdleConns = 20
	}
	if o.idleConnTimeout == 0 {
		o.idleConnTimeout = 90 * time.Second
	}
	if o.tlsHandShakeTimeout == 0 {
		o.tlsHandShakeTimeout = 10 * time.Second
	}
	if o.expectContinueTimeout == 0 {
		o.expectContinueTimeout = 1 * time.Second
	}
	if o.backoff == nil {
		o.backoff = retryablehttp.DefaultBackoff
	}
	if o.checkRetry == nil {
		o.checkRetry = RetryPolicy
	}
	if o.minRetryWait == 0 {
		o.minRetryWait = 1 * time.Nanosecond
	}
	if o.maxRetryWait == 0 {
		o.maxRetryWait = 30 * time.Nanosecond
	}
	if o.maxRetryCount < 0 {
		o.maxRetryCount = 0
	}
}

type RetriableHTTPOption func(*retriableHTTPOptions)

func WithProxy(proxy func(req *http.Request) (*url.URL, error)) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.proxy = proxy
	}
}

func WithDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.dialer = dialer
	}
}

func WithDialTimeout(timeout time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.dialTimeout = timeout
	}
}

func WithCaCert(caCert string) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.caCert = caCert
	}
}

func WithKeepAlive(timeout time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.keepAlive = timeout
	}
}

func WithMaxIdleConns(count int) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.maxIdleConns = count
	}
}

func WithIdleConnTimeout(timeout time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.idleConnTimeout = timeout
	}
}

func WithTLSHandShakeTimeout(timeout time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.tlsHandShakeTimeout = timeout
	}
}

func WithExpectContinueTimeout(timeout time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.expectContinueTimeout = timeout
	}
}

func WithTLSInsecureSkipVerify() RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.tlsInsecureSkipVerify = true
	}
}

func WithBackoff(backoff retryablehttp.Backoff) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.backoff = backoff
	}
}

func WithCheckRetry(checkRetry retryablehttp.CheckRetry) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.checkRetry = checkRetry
	}
}

func WithMinRetryWait(min time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.minRetryWait = min
	}
}

func WithMaxRetryWait(max time.Duration) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.maxRetryWait = max
	}
}

func WithMaxRetryCount(count int) RetriableHTTPOption {
	return func(rh *retriableHTTPOptions) {
		rh.maxRetryCount = count
	}
}

func NewRetriableHTTPClient(opts ...RetriableHTTPOption) *retryablehttp.Client {

	var options retriableHTTPOptions
	for _, opt := range opts {
		opt(&options)
	}

	options.setDefaults()

	if options.dialer == nil {
		options.dialer = (&net.Dialer{
			Timeout:   options.dialTimeout,
			KeepAlive: options.keepAlive,
		}).DialContext
	}
	transport := &http.Transport{
		Proxy:                 options.proxy,
		DialContext:           options.dialer,
		MaxIdleConns:          options.maxIdleConns,
		IdleConnTimeout:       options.idleConnTimeout,
		TLSHandshakeTimeout:   options.tlsHandShakeTimeout,
		ExpectContinueTimeout: options.expectContinueTimeout,
	}

	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: options.tlsInsecureSkipVerify,
	}

	if options.caCert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(options.caCert))
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	c := retryablehttp.Client{
		HTTPClient: &http.Client{
			Transport: transport,
		},
		Backoff:      options.backoff,
		CheckRetry:   options.checkRetry,
		RetryWaitMin: options.minRetryWait,
		RetryWaitMax: options.maxRetryWait,
		RetryMax:     options.maxRetryCount,
	}
	return &c
}

var (
	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)

	// A regular expression to match the error returned by net/http when the
	// TLS certificate is not trusted. This error isn't typed
	// specifically so we resort to matching on the error string.
	notTrustedErrorRe = regexp.MustCompile(`certificate is not trusted`)
)

func RetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	return baseRetryPolicy(resp, err)
}

func baseRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, v
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, v
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if notTrustedErrorRe.MatchString(v.Error()) {
				return false, v
			}
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, v
			}
		}

		// The error is likely recoverable so retry.
		return true, err
	}

	// not handling status code based errors
	return false, nil
}

var ErrRetry = errors.New("retry")

func NewErrRetry(reason string) error {
	return fmt.Errorf("%w: %s", ErrRetry, reason)
}

func IsErrRetry(err error) bool {
	return errors.Is(err, ErrRetry)
}
