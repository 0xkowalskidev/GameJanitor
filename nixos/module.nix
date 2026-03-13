self:

{ config, lib, pkgs, ... }:

let
  cfg = config.services.gamejanitor;
in {
  options.services.gamejanitor = {
    enable = lib.mkEnableOption "Gamejanitor game server manager";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      description = "The gamejanitor package to use";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "Port for the web UI and API";
    };

    dataDir = lib.mkOption {
      type = lib.types.path;
      default = "/var/lib/gamejanitor";
      description = "Directory for database and backups";
    };
  };

  config = lib.mkIf cfg.enable {
    virtualisation.docker.enable = true;

    systemd.services.gamejanitor = {
      description = "Gamejanitor Game Server Manager";
      after = [ "network.target" "docker.service" ];
      wants = [ "docker.service" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        ExecStart = "${cfg.package}/bin/gamejanitor serve --port ${toString cfg.port} --data-dir ${cfg.dataDir}";
        Restart = "always";
        RestartSec = 5;
        SupplementaryGroups = [ "docker" ];
        DynamicUser = true;
        StateDirectory = "gamejanitor";
      };
    };
  };
}
