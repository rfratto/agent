---
aliases:
- /docs/agent/latest/flow/
title: Grafana Agent Flow
weight: 900
---

# Grafana Agent Flow (Experimental)

Grafana Agent Flow is a _component-based_ experimental revision of Grafana
Agent with a focus on ease-of-use, debuggability, and ability to adapt to the
needs of power users.

> **WARNING**: Grafana Agent Flow is experimental, and is subject to frequent
> breaking changes or removal from the project at any time.
>
> You should only use Grafana Agent Flow if you are okay with bleeding edge
> functionality and want to provide feedback to the developers.
>
> It is not recommended to use Grafana Agent Flow in production.

## Features

* Write declarative configurations with a Terraform-inspired configuration
  language
* Declare components to configure parts of a pipeline
* Use expressions to bind components together to build a programmable pipeline

## Example

```river
// Discover Kubernetes pods to collect metrics from.
discovery.kubernetes "pods" {
  role = "pod"
}

// Scrape metrics from Kubernetes pods and send to a metrics.remote_write
// component.
metrics.scrape "default" {
  targets    = discovery.kubernetes.pods.targets
  forward_to = [metrics.remote_write.default.receiver]
}

// Get an API key from disk.
local.file "apikey" {
  filename  = "/var/data/my-api-key.txt"
  is_secret = true
}

// Collect and send metrics to a Prometheus remote_write endpoint.
metrics.remote_write "default" {
  remote_write {
    url = "http://localhost:9009/api/prom/push"

    basic_auth {
      username = "MY_USERNAME"
      password = local.file.apikey.content
    }
  }
}
```

## Next steps

* Read our [getting started guide]({{< relref "./getting_started" >}})
* Read documentation on the [configuration language]({{< relref "./river" >}})
* Understand the [component system]({{< relref "./controller" >}}) used by Grafana Agent Flow
* View the list of available [components]({{< relref "./components/" >}})

## Provide feedback

Feedback about Grafana Agent Flow and its configuration language can be
provided in [this](https://github.com/grafana/agent/discussions/1969) GitHub
discussion.
