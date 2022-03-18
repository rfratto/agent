# syntax=docker/dockerfile:1.4
# This dockerfile requires using BuildKit (DOCKER_BUILDKIT=1) or using docker
# buildx.

# Use a base image that has a bunch of cross compilers so this build image can
# be used for building for any officially supported platform.
#
# NOTE: rfratto/seego:latest is currently amd64-only.
FROM rfratto/seego:latest

# Install Go
ENV GOLANG_VERSION         1.17.6
ENV GOLANG_DOWNLOAD_URL    https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 231654bbf2dab3d86c1619ce799e77b03d96f9b50770297c8f4dff8836fc8ca2
RUN <<EOF
  # Remove existing go from seego
  rm -rf /usr/local/go
  curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz
  echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c -
  tar -C /usr/local -xzf golang.tar.gz
  rm golang.tar.gz
EOF

# Install dependencies
RUN <<EOF
  HOST_ARCH=$(dpkg --print-architecture)

  # Essential dependencies for building
  apt-get update && apt-get install -y \
    apt-transport-https build-essential ca-certificates curl gnupg lsb-release \
    rubygems rpm nsis

  # Add docker cli for interacting with host docker daemon
  curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add -
  add-apt-repository "deb [arch=$HOST_ARCH] https://download.docker.com/linux/debian buster stable"
  apt-get update && apt-get install -y \
    docker-ce docker-ce-cli containerd.io gettext

  # Other miscellenaous dependencies used with the Makefile
  gem install --no-ri --no-rdoc fpm
  go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0
  go install github.com/mitchellh/gox@v1.0.1
  go install github.com/tcnksm/ghr@v0.14.0

  # Cleanup: Remove the apt cache to minimize the size of this layer
  rm -rf /var/lib/apt/lists/*
EOF

# Use seegocc for all crosscompiling.
ENV CC=seegocc

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
WORKDIR /src
