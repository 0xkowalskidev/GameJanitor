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

# Pre-install Valheim dedicated server (app 896660) so first start only needs a delta update
# First run may self-update and exit non-zero, so run twice
RUN steamcmd.sh +quit || true
RUN steamcmd.sh \
    +force_install_dir /data/server \
    +login anonymous \
    +app_update 896660 validate \
    +quit

COPY scripts/ /scripts/

USER root
RUN chmod +x /scripts/*
USER gameserver

EXPOSE 2456/udp 2457/udp
