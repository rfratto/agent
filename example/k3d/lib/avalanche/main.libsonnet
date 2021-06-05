local k = import 'ksonnet-util/kausal.libsonnet';

local configMap = k.core.v1.configMap;
local container = k.core.v1.container;
local containerPort = k.core.v1.containerPort;
local deployment = k.apps.v1.deployment;
local pvc = k.core.v1.persistentVolumeClaim;
local service = k.core.v1.service;
local volumeMount = k.core.v1.volumeMount;
local volume = k.core.v1.volume;

local default_config = {
  metric_count: 500,
  label_count: 10,
  series_count: 10,
};

{
  new(name='avalanche', namespace=''):: {
    local this = self,

    _images:: {
      avalanche: 'quay.io/freshtracks.io/avalanche:master-2020-12-28-0c1c64c',
    },
    _config:: default_config,

    container::
      container.new('avalanche', this._images.avalanche) +
      container.withPorts([
        containerPort.newNamed(name='http-metrics', containerPort=9091),
      ]) +
      container.withArgsMixin([
        '--%s=%s' % [std.strReplace(key, '_', '-'), this._config[key]]
        for key in std.objectFields(this._config)
        if this._config[key] != null
      ]),

    deployment:
      deployment.new(name, 1, [this.container]) +
      deployment.mixin.metadata.withNamespace(namespace),

    service:
      k.util.serviceFor(this.deployment) +
      service.mixin.metadata.withNamespace(namespace),
  },
}
