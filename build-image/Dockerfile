FROM rfratto/seego

RUN apt-get update && apt-get install -y \
  apt-transport-https build-essential ca-certificates curl gnupg lsb-release \
  nsis rpm ruby ruby-dev rubygems wget \
  \
  && curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add - \
  && add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian buster stable" \
  && apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io gettext \
  \
  && gem install --no-ri --no-rdoc fpm \
  \
  && rm -rf /usr/local/go \
  && wget -q https://golang.org/dl/go1.16.5.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf go1.16.5.linux-amd64.tar.gz \
  && rm go1.16.5.linux-amd64.tar.gz \
  \
  && rm -rf /var/lib/apt/lists/*

# Install dependencies used for the operator
# Keep versions in sync with cmd/agent-operator/DEVELOPERS.md
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0
RUN go install github.com/mitchellh/gox@v1.0.1
RUN go install github.com/tcnksm/ghr@v0.14.0
