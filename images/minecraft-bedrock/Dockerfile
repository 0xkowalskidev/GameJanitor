FROM registry.0xkowalski.dev/gamejanitor/base

USER root
RUN apt-get update && \
    apt-get install -y --no-install-recommends unzip libcurl4 ca-certificates jq && \
    rm -rf /var/lib/apt/lists/*
USER gameserver

COPY scripts/ /scripts/
COPY defaults/ /defaults/

USER root
RUN chmod +x /scripts/*
USER gameserver

EXPOSE 19132/udp
