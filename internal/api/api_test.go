package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/heptio/developer-dash/internal/cluster/fake"
	"github.com/heptio/developer-dash/internal/log"
	"github.com/heptio/developer-dash/internal/module"
	modulefake "github.com/heptio/developer-dash/internal/module/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_routes(t *testing.T) {
	cases := []struct {
		path                string
		method              string
		body                io.Reader
		expectedCode        int
		expectedContent     string
		expectedContentPath string
		expectedNamespace   string
	}{
		{
			path:         "/cluster-info",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			path:         "/namespaces",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			path:         "/navigation",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			path:            "/content/module/",
			method:          http.MethodGet,
			expectedCode:    http.StatusOK,
			expectedContent: "{\"title\":[{\"metadata\":{\"type\":\"text\"},\"config\":{\"value\":\"/\"}}],\"viewComponents\":null}\n",
		},
		{
			path:                "/content/module/namespace/another-namespace/",
			method:              http.MethodGet,
			expectedCode:        http.StatusOK,
			expectedContent:     "{\"title\":[{\"metadata\":{\"type\":\"text\"},\"config\":{\"value\":\"/\"}}],\"viewComponents\":null}\n",
			expectedNamespace:   "another-namespace",
			expectedContentPath: "/",
		},
		{
			path:                "/content/module/?namespace=fromquery",
			method:              http.MethodGet,
			expectedCode:        http.StatusOK,
			expectedContent:     "{\"title\":[{\"metadata\":{\"type\":\"text\"},\"config\":{\"value\":\"/\"}}],\"viewComponents\":null}\n",
			expectedNamespace:   "fromquery",
			expectedContentPath: "/",
		},
		{
			path:                "/content/module/namespace/path-takes-precedence/?namespace=fromquery",
			method:              http.MethodGet,
			expectedCode:        http.StatusOK,
			expectedContent:     "{\"title\":[{\"metadata\":{\"type\":\"text\"},\"config\":{\"value\":\"/\"}}],\"viewComponents\":null}\n",
			expectedNamespace:   "path-takes-precedence",
			expectedContentPath: "/",
		},
		{
			path:            "/content/module/nested",
			method:          http.MethodGet,
			expectedCode:    http.StatusOK,
			expectedContent: "{\"title\":[{\"metadata\":{\"type\":\"text\"},\"config\":{\"value\":\"/nested\"}}],\"viewComponents\":null}\n",
		},
		{
			path:                "/content/module/namespace/default/nested",
			method:              http.MethodGet,
			expectedCode:        http.StatusOK,
			expectedContent:     "{\"title\":[{\"metadata\":{\"type\":\"text\"},\"config\":{\"value\":\"/nested\"}}],\"viewComponents\":null}\n",
			expectedNamespace:   "default",
			expectedContentPath: "/nested",
		},
		{
			path:         "/missing",
			method:       http.MethodGet,
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("%s: %s", tc.method, tc.path)
		t.Run(name, func(t *testing.T) {
			m := modulefake.NewModule("module", log.NopLogger())

			manager := modulefake.NewStubManager("default", []module.Module{m})

			nsClient := fake.NewNamespaceClient([]string{"default"}, nil, "default")
			infoClient := fake.ClusterInfo{}
			srv := New("/", nsClient, infoClient, manager, log.NopLogger())

			err := srv.RegisterModule(m)
			require.NoError(t, err)

			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			u, err := url.Parse(ts.URL)
			require.NoError(t, err)

			// Add relative section to server url
			u, err = u.Parse(tc.path)
			require.NoError(t, err)

			req, err := http.NewRequest(tc.method, u.String(), tc.body)
			require.NoError(t, err)

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			defer res.Body.Close()

			data, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)

			if tc.expectedContent != "" {
				assert.Equal(t, tc.expectedContent, string(data))
			}
			assert.Equal(t, tc.expectedCode, res.StatusCode)

			if tc.expectedContentPath != "" {
				assert.Equal(t, tc.expectedContentPath, m.ObservedContentPath, "content path mismatch")
			}
			if tc.expectedNamespace != "" {
				assert.Equal(t, tc.expectedNamespace, m.ObservedNamespace, "namespace mismatch")
			}
		})
	}
}
