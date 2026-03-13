FROM registry.0xkowalski.dev/gamejanitor/base

USER root
RUN apt-get update && \
    apt-get install -y --no-install-recommends unzip ca-certificates && \
    rm -rf /var/lib/apt/lists/*
USER gameserver

COPY scripts/ /scripts/

USER root
RUN chmod +x /scripts/*
USER gameserver

EXPOSE 7777
