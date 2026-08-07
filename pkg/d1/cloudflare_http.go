//go:build !(js && wasm)

// D1 over Cloudflare's HTTP API, registered as the same "d1" driver the Worker
// binding uses so that application code is identical on every target.
//
// D1's query endpoint takes one statement and its parameters and returns the
// rows as JSON objects. Those objects carry no column order — Go maps have
// none either — so the column list is recovered by scanning the raw JSON of
// the first row in the order the keys appear, which is the order D1 emits
// them. Without that, `SELECT id, name` would scan into the wrong variables
// depending on map iteration, which is the kind of bug that looks like data
// corruption.
package d1

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func init() {
	sql.Register("d1", &httpDriver{})
}

// checkConfig fails before the first query rather than inside it.
func checkConfig(binding string) error {
	if accountID() == "" {
		return missing("the Cloudflare account", "CLOUDFLARE_ACCOUNT_ID")
	}
	if apiToken() == "" {
		return missing("the API token", "CLOUDFLARE_API_TOKEN")
	}
	if databaseID(binding) == "" {
		return missing("the database id for binding "+binding,
			"D1_"+strings.ToUpper(binding)+"_ID", "D1_DATABASE_ID")
	}
	return nil
}

func accountID() string { return envOr("CLOUDFLARE_ACCOUNT_ID") }
func apiToken() string  { return envOr("CLOUDFLARE_API_TOKEN") }

// databaseID resolves the binding to a database. Per-binding first, so an app
// with more than one database can name them.
func databaseID(binding string) string {
	return envOr("D1_"+strings.ToUpper(binding)+"_ID", "D1_DATABASE_ID")
}

type httpDriver struct{}

var (
	_ driver.Driver            = (*httpDriver)(nil)
	_ driver.QueryerContext    = (*conn)(nil)
	_ driver.ExecerContext     = (*conn)(nil)
	_ driver.SessionResetter   = (*conn)(nil)
	_ driver.Rows              = (*rows)(nil)
	_ driver.StmtQueryContext  = (*stmt)(nil)
	_ driver.StmtExecContext   = (*stmt)(nil)
	_ driver.NamedValueChecker = (*conn)(nil)
)

func (d *httpDriver) Open(binding string) (driver.Conn, error) {
	if binding == "" {
		binding = Binding
	}
	if err := checkConfig(binding); err != nil {
		return nil, err
	}
	return &conn{
		endpoint: fmt.Sprintf(
			"https://api.cloudflare.com/client/v4/accounts/%s/d1/database/%s/query",
			accountID(), databaseID(binding)),
		token:  apiToken(),
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

type conn struct {
	endpoint string
	token    string
	client   *http.Client
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return &stmt{conn: c, query: query}, nil
}

func (c *conn) Close() error { return nil }

// Begin is not supported: the HTTP endpoint executes one statement per
// request, so there is no session to hold a transaction open across. Returning
// an error here is better than accepting BEGIN and silently not isolating
// anything.
func (c *conn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("d1: transactions are not available over the HTTP API — " +
		"use a batch of statements, or run inside a Worker where D1 exposes them")
}

func (c *conn) ResetSession(context.Context) error { return nil }

// CheckNamedValue widens what can be passed as a parameter. D1 takes JSON, so
// anything the encoder handles is fine, and rejecting types here would be
// stricter than the endpoint.
func (c *conn) CheckNamedValue(nv *driver.NamedValue) error { return nil }

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	res, err := c.do(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return newRows(res)
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	res, err := c.do(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return result{changes: res.Meta.Changes, lastID: res.Meta.LastRowID}, nil
}

// queryResult is one element of D1's "result" array.
type queryResult struct {
	Results []json.RawMessage `json:"results"`
	Success bool              `json:"success"`
	Meta    struct {
		Changes   int64 `json:"changes"`
		LastRowID int64 `json:"last_row_id"`
	} `json:"meta"`
}

func (c *conn) do(ctx context.Context, query string, args []driver.NamedValue) (*queryResult, error) {
	params := make([]any, len(args))
	for i, a := range args {
		params[i] = a.Value
	}
	body, err := json.Marshal(map[string]any{"sql": query, "params": params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("d1: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Result  []queryResult `json:"result"`
		Success bool          `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("d1: HTTP %d, and the body is not the expected JSON: %s",
			resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if !envelope.Success || len(envelope.Errors) > 0 {
		// Report Cloudflare's own message: a 401 here means the token, and a
		// 7xxx code means the SQL, and they need different fixes.
		if len(envelope.Errors) > 0 {
			return nil, fmt.Errorf("d1: %s (code %d)", envelope.Errors[0].Message, envelope.Errors[0].Code)
		}
		return nil, fmt.Errorf("d1: request failed with HTTP %d", resp.StatusCode)
	}
	if len(envelope.Result) == 0 {
		return &queryResult{}, nil
	}
	return &envelope.Result[0], nil
}

type stmt struct {
	conn  *conn
	query string
}

func (s *stmt) Close() error  { return nil }
func (s *stmt) NumInput() int { return -1 } // D1 counts them; let it

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), named(args))
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), named(args))
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.conn.ExecContext(ctx, s.query, args)
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.conn.QueryContext(ctx, s.query, args)
}

func named(args []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, len(args))
	for i, v := range args {
		out[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return out
}

type result struct {
	changes int64
	lastID  int64
}

func (r result) LastInsertId() (int64, error) { return r.lastID, nil }
func (r result) RowsAffected() (int64, error) { return r.changes, nil }

type rows struct {
	columns []string
	data    []json.RawMessage
	pos     int
}

func newRows(res *queryResult) (*rows, error) {
	r := &rows{data: res.Results}
	if len(res.Results) > 0 {
		cols, err := keyOrder(res.Results[0])
		if err != nil {
			return nil, err
		}
		r.columns = cols
	}
	return r, nil
}

// keyOrder returns an object's keys in the order they appear in the JSON,
// which is the order D1 selected them. encoding/json into a map would lose it.
func keyOrder(raw json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("d1: expected a row object, got %v", tok)
	}
	var keys []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("d1: expected a column name, got %v", keyTok)
		}
		keys = append(keys, key)
		// Skip the value, including nested objects and arrays.
		var discard json.RawMessage
		if err := dec.Decode(&discard); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func (r *rows) Columns() []string { return r.columns }
func (r *rows) Close() error      { return nil }

func (r *rows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	var row map[string]any
	if err := json.Unmarshal(r.data[r.pos], &row); err != nil {
		return err
	}
	r.pos++
	for i, col := range r.columns {
		if i >= len(dest) {
			break
		}
		dest[i] = convert(row[col])
	}
	return nil
}

// convert maps JSON's types onto the ones database/sql expects. Every JSON
// number decodes to float64, so an integer column would otherwise arrive as a
// float and fail to scan into an int.
func convert(v any) driver.Value {
	switch t := v.(type) {
	case nil:
		return nil
	case float64:
		if t == float64(int64(t)) {
			return int64(t)
		}
		return t
	case bool:
		return t
	case string:
		return t
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return nil
		}
		return string(b)
	}
}
