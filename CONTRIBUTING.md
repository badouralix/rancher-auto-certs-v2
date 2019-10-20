# Contributing

- [Bootstrap](#bootstrap)
- [Local run](#local-run)
- [Test](#test)

## Bootstrap

To create the go module, see [https://medium.com/mindorks/create-projects-independent-of-gopath-using-go-modules-802260cdfb51](https://medium.com/mindorks/create-projects-independent-of-gopath-using-go-modules-802260cdfb51):

```bash
go mod init github.com/badouralix/rancher-auto-certs-v2
# For some reasons, it does not pick github.com/go-acme/lego@v3.1.0 which fixes the issues below in https://github.com/go-acme/lego/pull/943
go mod edit -require=github.com/go-acme/lego@v2.6.0
```

`go-acme/lego` does not support go modules yet ( see [go-acme/lego/issues/827](https://github.com/go-acme/lego/issues/827) and [go-acme/lego/pull/706](https://github.com/go-acme/lego/pull/706) )

The encountered failures are:

```text
# github.com/go-acme/lego/providers/dns/designate
$GOPATH/pkg/mod/github.com/go-acme/lego@v2.6.0+incompatible/providers/dns/designate/designate.go:206:3: cannot use record.TTL (type int) as type *int in field value
# github.com/go-acme/lego/providers/dns/dnspod
$GOPATH/pkg/mod/github.com/go-acme/lego@v2.6.0+incompatible/providers/dns/dnspod/dnspod.go:140:19: cannot convert 0 (type untyped number) to type "encoding/json".Number
$GOPATH/pkg/mod/github.com/go-acme/lego@v2.6.0+incompatible/providers/dns/dnspod/dnspod.go:140:19: invalid operation: hostedZone.ID == 0 (mismatched types "encoding/json".Number and int)
```

`rancher/go-rancher` latest official release was built a few years ago ( see [rancher/go-rancher/releases/tag/v0.1.0](https://github.com/rancher/go-rancher/releases/tag/v0.1.0) ) and it does not provide features we need here:

```text
# github.com/badouralix/rancher-auto-certs-v2
./main.go:56:35: existingCerts[cc.Name].ExpiresAt undefined (type *client.Certificate has no field or method ExpiresAt)
./main.go:57:78: existingCerts[cc.Name].ExpiresAt undefined (type *client.Certificate has no field or method ExpiresAt)
./rancher.go:28:3: unknown field 'Timeout' in struct literal of type client.ClientOpts
```

To fix this, we hardcode the versions of a few dependencies:

```bash
go get github.com/decker502/dnspod-go@83a3ba562b048c9fc88229408e593494b7774684
go get github.com/gophercloud/gophercloud@a2b0ad6ce68c8302027db1a5f9dbb03b0c8ab072
go get github.com/rancher/go-rancher@222ed122ed79d4facfa1bfbb24772530e0f9f900
```

Once `go.mod` is updated, we get the actual versions to use with `go mod edit`:

```bash
go mod edit -require=github.com/decker502/dnspod-go@v0.0.0-20180416134550-83a3ba562b04
go mod edit -require=github.com/gophercloud/gophercloud@v0.0.0-20190204021847-a2b0ad6ce68c
go mod edit -require=github.com/rancher/go-rancher@v0.1.1-0.20190320041936-222ed122ed79
```

Note that we still facing minor errors

```bash
$ go mod tidy
go: github.com/h2non/gock@v1.0.14: parsing go.mod: unexpected module path "gopkg.in/h2non/gock.v1"
go: error loading module requirements
# See https://github.com/h2non/gock/issues/50
```

## Local run

Use the following command:

```bash
go run .
```

## Test

TBD
