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
      in
      {

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Runtimes
            nodejs
            go

            # Tools
            sqlc
            typos
            mockgen
            just
            golangci-lint
            golint

            # LSP
            typos-lsp
            golangci-lint-langserver
          ];
        };

        shellHook = ''
          export GOTOOLCHAIN="local"
        '';

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
