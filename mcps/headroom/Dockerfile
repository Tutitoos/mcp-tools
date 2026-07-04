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
    && mkdir -p /home/mcp/.cache /home/mcp/.config/headroom /home/mcp/.local/share/headroom \
    && chown -R mcp:mcp /home/mcp

USER mcp

ENV HOME=/home/mcp
ENV PATH="/home/mcp/.local/bin:${PATH}"
ENV PYTHONUNBUFFERED=1
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

RUN pip install --no-cache-dir "headroom-ai[mcp,proxy]"

WORKDIR /home/mcp

ENTRYPOINT ["sleep"]
CMD ["infinity"]
