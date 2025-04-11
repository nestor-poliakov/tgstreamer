package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"tgstreamer/internal/app"
	"time"
)

const host = "sponsor.ajay.app"

type SponsorBlockClient struct {
	client *http.Client
}

func NewSponsorBlock() *SponsorBlockClient {
	sb := &SponsorBlockClient{
		client: &http.Client{
			Timeout: time.Minute,
		},
	}
	return sb
}

func (s *SponsorBlockClient) GetSegments(ctx context.Context, videoCode string) (segments []app.Segment, err error) {
	req := http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   host,
			Path:   "/api/skipSegments",
			RawQuery: url.Values{
				"videoID": []string{videoCode},
			}.Encode(),
		},
	}
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("do http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code not ok (%s)", resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(&segments)
	if err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}
	return segments, nil
}
