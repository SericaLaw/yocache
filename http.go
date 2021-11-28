package yocache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"yocache/consistenthash"
)

const (
	defaultBasePath = "/_yocache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker and http.Handler for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string
	// opts specifies the options.
	opts HTTPPoolOptions

	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string, opts HTTPPoolOptions) *HTTPPool {
	if opts.BasePath == "" {
		opts.BasePath = defaultBasePath
	}
	if opts.Replicas == 0 {
		opts.Replicas = defaultReplicas
	}
	if opts.Codec == nil {
		opts.Codec = RawCodec{}
	}

	return &HTTPPool{
		self: self,
		opts: opts,
	}
}

// HTTPPoolOptions are the configurations of a HTTPPool.
type HTTPPoolOptions struct {
	// BasePath specifies the HTTP path that will serve groupcache requests.
	// If blank, it defaults to "/_yocache/".
	BasePath string

	// Replicas specifies the number of key replicas on the consistent hash.
	// If blank, it defaults to 50.
	Replicas int

	// HashFn specifies the hash function of the consistent hash.
	// If blank, it defaults to crc32.ChecksumIEEE.
	HashFn consistenthash.Hash

	// Codec specifies the encoding and decoding mechanisms for internal HTTP connections.
	// If blank, it defaults to RawCodec.
	Codec Codec
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests from remote peers for fetching keys.
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.opts.BasePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.opts.BasePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the value to the response body as a proto message.
	body, err := p.opts.Codec.Encode(view)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// Set updates the pool's list of peers.
// Each value in peers should be a valid base URL,
// for example "http://example.net:8000".
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(p.opts.Replicas, p.opts.HashFn)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.opts.BasePath,
			decoder: DecoderFunc(p.opts.Codec.Decode),
		}
	}
}

// PickPeer picks the peer associated with the key, return nil if the peer is self.
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// httpGetter implements PeerGetter.
type httpGetter struct {
	baseURL string  // peer's base url
	decoder Decoder
}

// Get gets cache value from a peer using internal encoding/decoding mechanism through HTTP.
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	view, err := h.decoder.Decode(body)
	if err != nil {
		return nil, fmt.Errorf("decoding response body: %v", err)
	}

	return view.ByteSlice(), nil
}

var _ PeerGetter = (*httpGetter)(nil)
