FROM registry.access.redhat.com/ubi9/go-toolset:latest as addon
ENV GOPATH=$APP_ROOT
COPY --chown=1001:0 . .
RUN make cmd

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
RUN microdnf -y install \
    glibc-langpack-en \
    openssh-clients \
    subversion \
    git \
    tar \
    python3 \
    curl

RUN GOOSE_VERSION=v1.24.0 \
    curl -fsSL https://github.com/block/goose/releases/download/stable/download_cli.sh | bash \
    && mv ~/.local/bin/goose /usr/local/bin/goose

RUN sed -i 's/^LANG=.*/LANG="en_US.utf8"/' /etc/locale.conf
ENV LANG=en_US.utf8

RUN echo "addon:x:1001:1001:addon user:/addon:/sbin/nologin" >> /etc/passwd
RUN echo -e "StrictHostKeyChecking no" \
 "\nUserKnownHostsFile /dev/null" > /etc/ssh/ssh_config.d/99-konveyor.conf
ENV HOME=/addon ADDON=/addon
WORKDIR /addon
ARG GOPATH=/opt/app-root
COPY --from=addon $GOPATH/src/bin/addon /usr/bin
COPY recipes/goose/recipes /opt/goose/recipes
ENTRYPOINT ["/usr/bin/addon"]
