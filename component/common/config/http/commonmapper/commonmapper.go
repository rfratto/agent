// Package commonmapper exposes utilities to map between [http.ClientConfig]
// and [config.HTTPClientConfig] from the [github.com/prometheus/common/config]
// library.
package commonmapper

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/grafana/agent/component/common/config/http"
	"github.com/prometheus/common/config"
	"golang.org/x/exp/maps"
)

const (
	// kvpPrefix is used to prefix paths corresponding to stored key-value pairs.
	kvpPrefix = "kvp/"

	// filePrefix is used to prefix paths corresponding to recognized on-disk
	// files.
	filePrefix = "file/"
)

// Mapper manages mappings between [http.ClientConfig]s and
// [config.HTTPClientConfig]s.
//
// A Mapper is necessary as [http.ClientConfig] supports specifying TLS
// certificates from raw strings, where [config.HTTPClientConfig] requires TLS
// certificates to come from the local filesystem.
//
// Mapper works by managing an instance of [fs.FS] which manages virtual files
// corresponding to the set of source [http.ClientConfig]s.
//
// Mapper can only be used in conjunction with instances of [config.HTTPClient]
// where custom optioms can be passed to it which specify the [fs.FS] instance
// to use.
//
// The zero value of Mapper is ready for use.
//
// A Mapper must not be copied after it is created.
type Mapper struct {
	mut       sync.RWMutex
	kvps      map[string]string   // Recognized key-value pairs, for example, TLS keys.
	files     map[string]struct{} // Recognized file names.
	lastKVPs  map[string]string   // Key-value pairs initialized in the most recent call to Map.
	lastFiles map[string]struct{} // File names initialized in the most recent call to Map.
}

// FS returns the filesystem managed by the Mapper. The resulting filesystem
// will only recognize files to open based on calls to Map.
func (m *Mapper) FS() fs.FS { return mapperFilesystem{m} }

// Map maps the set of provided [http.ClientConfig] instances and returns a set
// of equivalent [config.HTTPClientConfig] instances. The virtual filesystem
// will be updated to recognize values specified in the list of configs.
//
// The virtual filesystem will continue to recognize paths from previous calls
// to Map until [Mapper.GC] is called.
func (m *Mapper) Map(configs ...http.ClientConfig) []config.HTTPClientConfig {
	m.init()

	m.mut.Lock()
	defer m.mut.Unlock()

	maps.Clear(m.lastKVPs)
	maps.Clear(m.lastFiles)

	res := make([]config.HTTPClientConfig, 0, len(configs))
	for i, inConfig := range configs {
		// keyPrefix is the prefix to use for stored kvps. It does *not* include
		// kvpPrefix, which is only used for disambiguating where a file comes
		// from.
		keyPrefix := strconv.Itoa(i)

		outConfig := config.HTTPClientConfig{
			BasicAuth:       m.mapBasicAuth(inConfig.BasicAuth),
			Authorization:   m.mapAuthorization(inConfig.Authorization),
			OAuth2:          m.mapOAuth2(path.Join(keyPrefix, "oauth2"), inConfig.OAuth2),
			BearerToken:     config.Secret(inConfig.BearerToken),
			BearerTokenFile: m.mapFile(inConfig.BearerTokenFile),
			ProxyURL:        config.URL(inConfig.ProxyURL),
			TLSConfig:       m.mapTLSConfig(path.Join(keyPrefix, "tls"), inConfig.TLSConfig),
			FollowRedirects: inConfig.FollowRedirects,
			EnableHTTP2:     inConfig.EnableHTTP2,
		}

		res = append(res, outConfig)
	}

	// Add entries in m.lastKVPs and m.lastFiles to m.kvps and m.files.
	for key, value := range m.lastKVPs {
		m.kvps[key] = value
	}
	for key, value := range m.lastFiles {
		m.files[key] = value
	}

	return res
}

func (m *Mapper) init() {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.kvps == nil {
		m.kvps = make(map[string]string)
	}
	if m.files == nil {
		m.files = make(map[string]struct{})
	}
	if m.lastKVPs == nil {
		m.lastKVPs = make(map[string]string)
	}
	if m.lastFiles == nil {
		m.lastFiles = make(map[string]struct{})
	}
}

// mapBasicAuth creates a mapping from [*http.BasicAuth] to
// [*config.BasicAuth].
//
// m.mut must be held when calling.
func (m *Mapper) mapBasicAuth(in *http.BasicAuth) *config.BasicAuth {
	if in == nil {
		return nil
	}

	return &config.BasicAuth{
		Username:     in.Username,
		Password:     config.Secret(in.Password),
		PasswordFile: m.mapFile(in.PasswordFile),
	}
}

// mapFile creates a mapping for an individual file. If the input file is
// empty, mapFile is a no-op.
//
// m.mut must be held when calling.
func (m *Mapper) mapFile(filename string) string {
	if filename == "" {
		return ""
	}

	// Make the filepath absolute since mapperFilesystem will have no way of
	// recognizing if paths used to be relative.
	if !filepath.IsAbs(filename) {
		// Making the path absolute is best-effort here; ignore errors.
		filename, _ = filepath.Abs(filename)
	}

	// Ensure that filenames are always using a slash character for compatibility
	// with fs.FS.
	filename = filepath.ToSlash(filename)
	m.lastFiles[filename] = struct{}{}

	return path.Join(filePrefix, filename)
}

// mapAuthorization creates a mapping from [*http.Authorization] to
// [*config.Authorization].
//
// m.mut must be held when calling.
func (m *Mapper) mapAuthorization(in *http.Authorization) *config.Authorization {
	if in == nil {
		return nil
	}

	return &config.Authorization{
		Type:            in.Type,
		Credentials:     config.Secret(in.Credentials),
		CredentialsFile: m.mapFile(in.CredentialsFile),
	}
}

// mapOAuth2 creates a mapping from [*http.OAuth2] to [*config.OAuth2].
//
// m.mut must be held when calling.
func (m *Mapper) mapOAuth2(kvpPrefix string, in *http.OAuth2) *config.OAuth2 {
	if in == nil {
		return nil
	}

	return &config.OAuth2{
		ClientID:         in.ClientID,
		ClientSecret:     config.Secret(in.ClientSecret),
		ClientSecretFile: m.mapFile(in.ClientSecretFile),
		Scopes:           in.Scopes,
		TokenURL:         in.TokenURL,
		EndpointParams:   in.EndpointParams,
		ProxyURL:         config.URL(in.ProxyURL),
		TLSConfig:        m.mapTLSConfig(path.Join(kvpPrefix, "tls"), in.TLSConfig),
	}
}

// mapTLSConfig creates a mapping from [http.TLSConfig] to [config.TLSConfig].
//
// m.mut must be held when calling.
func (m *Mapper) mapTLSConfig(kvpPrefix string, in http.TLSConfig) config.TLSConfig {
	var (
		virtualCAFile   string
		virtualCertFile string
		virtualKeyFile  string
	)

	if in.CAFile != "" {
		virtualCAFile = m.mapFile(in.CAFile)
	} else {
		virtualCAFile = m.mapKVP(path.Join(kvpPrefix, "ca"), in.CA.Value)
	}

	if in.CertFile != "" {
		virtualCertFile = m.mapFile(in.CertFile)
	} else {
		virtualCertFile = m.mapKVP(path.Join(kvpPrefix, "cert"), in.Cert.Value)
	}

	if in.KeyFile != "" {
		virtualKeyFile = m.mapFile(in.KeyFile)
	} else {
		virtualKeyFile = m.mapKVP(path.Join(kvpPrefix, "key"), string(in.Key))
	}

	return config.TLSConfig{
		CAFile:             virtualCAFile,
		CertFile:           virtualCertFile,
		KeyFile:            virtualKeyFile,
		ServerName:         in.ServerName,
		InsecureSkipVerify: in.InsecureSkipVerify,
		MinVersion:         config.TLSVersion(in.MinVersion),
		MaxVersion:         config.TLSVersion(in.MaxVersion),
	}
}

// mapKVP creates a mapping from a virtual key to a virtual file. m.mut must be
// held when calling.
func (m *Mapper) mapKVP(key, value string) string {
	m.lastKVPs[key] = value
	return path.Join(kvpPrefix, key)
}

// GC removes old entries from the virtual filesystem that are no longer needed
// based on the last call to ApplyConfigs. GC should only be invoked after
// previously mapped configs are no longer in use.
func (m *Mapper) GC() {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Remove any entries from m.kvps and m.files which are not in m.lastKVPs or m.lastFiles.
	for key := range m.kvps {
		if _, exist := m.lastKVPs[key]; exist {
			continue
		}
		delete(m.kvps, key)
	}

	for key := range m.files {
		if _, exist := m.lastFiles[key]; exist {
			continue
		}
		delete(m.files, key)
	}
}

type mapperFilesystem struct{ *Mapper }

var (
	_ fs.FS = mapperFilesystem{}
)

func (mfs mapperFilesystem) Open(name string) (fs.File, error) {
	switch {
	case strings.HasPrefix(name, kvpPrefix):
		return mfs.openKVP(strings.TrimPrefix(name, kvpPrefix))
	case strings.HasPrefix(name, filePrefix):
		return mfs.openFile(strings.TrimPrefix(name, filePrefix))
	default:
		return nil, fs.ErrNotExist
	}
}

func (mfs mapperFilesystem) openKVP(name string) (fs.File, error) {
	mfs.mut.RLock()
	defer mfs.mut.RUnlock()

	value, exists := mfs.kvps[name]
	if !exists {
		return nil, fs.ErrNotExist
	}

	return newStringFile(name, value), nil
}

func (mfs mapperFilesystem) openFile(name string) (fs.File, error) {
	// TODO(rfratto): Does this work with Windows drives?

	mfs.mut.RLock()
	defer mfs.mut.RUnlock()

	// Restore the file to be an absolute path. UNIX-style paths will never be
	// absolute at this stage, since the leading slash following filePrefix gets
	// trimmed.
	//
	// Windows paths may be absolute here, in the case of the path already
	// containing a drive.
	if !filepath.IsAbs(name) {
		name = "/" + name
	}

	// Even if the file exists on disk, we don't want to open it (for security
	// reasons) if there's no virtual mapping to it.
	if _, exists := mfs.files[name]; !exists {
		return nil, fs.ErrNotExist
	}

	return os.Open(name)
}
