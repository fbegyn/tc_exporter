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
        src = ./.;
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
        vendorHash = "sha256-nNHmEAmrpC/hS3KOyhxsHyWdH7q4YCjQLD7GOegGMd0=";
        subPackages = [];
        ldflags = [
          "-s" "-w"
        ];
      };
      devShells.default = pkgs.mkShell rec {
        buildInputs = [
          pkgs.go_1_23
          pkgs.gofumpt
          pkgs.go-tools
          pkgs.git
          pkgs.nix
          pkgs.nfpm
          pkgs.goreleaser
        ];
      };
    });
}
