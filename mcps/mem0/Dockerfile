FROM python:3.12-slim

ARG UID=1000
ARG GID=1000

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g "${GID}" mcp \
    && useradd -m -u "${UID}" -g "${GID}" -s /bin/bash mcp \
    && mkdir -p /opt/mem0-src /data /home/mcp/.cache/uv \
    && chown -R mcp:mcp /opt/mem0-src /data /home/mcp

RUN pip install --no-cache-dir uv

USER mcp

ENV HOME=/home/mcp
ENV UV_CACHE_DIR=/home/mcp/.cache/uv
ENV PYTHONUNBUFFERED=1

WORKDIR /opt/mem0-src

ENTRYPOINT ["sleep"]
CMD ["infinity"]
