package knowledge

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Embedder is deliberately local-only. Implementations must never send text to
// a remote service; the HTTP implementation below accepts numeric loopback only.
type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type HTTPEmbedder struct {
	endpoint string
	client   *http.Client
}

func NewHTTPEmbedder(endpoint string) (*HTTPEmbedder, error) {
	u, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("embedding endpoint must be numeric HTTP loopback")
	}
	if net.ParseIP(u.Hostname()) == nil {
		return nil, fmt.Errorf("embedding endpoint must use a numeric loopback address")
	}
	dial := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, MaxConnsPerHost: 2, ResponseHeaderTimeout: 60 * time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
			return nil, fmt.Errorf("non-loopback embedding dial rejected")
		}
		return dial.DialContext(ctx, network, address)
	}}
	return &HTTPEmbedder{endpoint: strings.TrimRight(endpoint, "/"), client: &http.Client{Transport: transport, Timeout: 90 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("embedding redirect rejected") }}}, nil
}

func (h *HTTPEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 || len(texts) > 32 {
		return nil, fmt.Errorf("embedding batch must contain 1..32 texts")
	}
	for _, text := range texts {
		if text == "" || len(text) > 16384 {
			return nil, fmt.Errorf("embedding text is empty or too large")
		}
	}
	payload := map[string]any{"input": texts}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.endpoint+"/v1/embeddings", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("local embedding HTTP %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Data) != len(texts) {
		return nil, fmt.Errorf("embedding count %d does not match input count %d", len(out.Data), len(texts))
	}
	result := make([][]float32, len(texts))
	for _, item := range out.Data {
		if item.Index < 0 || item.Index >= len(result) || len(item.Embedding) == 0 {
			return nil, fmt.Errorf("invalid embedding index or dimension")
		}
		if result[item.Index] != nil {
			return nil, fmt.Errorf("duplicate embedding index")
		}
		result[item.Index] = item.Embedding
	}
	for _, vector := range result {
		if len(vector) == 0 {
			return nil, fmt.Errorf("missing embedding index")
		}
	}
	return result, nil
}

func encodeVector(vector []float32) ([]byte, error) {
	if len(vector) == 0 || len(vector) > 16384 {
		return nil, fmt.Errorf("invalid embedding dimension")
	}
	b := make([]byte, 4*len(vector))
	for n, value := range vector {
		binary.LittleEndian.PutUint32(b[n*4:], mathFloat32bits(value))
	}
	return b, nil
}

func decodeVector(b []byte) ([]float32, error) {
	if len(b) == 0 || len(b)%4 != 0 || len(b) > 16384*4 {
		return nil, fmt.Errorf("invalid stored embedding")
	}
	v := make([]float32, len(b)/4)
	for n := range v {
		v[n] = mathFloat32frombits(binary.LittleEndian.Uint32(b[n*4:]))
	}
	return v, nil
}

func mathFloat32bits(v float32) uint32     { return math.Float32bits(v) }
func mathFloat32frombits(v uint32) float32 { return math.Float32frombits(v) }
