package example

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/vkviyu/nexus/transport/client"
)

var GetBaidu = func() *client.Contract[string] {
	return &client.Contract[string]{
		URL: "https://www.baidu.com",
		BeforeRequest: func(req *http.Request) error {
			fmt.Printf("请求地址：%s\n", req.URL.Host)
			return nil
		},
		ParseResponse: func(r *http.Response) (*string, error) {
			fmt.Printf("status code: %d\n", r.StatusCode)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			str := string(body)
			return &str, nil
		},
	}
}

func TestGetBaidu(t *testing.T) {
	c := client.NewHTTPClient()
	r, err := client.Do(context.Background(), c, GetBaidu())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", *r)
}
