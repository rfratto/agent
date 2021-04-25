// agent.libsonnet is the entrypoint for rendering a Grafana Agent config file
// based on the Operator custom resources.
//
// When writing an object, any field will null will be removed from the final
// YAML. This is useful as we don't want to always translate unfilled values
// from the custom resources to a field in the YAML.
//
// A series of helper methods to convert default values into null (so they can
// be trimmed) are in ./ext/optionals.libsonnet.
//
// When writing a new function, please document the expected types of the
// arguments.

local new_remote_write = import './component/remote_write.libsonnet';

local marshal = import './ext/marshal.libsonnet';
local optionals = import './ext/optionals.libsonnet';

// ctx is expected to be a config.Deployment.
function(ctx) marshal.YAML(optionals.trim({
  local spec = ctx.Agent.Spec,
  local prometheus = spec.Prometheus,

  server: {
    http_listen_port: 8080,
    log_level: optionals.string(spec.LogLevel),
    log_format: optionals.string(spec.LogFormat),
  },

  prometheus: {
    wal_directory: '/var/lib/grafana-agent/data',
    global: {
      external_labels: optionals.object(prometheus.ExternalLabels),
      scrape_interval: optionals.string(prometheus.ScrapeInterval),
      scrape_timeout: optionals.string(prometheus.ScrapeTimeout),
      remote_write: optionals.array(std.map(
        function(rw) new_remote_write(ctx.Agent.ObjectMeta.Namespace, rw),
        prometheus.RemoteWrite,
      )),
    },
  },
}))
