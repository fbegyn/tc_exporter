{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    ...
  } @inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
      };
    in rec {
      defaultPackage = pkgs.buildGoModule {
        name = "tc_exporter";
        src = pkgs.stdenv.mkDerivation {
          name = "gosrc";
          srcs = [./go.mod ./go.sum ./cmd ./collector ./scripts];
          phases = "installPhase";
          installPhase = ''
            mkdir $out
            for src in $srcs; do
              for srcFile in $src; do
                cp -r $srcFile $out/$(stripHash $srcFile)
              done
            done
          '';
        };
        CGO_ENABLED = 0;
        doCheck = false;
        checkPhase = ''
          staticcheck ./cmd/tc_exporter/
          staticcheck ./collector/
          go test -exec sudo ./collector/ -v
        '';
        checkInputs = [
          pkgs.iproute2
        ];
        vendorSha256 = "sha256-nNHmEAmrpC/hS3KOyhxsHyWdH7q4YCjQLD7GOegGMd0=";
        subPackages = [];
        ldflags = [
          "-s" "-w"
        ];
      };
      devShells.default = pkgs.mkShell rec {
        buildInputs = [
          pkgs.go
          pkgs.gofumpt
          pkgs.go-tools
          pkgs.iproute2
          pkgs.git
          pkgs.nix
          pkgs.nfpm
          pkgs.goreleaser
        ];
      };
    });
}
