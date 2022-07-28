---
aliases:
- /docs/agent/latest/flow/getting-started
title: Getting Started
weight: 100
---

# Getting Started with Grafana Agent Flow

## Install Grafana Agent

To install Grafana Agent Flow, first [install Grafana Agent][]. Grafana Agent
Flow is an operating mode which will be available in an upcoming Grafana Agent
release.

## Running Grafana Agent Flow

To enable Grafana Agent Flow, set `EXPERIMENTAL_ENABLE_FLOW=true` in your
environment variables.

Then, run Grafana Agent, passing flags to point at a config file to run with:

```
EXPERIMENTAL_ENABLE_FLOW=true agent --config.file=/path/to/my/config.river
```

You can use this file as an example to get started:

```river
metrics.scrape "default" {
  targets = [
    {"__address__" = "demo.robustperception.io:9090"},
  ]
  forward_to = [metrics.remote_write.default.receiver]
}

metrics.remote_write "default" {
  // No remote_writes configured; metrics will be accumulated locally in a WAL
  // and discarded.
}
```

[install Grafana Agent]: {{< relref "../set-up" >}}
