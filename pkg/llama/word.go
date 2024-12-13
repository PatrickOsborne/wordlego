package llama

import (
	"fmt"
	ollama "github.com/ollama/ollama/api"
	"net/http"
	"net/url"
	"time"
)

func CreateSimpleClient() (*ollama.Client, error) {
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		return client, fmt.Errorf("failed to create ollama client from environment (%w)", err)
	}

	return client, nil
}

func CreateClient() (*ollama.Client, error) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	httpClient := &http.Client{
		Transport: t,
		Timeout:   60 * time.Second,
	}

	llamaUrl, err := url.Parse("http://localhost:11434")
	if err != nil {
		return nil, fmt.Errorf("failed to parse url (%s)", err)
	}

	client := ollama.NewClient(llamaUrl, httpClient)
	return client, nil
}
