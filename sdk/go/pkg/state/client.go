package stateclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	"github.com/RafaySystems/function-templates/sdk/go/pkg/httputil"
	"github.com/pkg/errors"
)

type StateScope struct {
	OrganizationID string  `json:"organization_id"`
	ProjectID      *string `json:"project_id,omitempty"`
	EnvironmentID  *string `json:"environment_id,omitempty"`
}

type StateScopeBuilder interface {
	WithOrgScope() ScopedState
	WithProjectScope() ScopedState
	WithEnvScope() ScopedState
	WithCustomScope(scope StateScope) ScopedState
}

type boundStateBuilder struct {
	httpClient     *http.Client
	baseURL        string
	token          string
	organizationID string
	projectID      string
	environmentID  string
}

func NewBoundState(req sdk.Request) StateScopeBuilder {
	return &boundStateBuilder{
		httpClient:     httputil.NewRetriableHTTPClient().StandardClient(),
		baseURL:        req.MetaString("stateStoreUrl"),
		token:          req.MetaString("stateStoreToken"),
		organizationID: req.MetaString("organizationID"),
		projectID:      req.MetaString("projectID"),
		environmentID:  req.MetaString("environmentID"),
	}
}

func (b *boundStateBuilder) WithOrgScope() ScopedState {
	return b.WithCustomScope(StateScope{OrganizationID: b.organizationID})
}

func (b *boundStateBuilder) WithProjectScope() ScopedState {
	return b.WithCustomScope(StateScope{
		OrganizationID: b.organizationID,
		ProjectID:      &b.projectID,
	})
}

func (b *boundStateBuilder) WithEnvScope() ScopedState {
	return b.WithCustomScope(StateScope{
		OrganizationID: b.organizationID,
		ProjectID:      &b.projectID,
		EnvironmentID:  &b.environmentID,
	})
}

func (b *boundStateBuilder) WithCustomScope(scope StateScope) ScopedState {
	return &boundState{
		httpClient:     b.httpClient,
		baseURL:        b.baseURL,
		token:          b.token,
		organizationID: b.organizationID,
		projectID:      b.projectID,
		environmentID:  b.environmentID,
		scope:          scope,
	}
}

type ScopedState interface {
	Get(ctx context.Context, key string) (json.RawMessage, uint32, error)
	Set(ctx context.Context, key string, updateFn func(old json.RawMessage) (json.RawMessage, error)) error
	SetKV(ctx context.Context, key string, value json.RawMessage, version uint32) error
	Delete(ctx context.Context, key string) error
}

type boundState struct {
	httpClient     *http.Client
	baseURL        string
	token          string
	organizationID string
	projectID      string
	environmentID  string
	scope          StateScope
}

type stateResponse struct {
	Value   json.RawMessage `json:"value"`
	Version uint32          `json:"version"`
}

func (b *boundState) injectHeaders(req *http.Request) {
	req.Header.Set("X-Eaas-State-Token", b.token)
	req.Header.Set("X-Organization-ID", b.organizationID)
	req.Header.Set("X-Project-ID", b.projectID)
	req.Header.Set("X-Environment-ID", b.environmentID)
	req.Header.Set("Content-Type", "application/json")
}

func (b *boundState) Get(ctx context.Context, key string) (json.RawMessage, uint32, error) {
	var zero json.RawMessage
	value, version, err := b.getRaw(ctx, key)
	if err != nil {
		return zero, 0, sdk.NewErrNotFound(fmt.Sprintf("key not found: %s", key))
	}
	return value, version, nil
}

// Developer passes key and value to set
func (b *boundState) SetKV(ctx context.Context, key string, value json.RawMessage, version uint32) error {

	// Send update with version for OCC
	body := map[string]any{
		"scope":   b.scope,
		"key":     key,
		"value":   value,
		"version": version,
	}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, b.baseURL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	b.injectHeaders(req)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return sdk.NewErrConflict("version conflict on set")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return sdk.NewErrFailed(fmt.Sprintf("set failed: %s", string(body)))
	}

	// Success
	return nil
}

// Set abstracts create/update with OCC retry logic
// Developer passes a transformation function that takes the old value and returns the new one.
func (b *boundState) Set(ctx context.Context, key string, updateFn func(old json.RawMessage) (json.RawMessage, error)) error {
	for retries := 0; retries < 5; retries++ {
		// Step 1: Get latest raw value and version
		raw, version, err := b.getRaw(ctx, key)
		if err != nil && !sdk.IsErrNotFound(err) {
			return err
		}

		var oldVal json.RawMessage
		if err == nil && raw != nil {
			if err := json.Unmarshal(raw, &oldVal); err != nil {
				return err
			}
		}

		// Step 2: Apply developer function
		newVal, err := updateFn(oldVal)
		if err != nil {
			return err
		}

		// Step 3: Send update with version for OCC
		body := map[string]any{
			"scope":   b.scope,
			"key":     key,
			"value":   newVal,
			"version": version,
		}
		buf, _ := json.Marshal(body)

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, b.baseURL, bytes.NewReader(buf))
		if err != nil {
			return err
		}
		b.injectHeaders(req)

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusConflict {
			// OCC failure — retry
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return sdk.NewErrFailed(fmt.Sprintf("set failed: %s", string(body)))
		}

		// Success
		return nil
	}

	return sdk.NewErrTransient("set failed after max retries due to version conflicts")
}

// Helper for internal GetRaw
func (b *boundState) getRaw(ctx context.Context, key string) (json.RawMessage, uint32, error) {
	q := url.Values{}
	q.Set("organization_id", b.scope.OrganizationID)
	q.Set("key", key)
	if b.scope.ProjectID != nil {
		q.Set("project_id", *b.scope.ProjectID)
	}
	if b.scope.EnvironmentID != nil {
		q.Set("environment_id", *b.scope.EnvironmentID)
	}

	fullURL := fmt.Sprintf("%s?%s", b.baseURL, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, 0, err
	}
	b.injectHeaders(req)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, 0, sdk.NewErrNotFound("key not found in state store")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, sdk.NewErrNotFound(fmt.Sprintf("get kv state failed: %s", string(body)))
	}

	var result stateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}

	return result.Value, result.Version, nil
}

func (b *boundState) Delete(ctx context.Context, key string) error {

	body := map[string]any{
		"scope": b.scope,
		"key":   key,
	}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, b.baseURL, bytes.NewReader(buf))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	b.injectHeaders(req)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "http error")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return sdk.NewErrFailed(fmt.Sprintf("delete failed: %s", string(body)))
	}

	return nil
}
