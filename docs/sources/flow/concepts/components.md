---
aliases:
- /docs/agent/latest/concepts/components
title: Components
weight: 100
---

# Components

Pipelines within Grafana Agent Flow are built by specifying and chaining
together multiple _components_, each of which handles a specific task for
collecting telemetry.

Each component is composed of two parts:

* Arguments: settings which configure a component.
* Exports: named values which a component exposes to other components.

Each component has a name like `local.file` which generally describes what that
component is responsible for.

In addition to a name, components have user-specified labels. The combination
of the component's name and its label must be unique: this allows users to have
more than one `local.file` component, but not two `local.file` components with
the same `foobar` label.

> An "instance" of a component refers to a component that a user specified in
> the configuration file. Each component has a unique ID, which is formed by
> combining the component's name with its label.
>
> For example, a `local.file` component labeled `foo` has the ID
> `local.file.foo`. Conventionally, instances of components are referenced by
> their ID.

## Expressions

Each argument for a component can have its value computed using expressions.
Expressions allow users to dynamically determine the value to use. Expressions
can be things such as referencing one of the exports of another component
(`local.file.my_file.contents`) or getting the value of an environment variable
(`env("HOME")`).

## Components are dynamic

A component is a dynamic piece of your pipeline: its arguments and exports may
change throughout the lifetime of the Grafana Agent Flow process.

If the exports of a component changes, the arguments for any component which is
referencing it will be automatically reevaluated, and the component will be
reconfigured with the new arguments. For example, any component using a
`local.file.api_key` component instance for credentials will be updated with
the latest credentials if that file changes.

## Example pipeline

Components may do things like:

* Watch a file on disk so other files can reference its contents ([`local.file`][])
* Discover Kubernetes pods where Prometheus metrics may be collected from ([`discovery.kubernetes`][])
* Collect Prometheus metrics from discovered services ([`metrics.scrape`][])
* Forward Prometheus metrics to a database ([`metrics.remote_write`][])

You can imagine how these components might be chained together:

1. The `discovery.kubernetes` component finds pods where metrics can be collected.
2. The `metrics.scrape` component collects metrics from those pods, and sends
   metrics to the `metrics.remote_write` component.
3. The `metrics.remote_write` component forwards metrics to a database, using
   the file contents of `local.file` for an API key.

A user would use this config file to represent the above pipeline:

```river
// Get our API key from disk.
//
// This component has an exported value called "contents", holding the contents
// of the file.
//
// local.file.api_key will watch the file and update its exports any time the
// file changes.
local.file "api_key" {
  filename  = "/var/data/secrets/api-key"

  // Mark this file as sensitive to prevent its value from being shown to
  // users.
  is_secret = true
}

// Create a metrics.remote_write component which other components can send
// metrics to.
//
// This component exports a "receiver" value which can be used by other
// components to send metrics.
metrics.remote_write "prod" {
  remote_write {
    url = "https://prod:9090/api/v1/write"

    basic_auth {
      username = "admin"

      // Use our password file for authenticating with the production database.
      password = local.file.api_key.contents
    }
  }
}

// Find Kubernetes pods where we can collect metrics.
//
// This component exports a "targets" value which contains the list of
// discovered pods.
discovery.kubernetes "pods" {
  role = "pod"
}

// Collect metrics from Kubernetes pods and send them to prod.
metrics.scrape "default" {
  targets    = discovery.kubernetes.pods.targets
  forward_to = [metrics.remote_write.prod.receiver]
}
```

[`local.file`]: {{< relref "../reference/components/local.file" >}}
[`discovery.kubernetes`]: {{< relref "../reference/components/discovery.kubernetes" >}}
[`metrics.scrape`]: {{< relref "../reference/components/metrics.scrape" >}}
[`metrics.remote_write`]: {{< relref "../reference/components/metrics.remote_write" >}}
