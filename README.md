# eglogs

`eglogs` streams Envoy Gateway access logs from Kubernetes and formats them for
easier reading. It can filter logs by regular expression, HTTP status, and
request duration, or emit the original JSON log lines.

Inspired by my bash alias and implemented in Go using AI.

## Requirements

By default, `eglogs` uses the current kubeconfig context, reads the `envoy`
container in the `envoy-gateway-system` namespace, and selects pods belonging
to a gateway named `main`. It streams all available log history before following
new entries; use `-tail` to limit the initial history.

## Install

```sh
git clone https://github.com/towolf/eglogs.git
cd eglogs
go build -o eglogs main.go
```

## Usage

```sh
$ eglogs -h
Usage: eglogs [options]

Kubernetes source:
  -namespace, -n string   Kubernetes namespace (default "envoy-gateway-system")
  -selector, -l string    Pod label selector (default "gateway.envoyproxy.io/owning-gateway-name=main")
  -container, -c string   Container name (default "envoy")
  -kubeconfig string      Optional path to explicit kubeconfig file
  -tail int               Lines of recent log history to show (default 100)

Log filters:
  -include, -i regexp     Regex pattern to include (can be repeated)
  -exclude, -e regexp     Regex pattern to exclude (can be repeated)
  -status, -s range       HTTP response codes (e.g. '404,500', '200-300', '400-')
  -duration, -d range     Request duration in ms (e.g. '-200', '200-500', '1000-')

Output:
  -json, -j               Emit raw JSON log lines instead of prettified text
  -path-segments, -p int  Limit the request path to this many segments (default unlimited)
  -omit-user-agent, -u    Omit the user agent from prettified output
  -xds-route-names, -x    Show XDS route and rule names in prettified output
  -h, --help              Show help
```


Select a different gateway or namespace:

```sh
eglogs -n my-gateway-system \
  -selector 'gateway.envoyproxy.io/owning-gateway-name=my-gateway'
```

Show recent server errors taking at least one second:

```sh
eglogs -tail 100 -status '500-' -duration '1000-'
```

Include or exclude matching log lines. These flags may be repeated:

```sh
eglogs -i /api/ -e /healthz -e Blackbox-Exporter
```

Emit the original JSON instead of formatted output:

```sh
eglogs -json
```

Show the XDS route and rule names in formatted output:

```sh
eglogs -x
```

Limit the request path to two segments and omit the user agent from formatted output:

```sh
eglogs -p 2 -omit-user-agent
```

Use a specific kubeconfig or container:

```sh
eglogs -kubeconfig ~/.kube/config -c envoy
```

Run `eglogs -h` for all options. Kubernetes flags support `-n`/`-namespace`,
`-l`/`-selector`, and `-c`/`-container`. Short forms for filters and output are
`-i`, `-e`, `-s`, `-d`, `-j`, `-p` (path segment limit), `-u` (omit user agent),
and `-x` (XDS route and rule names).

## Build

Build a local executable from the repository root:

```sh
go build -o eglogs .
```

Run it without installing:

```sh
go run .
```
