package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/lula/src/types"
)

func TestCreateApiDomain(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec        *ApiSpec
		expectedErr bool
	}{
		"nil spec": {
			spec:        nil,
			expectedErr: true,
		},
		"empty requests": {
			spec: &ApiSpec{
				Requests: []Request{},
			},
			expectedErr: true,
		},
		"invalid request - no name": {
			spec: &ApiSpec{
				Requests: []Request{
					{
						URL: "test",
					},
				},
			},
			expectedErr: true,
		},
		"invalid request - no url": {
			spec: &ApiSpec{
				Requests: []Request{
					{
						Name: "test",
					},
				},
			},
			expectedErr: true,
		},
		"valid request": {
			spec: &ApiSpec{
				Requests: []Request{
					{
						Name: "test",
						URL:  "test",
					},
				},
			},
			expectedErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := CreateApiDomain(tt.spec)
			if (err != nil) != tt.expectedErr {
				t.Fatalf("CreateApiDomain() error = %v, wantErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestGetResources(t *testing.T) {
	respBytes := []byte(`{"healthcheck": "ok"}`)
	// unmarshal the response
	var resp map[string]interface{}
	err := json.Unmarshal(respBytes, &resp)
	require.NoError(t, err)

	apiReqName := "test"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
		} else if _, ok := r.URL.Query()["label"]; !ok {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(respBytes)
			require.NoError(t, err)
		}
	}))
	defer svr.Close()

	t.Run("pass", func(t *testing.T) {
		api, err := CreateApiDomain(&ApiSpec{
			Requests: []Request{
				{
					Name:   apiReqName,
					URL:    svr.URL,
					Params: map[string]string{"label": "test"},
					Options: &ApiOpts{
						Headers: map[string]string{"Accept": "application/json"},
					},
				},
			},
		})

		require.NoError(t, err)
		drs, err := api.GetResources(context.Background())
		require.NoError(t, err)

		want := types.DomainResources{
			apiReqName: types.DomainResources{
				// "raw" is `{"healthcheck": "ok"}`
				"raw": json.RawMessage{0x7b, 0x22, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x22, 0x3a, 0x20, 0x22, 0x6f, 0x6b, 0x22, 0x7d},
				"response": map[string]interface{}{
					"healthcheck": "ok",
				},
				"status": 200,
			}}
		require.Equal(t, want, drs)
	})

	t.Run("fail", func(t *testing.T) {
		api, err := CreateApiDomain(&ApiSpec{
			Requests: []Request{
				{
					Name: apiReqName,
					URL:  svr.URL,
				},
			},
		})

		require.NoError(t, err) // the spec is correct
		drs, err := api.GetResources(context.Background())
		require.Error(t, err)
		require.Equal(t, drs, types.DomainResources{
			apiReqName: types.DomainResources{
				"status":   400,
				"raw":      json.RawMessage{},
				"response": nil,
			},
		})
	})
}
