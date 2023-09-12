package deluge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// API defines the structure for the API
type API struct {
	Host      string
	Password  string
	AuthToken *string
}

// BaseResponse defines the base response from the API
type BaseResponse struct {
	Error *Error `json:"error"`
	ID    int    `json:"id"`
}

// Error defines the structure for the error returned from API
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// AuthResponse defines the structure for auth.login response
type AuthResponse struct {
	Result bool `json:"result"`
	BaseResponse
}

// UpdateUIResponse defines the structure for web.update_ui response
type UpdateUIResponse struct {
	Result *UpdateUIResult `json:"result"`
	BaseResponse
}

// UpdateUIResult defines the structure for web.update_ui result in the response
type UpdateUIResult struct {
	Connected bool     `json:"connected"`
	Torrents  Torrents `json:"torrents"`
	Filters   Filters  `json:"filters"`
	Stats     Stats    `json:"stats"`
}

// Torrents defines the structure for torrents info map in the result
type Torrents map[string]TorrentInfo

// TorrentInfo defines the structure for torrent info in the result
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

// Filters defines the structure for filters in the result
type Filters struct {
	State       []FilterInfo `json:"state"`
	TrackerHost []FilterInfo `json:"tracker_host"`
	Owner       []FilterInfo `json:"owner"`
	Label       []FilterInfo `json:"label"`
}

// FilterInfo defines the structure for a filter info in the result
type FilterInfo struct {
	Key    string
	Number int
}

// Stats defines the structure for the overall stats in the result
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

// BaseRequest defines the structure for the base request to the API
type BaseRequest struct {
	Method string `json:"method"`
	ID     int    `json:"id"`
}

// UpdateUIRequest defines the structure for the web.update_ui request to the API
type UpdateUIRequest struct {
	Params UpdateUIRequestParams `json:"params"`
	BaseRequest
}

// UpdateUIRequestParams defines the structure for the web.update_ui request params
type UpdateUIRequestParams struct {
	Keys    []string
	Filters *map[string]string
}

// AuthRequest defines the structure for the auth.login request to the API
type AuthRequest struct {
	Params []string `json:"params"`
	BaseRequest
}

// NoAuthError defines the structure for the Unauthenticated error
type NoAuthError struct{}

// Error returns the Unauthenticated error
func (e *NoAuthError) Error() string {
	return "authentication failed, please check that you are authenticated"
}

// UnmarshalJSON unmarshalls a FilterInfo object
func (f *FilterInfo) UnmarshalJSON(p []byte) error {
	var tmp []json.RawMessage

	if err := json.Unmarshal(p, &tmp); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[0], &f.Key); err != nil {
		return err
	}

	return json.Unmarshal(tmp[1], &f.Number)
}

// MarshalJSON marshalls a UpdateUIRequestParams object to json
func (p *UpdateUIRequestParams) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{p.Keys, p.Filters})
}

func (d *API) makeRequest(host string, request interface{}) (*http.Response, error) {
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

// GetAuth makes auth.login request to Deluge and sets auth to API
func (d *API) GetAuth() error {
	authRequest := &AuthRequest{
		Params: []string{d.Password},
		BaseRequest: BaseRequest{
			Method: "auth.login",
			ID:     1,
		},
	}
	resp, err := d.makeRequest(d.Host, authRequest)
	defer func() {
		bodyErr := resp.Body.Close()
		if bodyErr != nil {
			log.Println(bodyErr)
		}
	}()
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
		c := c
		if c.Name == "_session_id" {
			d.AuthToken = &c.Value
			return nil
		}
	}

	return errors.New("unable to retrieve auth token")
}

// GetMetrics makes web.update_ui request to Deluge and returns the result
func (d *API) GetMetrics() (*UpdateUIResult, error) {
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
	defer func() {
		bodyErr := resp.Body.Close()
		if bodyErr != nil {
			log.Println(bodyErr)
		}
	}()
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
