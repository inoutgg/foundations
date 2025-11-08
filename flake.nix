{
  description = "foundations - a modular library designed to build maintainable production-grade systems.";

  inputs = {
    devenv.url = "github:cachix/devenv";
    nixpkgs.url = "nixpkgs/nixos-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    flake-parts.url = "github:hercules-ci/flake-parts";
    flake-root.url = "github:srid/flake-root";
  };

  outputs =
    {
      flake-parts,
      ...
    }@inputs:
    flake-parts.lib.mkFlake { inherit inputs; } {
      flake = { };

      systems = [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      imports = [
        inputs.flake-root.flakeModule
        inputs.treefmt-nix.flakeModule
        inputs.devenv.flakeModule
      ];

      perSystem =
        {
          pkgs,
          lib,
          config,
          ...
        }:
        {
          formatter = config.treefmt.build.wrapper;

          treefmt.config = {
            inherit (config.flake-root) projectRootFile;
            package = pkgs.treefmt;

            programs = {
              nixfmt.enable = true;
              gofumpt.enable = true;
              yamlfmt.enable = true;
            };
          };

          devenv.shells.default = {
            containers = lib.mkForce { };

            packages = with pkgs; [
              watchexec
              just
              lefthook
              typos

              gcc
              gotools
              govulncheck
              golangci-lint
              mockgen
            ];

            languages.go = {
              enable = true;
              package = pkgs.go_1_25;
            };

            env.GOTOOLCHAIN = lib.mkForce "local";
            env.GOFUMPT_SPLIT_LONG_LINES = lib.mkForce "on";

            env.TEST_DATABASE_URL = lib.mkForce "postgres://test:test@localhost:5432/test";
            services.postgres = {
              enable = true;
              package = pkgs.postgresql_17;

              initialScript = ''
                CREATE USER test SUPERUSER PASSWORD 'test';
                CREATE DATABASE test OWNER test;
              '';
              listen_addresses = "localhost";
              port = 5432;
            };
          };
        };
    };
}
