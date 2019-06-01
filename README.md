# `rancher-auto-certs-v2`

<https://github.com/jonremy/rancher-auto-certs> with wildcard support

- [DNS Provider setup](#dns-provider-setup)
  - [OVH](#ovh)
- [Rancher setup](#rancher-setup)
- [Configuration](#configuration)

## DNS Provider setup

### OVH

| Environment Variable Name | Description                     |
| :------------------------ | :------------------------------ |
| `OVH_APPLICATION_KEY`     | Application key                 |
| `OVH_APPLICATION_SECRET`  | Application secret              |
| `OVH_CONSUMER_KEY`        | Consumer key                    |
| `OVH_ENDPOINT`            | Endpoint URL (ovh-eu or ovh-ca) |

Create keys in <https://eu.api.ovh.com/createToken/>

| Field              | Value                               |
| :----------------- | :---------------------------------- |
| Script name        | rancher-auto-certs-v2               |
| Script description | Resolve ACME DNS-01 challenge       |
| Validity           | Unlimited                           |
| Rights             | POST /domain/zone/[FQDN]/record     |
| Rights             | POST /domain/zone/[FQDN]/refresh    |
| Rights             | DELETE /domain/zone/[FQDN]/record/* |

More documentation on <https://github.com/ovh/go-ovh#use-the-api-for-a-single-user>

Warning <https://community.ovh.com/t/createtoken-invalid-account-password/12454/2>

## Rancher setup

| Environment Variable Name | Description                                                                                                                       |
| :------------------------ | :-------------------------------------------------------------------------------------------------------------------------------- |
| `CATTLE_URL`              | The URL that is in the [host registration](https://rancher.com/docs/rancher/v1.6/en/configuration/settings/#host-registration)    |
| `CATTLE_ACCESS_KEY`       | An access key for the [environment](https://rancher.com/docs/rancher/v1.6/en/environments/) that the service is being launched in |
| `CATTLE_SECRET_KEY`       | A secret key for the access key                                                                                                   |

These environment variables are automatically provisioned for service accounts. Add the following labels to the Rancher service:

|                 Key                 |     Value     | Description                                                                                                                  |
| :---------------------------------: | :-----------: | :--------------------------------------------------------------------------------------------------------------------------- |
| `io.rancher.container.create_agent` |    `true`     | Used to indicate that the service account API keys will be passed as environment variables on each container                 |
|  `io.rancher.container.agent.role`  | `environment` | Used to indicate what kind of role the account will be. The value to use for creating service accounts will be `environment` |

More documentation on <https://rancher.com/docs/rancher/v1.6/en/rancher-services/service-accounts/>

## Configuration

Configuration is stored in `config/config.yml`. See [example](config/config-example.yml).

It populates a `globalConfig` struct defined by the following:

```go
type certConfig struct {
	AccountEmail       string `yaml:"account_email"`
	AccountKey         string `yaml:"account_key"`
	CA                 string
	Challenge          string
	CreateKeyIfMissing *bool `yaml:"create_key_if_missing"`
	Description        string
	Domains            []string
	KeyType            string `yaml:"key_type"`
	Name               string
	Provider           string `json:",omitempty" yaml:",omitempty"`
}

type defaultConfig struct {
	AccountEmail       string `yaml:"account_email"`
	AccountKey         string `yaml:"account_key"`
	CA                 string
	Challenge          string
	CreateKeyIfMissing bool `yaml:"create_key_if_missing"`
	Description        string
	KeyType            string `yaml:"key_type"`
	Provider           string `json:",omitempty" yaml:",omitempty"`
}

type globalConfig struct {
	Default defaultConfig
	Certs   []certConfig
}
```

Each missing key in `certConfig` is then populated by values from `defaultConfig`.
