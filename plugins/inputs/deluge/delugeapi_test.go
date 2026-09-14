package deluge_test

import (
	"net/http/httptest"
	"testing"

	"github.com/sazap10/deluge-telegraf-plugin/plugins/inputs/deluge"
)

// unreachableHost returns a host URL that is guaranteed to refuse connections.
func unreachableHost(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(nil)
	host := srv.URL
	srv.Close()

	return host
}

func TestGetAuthUnreachableHostReturnsError(t *testing.T) {
	api := &deluge.API{Host: unreachableHost(t), Password: "deluge"}

	err := api.GetAuth()
	if err == nil {
		t.Fatal("expected an error when the deluge host is unreachable, got nil")
	}
	if api.AuthToken != nil {
		t.Fatalf("expected no auth token, got %q", *api.AuthToken)
	}
}

func TestGetMetricsUnreachableHostReturnsError(t *testing.T) {
	api := &deluge.API{Host: unreachableHost(t), Password: "deluge"}

	result, err := api.GetMetrics()
	if err == nil {
		t.Fatal("expected an error when the deluge host is unreachable, got nil")
	}
	if result != nil {
		t.Fatalf("expected no result, got %+v", result)
	}
}
