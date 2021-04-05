# Use the loki-build-image as a base as it shares most of the common tooling. We
# just need some additional tooling for building packages.
FROM grafana/loki-build-image:0.13.0

RUN apt-get update && apt-get install -y \
  build-essential \
  rpm \
  ruby \
  ruby-dev \
  rubygems \
  && gem install --no-ri --no-rdoc fpm \
  && rm -rf /var/lib/apt/lists/*

RUN \
  cd / && \
  GO111MODULE=on go get k8s.io/code-generator/cmd/client-gen@v0.18.2 && \
  GO111MODULE=on go get k8s.io/code-generator/cmd/informer-gen@v0.18.2 && \
  GO111MODULE=on go get k8s.io/code-generator/cmd/lister-gen@v0.18.2 && \
  GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1

# Fix permissions for /go/bin directory.
RUN chmod 0755 /go/bin

COPY build.sh /
RUN chmod +x /build.sh
ENTRYPOINT ["/build.sh"]
