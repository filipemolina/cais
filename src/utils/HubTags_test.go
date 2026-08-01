package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const nginxTagsFixtureNoVersionMatch = `{"count":1283,"next":null,"previous":null,"results":[
{"name":"stable-alpine3.24-perl","images":[{"architecture":"amd64"},{"architecture":"arm64"}]},
{"name":"stable-alpine-perl","images":[{"architecture":"amd64"},{"architecture":"arm64"}]},
{"name":"1.31.3-alpine3.24-perl","images":[{"architecture":"amd64"}]}
]}`

const redisTagsFixtureWithVersionMatch = `{"count":500,"next":null,"previous":null,"results":[
{"name":"8.10-alpine","images":[{"architecture":"amd64"}]},
{"name":"8.10.0-alpine","images":[{"architecture":"amd64"}]},
{"name":"8.10.0","images":[{"architecture":"amd64"},{"architecture":"arm64"}]}
]}`

func TestListTagsAndBestDefaultTag(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		repo    string
		want    string
	}{
		{"a repo whose recent tags are all compound falls back to latest - the real nginx case", nginxTagsFixtureNoVersionMatch, "nginx", "latest"},
		{"a repo with a bare version tag among recent ones picks it", redisTagsFixtureWithVersionMatch, "redis", "8.10.0"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(c.fixture))
			}))
			defer server.Close()

			old := hubBaseURL
			hubBaseURL = server.URL
			defer func() { hubBaseURL = old }()

			tags, err := ListTags(c.repo, 50)
			if err != nil {
				t.Fatalf("ListTags: %v", err)
			}
			if got := BestDefaultTag(tags); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestListTagsHandlesA429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	old := hubBaseURL
	hubBaseURL = server.URL
	defer func() { hubBaseURL = old }()

	if _, err := ListTags("nginx", 50); err == nil {
		t.Fatal("expected an error on 429, got nil")
	}
}
