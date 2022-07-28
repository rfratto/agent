---
aliases:
- /docs/agent/latest/flow/configuration-language/expressions
title: Expressions
weight: 400
---

# Expressions

Expressions represent or compute values that can be assigned to attributes
within a configuration.

Basic expressions are literal values, like `"Hello, world!"` or `true`.
Expressions may also do things like refer to values exported by components or
perform arithmetic.

Expressions can be used when configuring any component. However, blocks used to
configure Grafana Agent Flow's behavior like [`logging`][] do not support
expressions when configuring an attribute, and may only use literal values.

[`logging`]: {{< relref "../../controller-config/logging.md" >}}
