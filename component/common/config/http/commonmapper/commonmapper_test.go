package commonmapper

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/grafana/agent/component/common/config/http"
	"github.com/grafana/agent/pkg/flow/rivertypes"
	"github.com/prometheus/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_virtual_file_joins asserts that filepath.Join behaves as expected when
// combining two paths.
func Test_virtual_file_joins(t *testing.T) {
	tt := []struct {
		paths  []string
		expect string
	}{
		{
			paths:  []string{"root/", "relative/path"},
			expect: "root/relative/path",
		},
		{
			paths:  []string{"root/", "/absolute/path"},
			expect: "root/absolute/path",
		},
	}

	for _, tc := range tt {
		actual := filepath.Join(tc.paths...)
		assert.Equal(t, tc.expect, actual, "Unexpected result for joining %v", tc.paths)
	}
}

func TestMapper(t *testing.T) {
	t.Run("creates virtual mappings for TLS keys from strings", func(t *testing.T) {
		var mapper Mapper

		var (
			in = []http.ClientConfig{{
				TLSConfig: http.TLSConfig{
					CA:   rivertypes.OptionalSecret{Value: "ca_text"},
					Cert: rivertypes.OptionalSecret{Value: "cert_text"},
					Key:  rivertypes.Secret("key_text"),
				},
			}}

			out = mapper.Map(in...)
		)

		expect := []config.HTTPClientConfig{{
			TLSConfig: config.TLSConfig{
				CAFile:   "kvp/0/tls/ca",
				CertFile: "kvp/0/tls/cert",
				KeyFile:  "kvp/0/tls/key",
			},
		}}
		require.Equal(t, expect, out)
	})

	t.Run("can read virtual files", func(t *testing.T) {
		var mapper Mapper

		var (
			in = []http.ClientConfig{{
				TLSConfig: http.TLSConfig{
					CA:   rivertypes.OptionalSecret{Value: "ca_text"},
					Cert: rivertypes.OptionalSecret{Value: "cert_text"},
					Key:  rivertypes.Secret("key_text"),
				},
			}}

			out = mapper.Map(in...)
		)

		require.Len(t, out, 1)

		caBytes, err := fs.ReadFile(mapper.FS(), out[0].TLSConfig.CAFile)
		if assert.NoError(t, err) {
			require.Equal(t, "ca_text", string(caBytes))
		}

		certBytes, err := fs.ReadFile(mapper.FS(), out[0].TLSConfig.CertFile)
		if assert.NoError(t, err) {
			require.Equal(t, "cert_text", string(certBytes))
		}

		keyBytes, err := fs.ReadFile(mapper.FS(), out[0].TLSConfig.KeyFile)
		if assert.NoError(t, err) {
			require.Equal(t, "key_text", string(keyBytes))
		}
	})

	t.Run("creates virtual mappings for TLS keys from files", func(t *testing.T) {
		var mapper Mapper

		var (
			tempDir = t.TempDir()

			caFile   = makeTempFile(t, tempDir, "ca_text")
			certFile = makeTempFile(t, tempDir, "cert_text")
			keyFile  = makeTempFile(t, tempDir, "key_text")

			in = []http.ClientConfig{{
				TLSConfig: http.TLSConfig{
					CAFile:   caFile,
					CertFile: certFile,
					KeyFile:  keyFile,
				},
			}}

			out = mapper.Map(in...)
		)

		expect := []config.HTTPClientConfig{{
			TLSConfig: config.TLSConfig{
				CAFile:   path.Join("file/", caFile),
				CertFile: path.Join("file/", certFile),
				KeyFile:  path.Join("file/", keyFile),
			},
		}}
		require.Equal(t, expect, out)
	})

	t.Run("can read physical files", func(t *testing.T) {
		var mapper Mapper

		var (
			tempDir = t.TempDir()

			caFile   = makeTempFile(t, tempDir, "ca_text")
			certFile = makeTempFile(t, tempDir, "cert_text")
			keyFile  = makeTempFile(t, tempDir, "key_text")

			in = []http.ClientConfig{{
				TLSConfig: http.TLSConfig{
					CAFile:   caFile,
					CertFile: certFile,
					KeyFile:  keyFile,
				},
			}}

			out = mapper.Map(in...)
		)

		require.Len(t, out, 1)

		caBytes, err := fs.ReadFile(mapper.FS(), out[0].TLSConfig.CAFile)
		if assert.NoError(t, err) {
			require.Equal(t, "ca_text", string(caBytes))
		}

		certBytes, err := fs.ReadFile(mapper.FS(), out[0].TLSConfig.CertFile)
		if assert.NoError(t, err) {
			require.Equal(t, "cert_text", string(certBytes))
		}

		keyBytes, err := fs.ReadFile(mapper.FS(), out[0].TLSConfig.KeyFile)
		if assert.NoError(t, err) {
			require.Equal(t, "key_text", string(keyBytes))
		}
	})
}

func makeTempFile(t *testing.T, dir string, contents string) (filename string) {
	t.Helper()

	f, err := os.CreateTemp(dir, "*.txt")
	require.NoError(t, err)
	defer f.Close()

	_, err = fmt.Fprint(f, contents)
	require.NoError(t, err)

	return f.Name()
}
