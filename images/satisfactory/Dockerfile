FROM registry.0xkowalski.dev/gamejanitor/base

USER root

# SteamCMD requires 32-bit libraries
RUN dpkg --add-architecture i386 && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
        lib32stdc++6 && \
    rm -rf /var/lib/apt/lists/*

# Install SteamCMD
RUN mkdir -p /opt/steamcmd && \
    curl -fsSL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz | tar xz -C /opt/steamcmd && \
    chown -R gameserver:gameserver /opt/steamcmd
ENV PATH="/opt/steamcmd:${PATH}"

USER gameserver

# Pre-install Satisfactory dedicated server so first start only needs a delta update
# First run may self-update and exit non-zero, so run twice
# SteamCMD exits non-zero even on successful install, so verify the binary exists
RUN steamcmd.sh +quit || true && \
    (steamcmd.sh \
    +force_install_dir /data/server \
    +login anonymous \
    +app_update 1690800 -beta public validate \
    +quit || true) && \
    test -f /data/server/FactoryServer.sh

COPY scripts/ /scripts/

USER root
RUN chmod +x /scripts/*
USER gameserver

STOPSIGNAL SIGINT

EXPOSE 7777/udp 7777/tcp 8888/tcp
