package testutil

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

// NewAPI creates a Huma test API.
// Returns a humatest.TestAPI which embeds huma.API and exposes .Do /
// .Get / .Post helpers that issue requests and return *httptest.ResponseRecorder.
func NewAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("Test API", "0.0.0"))
	return api
}
