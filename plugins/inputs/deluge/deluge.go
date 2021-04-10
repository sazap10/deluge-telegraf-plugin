package deluge

import (
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/pkg/errors"
)

// Deluge defines the data structure for plugin
type Deluge struct {
	Host     string `toml:"host"`
	Password string `toml:"password"`
	Log      telegraf.Logger
	API      *API
}

// Description returns the plugin description
func (d *Deluge) Description() string {
	return "A plugin to gather data from Deluge."
}

// SampleConfig returns the sample config for the plugin
func (d *Deluge) SampleConfig() string {
	return `
## Indicate if everything is fine
[inputs.deluge]
  ## Deluge WebUI url
	# host = ""
	## Deluge WebUI password
  # password = ""
`
}

// Init is for setup, and validating config.
func (d *Deluge) Init() error {
	API := &API{
		Host:     d.Host,
		Password: d.Password,
	}
	err := API.GetAuth()
	if err != nil {
		return err
	}
	d.API = API
	return nil
}

// Gather gathers and sends metrics to output
func (d *Deluge) Gather(acc telegraf.Accumulator) error {
	result, err := d.API.GetMetrics()
	if err != nil {
		_, ok := err.(*NoAuthError)
		if ok {
			// Unauthenticated, try authenticating and retry get metrics
			err = d.API.GetAuth()
			if err != nil {
				return err
			}
			result, err = d.API.GetMetrics()
		}
		if err != nil {
			return errors.Wrap(err, "unable to get metrics from deluge")
		}
	}

	d.gatherOverview(acc, result.Stats)
	for _, t := range result.Torrents {
		d.gatherTorrent(acc, t)
	}

	return nil
}

func (d *Deluge) gatherOverview(acc telegraf.Accumulator, overview Stats) {
	fields := map[string]interface{}{
		"max_download":           overview.MaxDownload,
		"max_upload":             overview.MaxUpload,
		"max_num_connections":    overview.MaxNumConnections,
		"num_connections":        overview.NumConnections,
		"upload_rate":            overview.UploadRate,
		"download_rate":          overview.DownloadRate,
		"download_protocol_rate": overview.DownloadProtocolRate,
		"upload_protocol_rate":   overview.UploadProtocolRate,
		"dht_nodes":              overview.DhtNodes,
	}
	acc.AddFields("deluge_overview", fields, nil)
}

func (d *Deluge) gatherTorrent(acc telegraf.Accumulator, torrent TorrentInfo) {
	fields := map[string]interface{}{
		"ratio":          torrent.Ratio,
		"total_size":     torrent.TotalSize,
		"total_uploaded": torrent.TotalUploaded,
		"progress":       torrent.Progress,
		"total_done":     torrent.TotalDone,
	}
	tags := map[string]string{
		"name":         torrent.Name,
		"state":        torrent.State,
		"tracker_host": torrent.TrackerHost,
	}
	acc.AddFields("deluge_torrent", fields, tags)
}

func init() {
	inputs.Add("deluge", func() telegraf.Input { return &Deluge{} })
}
