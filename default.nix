{ pkgs ? import <nixpkgs> {} }:
with pkgs;

# assert lib.versionAtLeast go.version "1.18";

buildGoPackage rec {
  name = "rauljordan-personalsite";
  version = "latest";
  goPackagePath = "rauljordan/personalsite";
  src = ./.;

  goDeps = ./deps.nix;
  allowGoReference = false;

  meta = with lib; {
    description = "Personal blog written in Go and nix";
    homepage = "https://rauljordan.com";
    maintainers = with maintainers; [ rauljordan ];
    platforms = platforms.linux ++ platforms.darwin;
  };

  preBuild = ''
    export CGO_ENABLED=0
    buildFlagsArray+=(-pkgdir "$TMPDIR")
  '';

  postInstall = ''
    cp -rf $src/static $bin/static
    cp -rf $src/templates $bin/templates
  '';
}

