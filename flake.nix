{
  description = "runners";

  nixConfig = {
    extra-substituters = [
      "https://cache.trev.zip/nur"
    ];
    extra-trusted-public-keys = [
      "nur:70xGHUW1+1b8FqBchldaunN//pZNVo6FKuPL4U/n844="
    ];
  };

  inputs = {
    systems.url = "github:nix-systems/default";
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    trev = {
      url = "github:spotdemo4/nur";
      inputs.systems.follows = "systems";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      nixpkgs,
      trev,
      ...
    }:
    trev.libs.mkFlake (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            trev.overlays.packages
            trev.overlays.libs
          ];
        };
      in
      rec {
        devShells = {
          default = pkgs.mkShell {
            packages = with pkgs; [
              # go
              go
              gotools
              gopls
              golangci-lint
              goreleaser

              # formatting / linting
              nixfmt
              prettier
            ];
            shellHook = pkgs.shellhook.ref;
          };

          release = pkgs.mkShell {
            packages = with pkgs; [
              go
              gotools
              gopls
              goreleaser
            ];
          };
        };

        checks = pkgs.lib.mkChecks {
          go = {
            src = packages.default;
            deps = with pkgs; [
              golangci-lint
            ];
            script = ''
              go test ./...
              golangci-lint run ./...
            '';
          };

          nix = {
            src = ./.;
            deps = with pkgs; [
              nixfmt-tree
            ];
            script = ''
              treefmt --ci
            '';
          };

          actions = {
            src = ./.;
            deps = with pkgs; [
              prettier
              action-validator
              octoscan
              renovate
            ];
            script = ''
              prettier --check .
              action-validator .github/**/*.yaml
              octoscan scan .github
              renovate-config-validator .github/renovate.json
            '';
          };
        };

        packages.default = pkgs.buildGoModule (finalAttrs: {
          pname = "runners";
          version = "0.0.1";

          src = builtins.path {
            name = "root";
            path = ./.;
          };
          vendorHash = null;
          env.CGO_ENABLED = 0;

          meta = {
            description = "runners";
            homepage = "https://github.com/trunners/runners";
            changelog = "https://github.com/trunners/runners/releases/tag/v${finalAttrs.version}";
            license = pkgs.lib.licenses.mit;
            platforms = pkgs.lib.platforms.all;
          };
        });

        formatter = pkgs.nixfmt-tree;
      }
    );
}
