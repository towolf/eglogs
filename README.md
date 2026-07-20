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

Run with the defaults:

```sh
eglogs
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
eglogs -include '/api/' -exclude '/healthz'
```

Emit the original JSON instead of formatted output:

```sh
eglogs -json
```

Use a specific kubeconfig or container:

```sh
eglogs -kubeconfig ~/.kube/config -c envoy
```

Run `eglogs -h` for all options. Kubernetes flags support `-n`/`-namespace`,
`-l`/`-selector`, and `-c`/`-container`. Short forms for filters and output are
`-i`, `-e`, `-s`, `-d`, and `-j`.

## Build

Build a local executable from the repository root:

```sh
go build -o eglogs .
```

Run it without installing:

```sh
go run .
```
