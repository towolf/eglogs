# eglogs

`eglogs` streams Envoy Gateway access logs from Kubernetes and formats them for
easier reading. It can filter logs by regular expression, HTTP status, and
request duration, or emit the original JSON log lines.

## Requirements

- Go 1.26 or later
- A working Kubernetes configuration with permission to list pods and read
  their logs

By default, `eglogs` uses the current kubeconfig context, reads the `envoy`
container in the `envoy-gateway-system` namespace, and selects pods belonging
to a gateway named `main`.

## Install

Clone the repository and install the command with Go:

```sh
git clone https://github.com/towolf/eglogs.git
cd eglogs
go install .
```

The executable is installed in `$GOBIN`, or in `$GOPATH/bin` when `$GOBIN` is
not set. Ensure that directory is included in your `PATH`.

## Usage

Run with the defaults:

```sh
eglogs
```

Select a different gateway or namespace:

```sh
eglogs -namespace my-gateway-system \
  -l 'gateway.envoyproxy.io/owning-gateway-name=my-gateway'
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
eglogs -kubeconfig ~/.kube/config -container envoy
```

Run `eglogs -h` for all options. Short forms are available for the common
filters: `-i`, `-e`, `-s`, `-d`, and `-j`.

## Build

Build a local executable from the repository root:

```sh
go build -o eglogs .
```

Run it without installing:

```sh
go run .
```
