package dashboardd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardRouter(t *testing.T) {
	assert := assert.New(t)
	dashboard := Dashboardd{}
	router := httpRouter(&dashboard)

	testCases := []struct {
		path string
		want uint32
	}{
		{"/", http.StatusOK},
		{"/events", http.StatusOK},
		{"/entities", http.StatusOK},
		{"/index.html", http.StatusOK},
		{"/manifest.json", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("path %s", tc.path), func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			assert.Equal(http.StatusOK, res.Code)
			assert.NotEmpty(res.Body)
		})
	}
}
