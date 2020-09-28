package checks

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func Test_Setup_MandatoryParams(t *testing.T) {
	_, err := NewHTTP(&HTTPConfig{})
	require.Error(t, err)
}

func Test_Setup_Nil(t *testing.T) {
	_, err := NewHTTP(nil)
	require.Error(t, err)
}

func Test_SetupDefaultValues(t *testing.T) {
	parse, _ := url.Parse("htttps://sparetimecoders.com")
	httpCheck, err := NewHTTP(&HTTPConfig{
		URL: parse,
	})
	require.NoError(t, err)
	require.Equal(t, 200, httpCheck.Config.StatusCode)
	require.Equal(t, defaultHTTPTimeout, httpCheck.Config.Timeout)
	require.NotNil(t, httpCheck.Config.Client)
	require.NotEqual(t, http.DefaultClient, httpCheck.Config.Client)
}

func Test_SetupOverrideDefaultValues(t *testing.T) {
	parse, _ := url.Parse("htttps://sparetimecoders.com")
	httpCheck, err := NewHTTP(&HTTPConfig{
		URL:        parse,
		StatusCode: 404,
		Client:     http.DefaultClient,
		Timeout:    100,
	})
	require.NoError(t, err)
	require.Equal(t, 404, httpCheck.Config.StatusCode)
	require.Equal(t, time.Duration(100), httpCheck.Config.Timeout)
	require.Equal(t, http.DefaultClient, httpCheck.Config.Client)
}

func Test_StatusCheck_Ok(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()
	parse, _ := url.Parse(server.URL)
	httpCheck, _ := NewHTTP(&HTTPConfig{
		URL: parse,
	})

	status, err := httpCheck.Status()
	require.NoError(t, err)
	require.NotEqual(t, 0, status)
}

func Test_StatusCheck_Failed(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(`Failed`))
	}))
	// Close the server when test finishes
	defer server.Close()
	parse, _ := url.Parse(server.URL)
	httpCheck, _ := NewHTTP(&HTTPConfig{
		URL: parse,
	})

	_, err := httpCheck.Status()
	require.Error(t, err)

}
