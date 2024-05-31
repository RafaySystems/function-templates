package sdk

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/RafaySystems/function-templates/sdk/go/pkg/httputil"
	"golang.org/x/sync/errgroup"
)

type writer struct {
	sync.Mutex

	url           string
	token         string
	logger        *slog.Logger
	ctx           context.Context
	flushTickRate time.Duration
	client        *http.Client
	skipTLSVerify bool
	writeTimeout  time.Duration

	buf []byte

	stop chan struct{}
}

type WriterOption func(*writer)

var WithWriteFlushTickRate = func(tickRate time.Duration) WriterOption {
	return func(w *writer) {
		w.flushTickRate = tickRate
	}
}

var WithSkipTLSVerify = func(skipTLSVerify bool) WriterOption {
	return func(w *writer) {
		w.skipTLSVerify = skipTLSVerify
	}
}

var WithLogWriteTimeout = func(writeTimeout time.Duration) WriterOption {
	return func(w *writer) {
		w.writeTimeout = writeTimeout
	}
}

func NewActivityLogWriter(ctx context.Context, logger *slog.Logger, url, token string, opts ...WriterOption) io.WriteCloser {
	w := &writer{
		ctx:    ctx,
		logger: logger,
		url:    url,
		token:  token,
		buf:    []byte{},
		stop:   make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(w)
	}

	var httpopts []httputil.RetriableHTTPOption
	if w.skipTLSVerify {
		httpopts = append(httpopts, httputil.WithTLSInsecureSkipVerify())
	}

	w.client = httputil.NewRetriableHTTPClient(httpopts...).StandardClient()

	go w.startUpload()
	return w
}

func (w *writer) Close() error {
	close(w.stop)
	return w.flush()
}

func (w *writer) Write(b []byte) (n int, err error) {
	w.Lock()
	w.buf = append(w.buf, b...)
	w.Unlock()

	n = len(b)
	return
}

func (w *writer) startUpload() {
	ticker := time.NewTicker(w.flushTickRate)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			err := w.flush()
			if err != nil {
				w.logger.Error("error flushing writer", "error", err)
			}
		}
	}
}

func (w *writer) flush() error {
	if w.isEmpty() {
		return nil
	}

	w.logger.Debug("flushing writer", "buf", string(w.buf))

	w.Lock()
	reader := bytes.NewBuffer(append(w.buf[0:0:0], w.buf...))
	w.buf = w.buf[0:0:0]
	w.Unlock()

	if reader.Len() > 0 {
		piper, pipew := io.Pipe()
		writer := multipart.NewWriter(pipew)
		defer func() {
			writer.Close()
			pipew.Close()
			piper.Close()
		}()

		ctx, cancel := context.WithTimeout(w.ctx, w.writeTimeout)
		defer cancel()
		group, gctx := errgroup.WithContext(ctx)

		path := w.url + "?append=true"
		req, err := http.NewRequestWithContext(gctx, "POST", path, piper)
		if err != nil {
			w.logger.Error("error creating request", "error", err, "path", path)
			return err
		}

		req.Header.Add(WorkflowTokenHeader, w.token)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		group.Go(func() error {
			defer pipew.Close()
			defer writer.Close()

			fw, err := writer.CreateFormFile("content", "stdout")
			if err != nil {
				w.logger.Error("error creating form file", "error", err)
				return err
			}
			_, err = io.Copy(fw, reader)
			if err != nil {
				w.logger.Error("error copying to writer", "error", err)
				return err
			}
			return nil
		})

		group.Go(func() error {
			resp, err := w.client.Do(req)
			if err != nil {
				w.logger.Error("error sending request", "error", err, "path", path)
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				w.logger.Error("error response", "status", resp.Status)
				return errors.New("error: update activity log failed")
			}
			return nil
		})

		err = group.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *writer) isEmpty() bool {
	w.Lock()
	defer w.Unlock()
	return len(w.buf) == 0
}
