{
  pkgs ? import <nixpkgs> { },
}:
{
  plugins ? { },
}:
let
  config = {
    name = "golangci-lint";
    version = "v${pkgs.golangci-lint.version}";
    plugins = plugins;
  };

  configFile = pkgs.writeTextFile {
    name = "custom-gcl.yml";
    text = builtins.toJSON config;
  };
in
pkgs.stdenv.mkDerivation {
  name = "golangci-lint-custom";
  pname = "golangci-lint-custom";
  nativeBuildInputs = with pkgs; [
    git
    # golangci-lint
  ];

  dontUnpack = true;
  dontStrip = true;

  buildPhase = ''
    # Copy config file to build directory
    cp ${configFile} .custom-gcl.yml

    ${pkgs.golangci-lint}/bin/golangci-lint custom
  '';

  installPhase = ''
    mkdir -p $out/bin
  '';

  meta = with pkgs.lib; {
    platforms = platforms.all;
  };
}
