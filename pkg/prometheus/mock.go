package prometheus

import (
	"bytes"
	"encoding/json"
	"github.com/fusakla/promruval/pkg/config"
	"github.com/prometheus/common/model"
	"io/ioutil"
	"net/http"
	"time"
)

func NewQueryVectorResponseMock(seriesCount int) interface{} {
	d := make([]model.Sample, seriesCount)
	for i := range d {
		d[i] = model.Sample{
			Metric:    model.Metric{},
			Value:     0,
			Timestamp: 0,
		}
	}
	s := struct {
		ResultType string         `json:"resultType"`
		Value      []model.Sample `json:"result"`
	}{
		ResultType: "vector",
		Value:      d,
	}
	return s
}

func NewSeriesResponseMock(seriesCount int) interface{} {
	d := make([]map[string]string, seriesCount)
	for i := range d {
		d[i] = map[string]string{"foo": "bar"}
	}
	return d
}

type responseMock struct {
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorType string      `json:"errorType,omitempty"`
	Error     string      `json:"error,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
}

type mockV1Client struct {
	warning  bool
	error    bool
	data     interface{}
	duration time.Duration
}

func (m *mockV1Client) newResponseMock() *http.Response {
	status := 200
	resp := responseMock{
		Status: "success",
		Data:   m.data,
	}
	if m.warning {
		resp.Warnings = []string{"mocked warning"}
	}
	if m.error {
		status = 500
		resp.Status = "error"
		resp.ErrorType = "execution"
		resp.Error = "mocked error"
	}
	js, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	return &http.Response{StatusCode: status, Body: ioutil.NopCloser(bytes.NewReader(js))}
}

func (m mockV1Client) RoundTrip(_ *http.Request) (*http.Response, error) {
	time.Sleep(m.duration)
	return m.newResponseMock(), nil
}

func NewClientMock(data interface{}, duration time.Duration, warning bool, error bool) *Client {
	cli, err := NewClientWithRoundTripper(config.PrometheusConfig{}, mockV1Client{
		warning:  warning,
		error:    error,
		data:     data,
		duration: duration,
	})
	if err != nil {
		panic(err)
	}
	return cli
}
