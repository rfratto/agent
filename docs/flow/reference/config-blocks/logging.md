---
aliases:
- /docs/agent/latest/flow/reference/config-blocks/logging
title: logging
weight: 100
---

# `logging` block

`logging` is an optional block which configures the logging level and format of
Grafana Agent Flow logs.

Log lines are always written to standard error.

The `logging` block may only be provided once within a config file and does not
support specifying labels.

## Example

```river
logging {
  level  = "debug"
  format = "json"
}
```

## Arguments

This block can be configured with the following arguments:

Name | Type | Description | Default | Required
---- | ---- | ----------- | ------- | --------
`level` | `string` | Level to emit logs at (debug, info, warn, error) | `"info"` | no
`format` | `string` | Specifies how emitted logs are formatted (logfmt, json) | `"logfmt"` | no
