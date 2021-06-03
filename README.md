# Deluge Telegraf Plugin
A plugin for Telegraf to gather Deluge torrent client metrics from the web UI JSON API

## Usage
You can either:
* Use the pre-built docker image with telegraf available at [sazap10/deluge-telegraf-plugin](https://hub.docker.com/r/sazap10/deluge-telegraf-plugin). 
* Build the binary yourself and add to an existing telegraf deployment. See [building](#Building) for more information. Once built the binary can be mounted to the container as follows:
```
docker run -d --name=telegraf \
      -v $PWD/telegraf.conf:/etc/telegraf/telegraf.conf:ro \
      -v /path/to/local/deluge-telegraf-pluging:/path/to/container/deluge-telegraf-plugin \
      -v /path/to/local/plugin.conf:/path/to/container/plugin.conf \
      telegraf
```
Make sure the binary is built for the same architecture as the container and the configuration in `telegraf.conf` is pointing to the correct location of the binary and `plugin.conf`.

## Configuration
Example configuration `plugin.conf` which is required for the plugin
```
[[inputs.deluge]]
  host = "http://deluge:8112"
  password = "password"
```

Add the following to `telegraf.conf` to use the plugin:
```
[[inputs.execd]]
  command = ["/path/to/deluge-telegraf-plugin", "-config", "/path/to/plugin.conf"]
  signal = "none"
```

## Building
Run `make build` to generate a binary for `linux` running `amd64` architecture. You can customize the OS, architecture and any optional build arguments by passing the following:
* GOOS - Operating system to build binary for, defaults to `linux`. You can see a list of valid options [here](https://golang.org/doc/install/source#environment) or by running ` go tool dist list` 
* GOARCH - Architecture to build binary for, defaults to `amd64`. You can see a list of valid options [here](https://golang.org/doc/install/source#environment) or by running ` go tool dist list` 
* BUILD_ARGS - Any optional build arguments to provide to Go, eg to build for Raspberry PI which has ARM architecture run `GOOS=linux GOARCH=arm BUILD_ARGS="GOARM=5" make build`

## Development
For testing and developing locally you can use the `docker-compose.yml` to run influxdb, grafana and telegraf with the deluge plugin.

## Metrics
* deluge_overview
  * fields:
    * max_download (float)
    * max_upload (float)
    * max_num_connections (int)
    * num_connections (int)
    * upload_rate (float)
    * download_rate (float)
    * download_protocol_rate (float)
    * upload_protocol_rate (float)
    * dht_nodes (int)
* deluge_torrent
  * tags:
    * name - Name of the torrent
    * state - State of the torrent, possible values: Seeding, Downloading, Error
    * tracker_host - Host of the torrent tracker
  * fields:
    * ratio (float)
    * total_size (int)
    * total_uploaded (int)
    * progress (float)
    * total_done (int)

## Example output
```
deluge_overview upload_rate=0,upload_protocol_rate=42957,dht_nodes=367i,max_download=-1,num_connections=122i,download_rate=8183367,download_protocol_rate=9373.5,max_upload=-1,max_num_connections=200i 1618131905603734600
deluge_torrent,name=ubuntu-18.04.5-live-server-amd64.iso,state=Seeding,tracker_host=ubuntu.com ratio=0.000033068783523049206,total_size=990904320i,total_uploaded=32768i,progress=100,total_done=990904320i 1618131905603762600
deluge_torrent,name=ubuntu-20.10-live-server-amd64.iso,state=Downloading,tracker_host=ubuntu.com ratio=0,total_size=1046083584i,total_uploaded=0i,progress=18.961299896240234,total_done=198351911i 1618131905603766500
deluge_torrent,name=ubuntu-20.04.2-live-server-amd64.iso,state=Downloading,tracker_host=ubuntu.com ratio=-1,total_size=1215168512i,total_uploaded=0i,progress=0,total_done=0i 1618131905603901900
```