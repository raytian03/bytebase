package gitlab

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/internal/oauth"
)

func TestProvider_FetchUserInfo(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/users/1", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/users.html#single-user
							Body: io.NopCloser(strings.NewReader(`
{
  "id": 1,
  "username": "john_smith",
  "name": "John Smith",
  "state": "active",
  "avatar_url": "http://localhost:3000/uploads/user/avatar/1/cd8.jpeg",
  "web_url": "http://localhost:3000/john_smith",
  "created_at": "2012-05-23T08:00:58Z",
  "bio": "",
  "bot": false,
  "location": null,
  "public_email": "john@example.com",
  "skype": "",
  "linkedin": "",
  "twitter": "",
  "website_url": "",
  "organization": "",
  "job_title": "Operations Specialist",
  "followers": 1,
  "following": 1
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchUserInfo(ctx, common.OauthContext{}, "", "1")
	require.NoError(t, err)

	want := &vcs.UserInfo{
		PublicEmail: "john@example.com",
		Name:        "John Smith",
		State:       vcs.StateActive,
	}
	assert.Equal(t, want, got)
}

func TestOAuth_RefreshToken(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if token == "expired" {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body: io.NopCloser(strings.NewReader(`
					{"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
					`)),
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					// Example response taken from https://docs.gitlab.com/ee/api/oauth2.html#authorization-code-with-proof-key-for-code-exchange-pkce
					Body: io.NopCloser(strings.NewReader(`
{
 "access_token": "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54",
 "token_type": "bearer",
 "expires_in": 7200,
 "refresh_token": "8257e65c97202ed1726cf9571600918f3bffb2544b26e00a61df9897668c33a1",
 "created_at": 1607635748
}
`)),
				}, nil
			},
		},
	}
	token := "expired"

	calledRefresher := false
	refresher := func(_, _ string, _ int64) error {
		calledRefresher = true
		return nil
	}

	_, _, err := oauth.Get(
		ctx,
		client,
		"https://gitlab.example.com/api/v4/users/octocat",
		&token,
		tokenRefresher(
			"https://gitlab.example.com",
			oauthContext{},
			refresher,
		),
	)
	require.NoError(t, err)
	assert.Equal(t, "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54", token)
	assert.True(t, calledRefresher)
}
