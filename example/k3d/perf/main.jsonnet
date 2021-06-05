local default = import 'default/main.libsonnet';
local k = import 'ksonnet-util/kausal.libsonnet';

local avalanche = import 'avalanche/main.libsonnet';
local grafana_agent = import 'grafana-agent/v1/main.libsonnet';

local containerPort = k.core.v1.containerPort;
local ingress = k.networking.v1beta1.ingress;
local path = k.networking.v1beta1.httpIngressPath;
local rule = k.networking.v1beta1.ingressRule;
local service = k.core.v1.service;

local new_agent(name, version) =
  grafana_agent.newDeployment('grafana-agent-' + name, 'default') +
  grafana_agent.withImages({
    agent: 'grafana/agent:' + version,
    agentctl: 'grafana/agentctl:' + version,
  }) +
  grafana_agent.withPrometheusConfig({
    wal_directory: '/var/lib/agent/data',
    global: {
      scrape_interval: '30s',
      external_labels: { cluster: version },
    },
  }) +
  grafana_agent.withPrometheusInstances(grafana_agent.scrapeInstanceKubernetes {
    scrape_configs: std.map(function(config) config {
      relabel_configs+: [{ target_label: 'cluster', replacement: version }],
    }, super.scrape_configs),
    remote_write: [{
      url: 'http://cortex.default.svc.cluster.local/api/prom/push',
    }],
  });

{
  default: default.new(namespace='default') {
    grafana+: {
      ingress+:
        ingress.new('grafana-ingress') +
        ingress.mixin.spec.withRules([
          rule.withHost('grafana.k3d.localhost') +
          rule.http.withPaths([
            path.withPath('/')
            + path.backend.withServiceName('grafana')
            + path.backend.withServicePort(80),
          ]),
        ]),
    },
  },

  agent_v0_13_1: new_agent(name='v0-13-1', version='v0.13.1'),
  agent_v0_14_0: new_agent(name='v0-14-0', version='v0.14.0'),

  local avalanche_config = {
    _config+:: {
      metric_count: 1500,
    },
  },

  avalanche_1: avalanche.new(name='avalanche-1') + avalanche_config,
  avalanche_2: avalanche.new(name='avalanche-2') + avalanche_config,
  avalanche_3: avalanche.new(name='avalanche-3') + avalanche_config,
}
