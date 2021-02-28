# freebox-exporter

Prometheus exporter for the [Freebox](https://www.free.fr/freebox/)

**Disclaimer**: I am not related to Iliad, Free or any of their subsidiaries. I have only created this Prometheus exporter to monitor my own device using some [publicly available documentation](https://dev.freebox.fr/sdk/os/).

## Install

Having a working Golang environment using Go modules:

```bash
go install github.com/trazfr/freebox-exporter@latest
```

## Use

This program is to be run in 2 steps, as you must authorize the exporter to access the Freebox. Once authorized, it may be run from anywhere.

```
Usage: freebox-exporter [options] <api_token_file>

api_token_file: file to store the token for the API

options:
  -debug
        enable the debug mode
  -hostDetails
        get details about the hosts connected to wifi and ethernet. This increases the number of metrics
  -httpDiscovery
        use http://mafreebox.freebox.fr/api_version to discover the Freebox at the first run (by default: use mDNS)
  -listen string
        listen to address (default ":9091")
```

### Step 1 authorize API

From the Freebox network, generate a token file for the API. The file `token.json` must not exist:

```bash
$ freebox-exporter token.json
Could not find the configuration file token.json
Freebox discovery: mDNS
1 Please accept the login on the Freebox Server
...
```

You must accept the API on the Freebox device.

Once done, the credentials will be stored in the new file `token.json`

**In case of errors**:

If you get the message `panic: Access is timeout`, you have to be faster to accept the access on the Freebox.

If you get the message `panic: MDNS timeout`, there may be a firewall preventing you to use mDNS. You may try to get the token using HTTP:

```bash
$ freebox-exporter -httpDiscovery token.json
Could not find the configuration file token.json
Freebox discovery: GET http://mafreebox.freebox.fr/api_version
1 Please accept the login on the Freebox Server
...
```

### Step 2 run

Once you have generated the token you may run from anywhere.

```bash
$ freebox-exporter token.json
Use configuration file token.json
Listen to :9091
```

Then you may test it:

```bash
$ curl 127.0.0.1:9091/metrics
# HELP freebox_connection_bandwith_bps available upload/download bandwidth in bit/s
# TYPE freebox_connection_bandwith_bps gauge
...
```