package templates

import (
	"github.com/grafana/agent/pkg/operator/cueconfig:config"
)

#Input: config.#Deployment

server: {
	let spec = #Input.Agent.spec

	http_listen_port: 8080

	// TODO(rfratto): replace with language builtin "exists" once that is
	// implemented. https://github.com/cuelang/cue/issues/943
	if #Input.Agent.spec.logLevel != _|_ {
		log_level: spec.logLevel
	}

	log_format: #Input.Agent.spec.logFormat | "logfmt"
}

prometheus: {
	wal_directory: "/var/lib/grafana-agent/data"
}
