{
  description = "foundations - a modular library designed to build maintainable production-grade systems.";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        commonPackages = with pkgs; [
          nodejs
          sqlc
          golangci-lint
        ];
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = commonPackages ++ [ pkgs.go_1_23 ];
        };

        shellHook = ''
          export GOTOOLCHAIN="local"
        '';

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
