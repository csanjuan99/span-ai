package serve

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTheStartRoutesReadInPlace(t *testing.T) {
	for _, given := range []bool{true, false} {
		form := url.Values{"story": {"Refund twice"}}
		query := "/start/precheck"
		if given {
			form.Set("inPlace", "true")
			query += "?inPlace=true"
		}
		post := httptest.NewRequest("POST", "/spans", strings.NewReader(form.Encode()))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req, err := storyRequest(post)
		if err != nil {
			t.Fatal(err)
		}
		if check := checkInput(httptest.NewRequest("GET", query, nil), ""); req.InPlace != given || check.InPlace != given {
			t.Fatalf("given %v: request %v precheck %v", given, req.InPlace, check.InPlace)
		}
	}
}
