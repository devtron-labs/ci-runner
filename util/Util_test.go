package util

import (
	"net/url"
	"testing"
)

func TestParseUrl(t *testing.T) {
	type args struct {
		rawURL string
	}
	tests := []struct {
		name    string
		args    args
		wantUrl *url.URL
		wantErr bool
	}{
		{
			name:    "without scheme",
			args:    args{rawURL: "devtron.ai"},
			wantUrl: &url.URL{Host: "devtron.ai"},
			wantErr: false,
		},
		{
			name:    "with http",
			args:    args{rawURL: "http://devtron.ai"},
			wantUrl: &url.URL{Host: "devtron.ai", Scheme: "http"},
			wantErr: false,
		},
		{
			name:    "with https",
			args:    args{rawURL: "https://devtron.ai"},
			wantUrl: &url.URL{Host: "devtron.ai", Scheme: "https"},
			wantErr: false,
		},
		{
			name:    "with path",
			args:    args{rawURL: "https://devtron.ai/test-path"},
			wantUrl: &url.URL{Host: "devtron.ai", Scheme: "https", Path: "/test-path"},
			wantErr: false,
		},
		{
			name:    "with path without scheme",
			args:    args{rawURL: "devtron.ai/test-path"},
			wantUrl: &url.URL{Host: "devtron.ai", Path: "/test-path"},
			wantErr: false,
		},
		{
			name:    "with path and quesry without scheme",
			args:    args{rawURL: "devtron.ai/test-path?abc=test"},
			wantUrl: &url.URL{Host: "devtron.ai", Path: "/test-path", RawQuery: "abc=test"},
			wantErr: false,
		},
		{
			name:    "test error",
			args:    args{rawURL: "skdjncje938u4983(**&^^%$$#@!"},
			wantUrl: &url.URL{Host: "devtron.ai", Path: "/test-path", RawQuery: "abc=test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUrl, err := ParseUrl(tt.args.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !UrlEqual(gotUrl, tt.wantUrl) {
				t.Errorf("ParseUrl() gotUrl = %v, want %v", gotUrl, tt.wantUrl)
			}
		})
	}
}

func UrlEqual(got, want *url.URL) bool {
	return got.Host == want.Host &&
		got.Path == want.Path &&
		got.Scheme == want.Scheme &&
		got.RawQuery == want.RawQuery

}
