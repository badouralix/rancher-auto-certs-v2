# Contributing

- [Bootstrap](#bootstrap)
- [Local run](#local-run)
- [Test](#test)

## Bootstrap

To create the go module, see [https://medium.com/mindorks/create-projects-independent-of-gopath-using-go-modules-802260cdfb51](https://medium.com/mindorks/create-projects-independent-of-gopath-using-go-modules-802260cdfb51):

```bash
go mod init github.com/badouralix/rancher-auto-certs-v2
go mod tidy
```

`rancher/go-rancher` latest official release was built a few years ago ( see [rancher/go-rancher/releases/tag/v0.1.0](https://github.com/rancher/go-rancher/releases/tag/v0.1.0) ) and it does not provide features we need here:

```text
$ go build
# github.com/badouralix/rancher-auto-certs-v2
./main.go:70:73: cm.cache[cc.Name].ExpiresAt undefined (type *client.Certificate has no field or method ExpiresAt)
./rancher.go:29:3: unknown field 'Timeout' in struct literal of type client.ClientOpts
```

To fix this, we hardcode the versions of a few dependencies:

```bash
# Pick up latest commit from https://github.com/rancher/go-rancher/commits/master
$ go get github.com/rancher/go-rancher@7577995d47c054cf1d1a65d48882de28cbf429c6
go: github.com/rancher/go-rancher 7577995d47c054cf1d1a65d48882de28cbf429c6 => v0.1.1-0.20200505164325-7577995d47c0
go: finding module for package github.com/gorilla/websocket
go: finding module for package github.com/pkg/errors
go: found github.com/gorilla/websocket in github.com/gorilla/websocket v1.4.2
go: found github.com/pkg/errors in github.com/pkg/errors v0.9.1
```

Once `go.mod` is updated, we get the actual versions to use with `go mod edit`:

```bash
go mod edit -require=github.com/rancher/go-rancher@v0.1.1-0.20200505164325-7577995d47c0
go mod tidy
```

## Local run

Use the following command:

```bash
go run .
```

## Test

TBD
