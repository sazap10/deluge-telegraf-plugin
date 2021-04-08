package deluge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type DelugeAPI struct {
	Host      string
	Password  string
	AuthToken *string
}

type BaseResponse struct {
	Error *Error `json:"error"`
	ID    int    `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type AuthResponse struct {
	Result bool `json:"result"`
	BaseResponse
}

type UpdateUIResponse struct {
	Result *UpdateUIResult `json:"result"`
	BaseResponse
}

type UpdateUIResult struct {
	Connected bool     `json:"connected"`
	Torrents  Torrents `json:"torrents"`
	Filters   Filters  `json:"filters"`
	Stats     Stats    `json:"stats"`
}
type Torrents map[string]TorrentInfo

type TorrentInfo struct {
	TimeAdded     int     `json:"time_added"`
	Name          string  `json:"name"`
	TotalDone     int     `json:"total_done"`
	Progress      float64 `json:"progress"`
	State         string  `json:"state"`
	Ratio         float64 `json:"ratio"`
	TotalSize     int     `json:"total_size"`
	TotalUploaded int64   `json:"total_uploaded"`
	TrackerHost   string  `json:"tracker_host"`
}

type Filters struct {
	State       []FilterInfo `json:"state"`
	TrackerHost []FilterInfo `json:"tracker_host"`
	Owner       []FilterInfo `json:"owner"`
	Label       []FilterInfo `json:"label"`
}

type FilterInfo struct {
	Key    string
	Number int
}

type Stats struct {
	MaxDownload            float64 `json:"max_download"`
	MaxUpload              float64 `json:"max_upload"`
	MaxNumConnections      int     `json:"max_num_connections"`
	NumConnections         int     `json:"num_connections"`
	UploadRate             float64 `json:"upload_rate"`
	DownloadRate           float64 `json:"download_rate"`
	DownloadProtocolRate   float64 `json:"download_protocol_rate"`
	UploadProtocolRate     float64 `json:"upload_protocol_rate"`
	DhtNodes               int     `json:"dht_nodes"`
	HasIncomingConnections int     `json:"has_incoming_connections"`
	FreeSpace              int64   `json:"free_space"`
	ExternalIP             string  `json:"external_ip"`
}

type BaseRequest struct {
	Method string `json:"method"`
	ID     int    `json:"id"`
}

type UpdateUIRequest struct {
	Params UpdateUIRequestParams `json:"params"`
	BaseRequest
}

type UpdateUIRequestParams struct {
	Keys    []string
	Filters *map[string]string
}

type AuthRequest struct {
	Params []string `json:"params"`
	BaseRequest
}

type NoAuthError struct{}

func (e *NoAuthError) Error() string {
	return "authentication failed, please check that you are authenticated"
}

func (f *FilterInfo) UnmarshalJSON(p []byte) error {
	var tmp []json.RawMessage

	if err := json.Unmarshal(p, &tmp); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[0], &f.Key); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[1], &f.Number); err != nil {
		return err
	}

	return nil
}

func (p *UpdateUIRequestParams) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{p.Keys, p.Filters})
}

func (d *DelugeAPI) makeRequest(host string, request interface{}) (*http.Response, error) {
	httpClient := &http.Client{}
	u, _ := url.Parse(fmt.Sprintf("%s/json", host))

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if d.AuthToken != nil {
		c := http.Cookie{
			Name:  "_session_id",
			Value: *d.AuthToken,
		}
		req.Header.Set("Cookie", c.String())
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		// If the error type is *url.Error, sanitize its URL before returning.
		if e, ok := err.(*url.Error); ok {
			if url, urlErr := url.Parse(e.URL); urlErr == nil {
				e.URL = url.String()
				return nil, e
			}
		}

		return nil, err
	}

	return resp, nil
}

func (d *DelugeAPI) GetAuth() error {
	authRequest := &AuthRequest{
		Params: []string{d.Password},
		BaseRequest: BaseRequest{
			Method: "auth.login",
			ID:     1,
		},
	}
	resp, err := d.makeRequest(d.Host, authRequest)
	if err != nil {
		return errors.Wrap(err, "unable to authenticate with deluge, please check config")
	}
	var authReponse AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authReponse)
	if err != nil {
		return errors.Wrap(err, "unable to decode response from body")
	}
	if !authReponse.Result {
		return errors.New("authentication unsuccessful, please check that the configured password is correct")
	}

	for _, c := range resp.Cookies() {
		if c.Name == "_session_id" {
			d.AuthToken = &c.Value
			return nil
		}
	}

	return errors.New("unable to retrieve auth token")
}

func (d *DelugeAPI) GetMetrics() (*UpdateUIResult, error) {
	updateUIRequest := &UpdateUIRequest{
		Params: UpdateUIRequestParams{
			Keys: []string{
				"name",
				"total_size",
				"state",
				"progress",
				"ratio",
				"time_added",
				"tracker_host",
				"total_done",
				"total_uploaded",
			},
			Filters: &map[string]string{},
		},
		BaseRequest: BaseRequest{Method: "web.update_ui", ID: 1},
	}
	resp, err := d.makeRequest(d.Host, updateUIRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error making web.update_ui request to deluge")
	}
	var updateUIResponse UpdateUIResponse
	err = json.NewDecoder(resp.Body).Decode(&updateUIResponse)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode response from body")
	}

	if updateUIResponse.Error != nil {
		switch updateUIResponse.Error.Code {
		// Unauthenticated
		case 1:
			return nil, &NoAuthError{}
		default:
			return nil, errors.Errorf("error whilst getting metrics, code: %d, message: %s", updateUIResponse.Error.Code, updateUIResponse.Error.Message)
		}
	}

	return updateUIResponse.Result, nil
}
