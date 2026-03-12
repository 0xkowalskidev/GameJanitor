{
  description = "Gamejanitor - local game server hosting tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      packages.${system}.default = pkgs.buildGoModule {
        pname = "gamejanitor";
        version = "0.1.0";
        src = ./.;
        vendorHash = null; # Updated after go mod vendor
        CGO_ENABLED = 1;
        buildInputs = [ pkgs.sqlite ];
        nativeBuildInputs = [ pkgs.pkg-config ];
        subPackages = [ "cmd/gamejanitor" ];
      };

      nixosModules.default = ./nixos/module.nix;

      devShells.${system}.default = pkgs.mkShell {
        buildInputs = [
          pkgs.go
          pkgs.sqlite
          pkgs.docker-client
          pkgs.pkg-config
          pkgs.gcc
        ];

        shellHook = ''
          export CGO_ENABLED=1
        '';
      };

      apps.${system} = {
        build-image = {
          type = "app";
          program = toString (pkgs.writeShellScript "build-image" ''
            game="$1"
            if [ -z "$game" ]; then
              echo "Usage: nix run .#build-image -- <game>"
              exit 1
            fi
            docker build -t "registry.0xkowalski.dev/gamejanitor/$game" "images/$game"
          '');
        };

        push-image = {
          type = "app";
          program = toString (pkgs.writeShellScript "push-image" ''
            game="$1"
            if [ -z "$game" ]; then
              echo "Usage: nix run .#push-image -- <game>"
              exit 1
            fi
            docker push "registry.0xkowalski.dev/gamejanitor/$game"
          '');
        };

        push-all-images = {
          type = "app";
          program = toString (pkgs.writeShellScript "push-all-images" ''
            for dir in images/*/; do
              game=$(basename "$dir")
              echo "Building and pushing $game..."
              docker build -t "registry.0xkowalski.dev/gamejanitor/$game" "images/$game"
              docker push "registry.0xkowalski.dev/gamejanitor/$game"
            done
          '');
        };
      };
    };
}
