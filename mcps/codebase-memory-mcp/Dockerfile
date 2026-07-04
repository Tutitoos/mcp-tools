FROM debian:bookworm-slim

ARG UID=1000
ARG GID=1000

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        tar \
        gzip \
        git \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g "${GID}" mcp \
    && useradd -m -u "${UID}" -g "${GID}" -s /bin/bash mcp \
    && mkdir -p /home/mcp/.cache/codebase-memory-mcp \
    && mkdir -p /home/mcp/.config/codebase-memory-mcp \
    && mkdir -p /opt/codebase-memory-mcp \
    && chown -R mcp:mcp /home/mcp /opt/codebase-memory-mcp

USER mcp

ENV HOME=/home/mcp
ENV RUST_LOG=warn

RUN curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh \
    | bash -s -- --standard --dir /opt/codebase-memory-mcp --skip-config

USER root

RUN ln -sf /opt/codebase-memory-mcp/codebase-memory-mcp /usr/local/bin/codebase-memory-mcp

USER mcp

WORKDIR /home/mcp

ENTRYPOINT ["/usr/local/bin/codebase-memory-mcp"]
