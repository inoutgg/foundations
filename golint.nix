{
  pkgs ? import <nixpkgs> { },
  config ? { },
}:
pkgs.stdenv.mkDerivation {
  pname = "golangci-lint";
  nativeBuildInputs = with pkgs; [
    golangci-lint
  ];

  dontUnpack = true;
  dontStrip = true;

  buildPhase = ''
    ${pkgs.writeTextFile ".custom-gcl.yml" ''
      ${pkgs.formats.yaml config}
    ''}
    golangci-lint custom
  '';

  installPhase = ''
    mkdir -p $out/bin
    cp -R $build $out/bin
  '';

  meta = with pkgs.lib; {
    platforms = platforms.all;
  };
}
