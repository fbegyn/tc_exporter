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
        config = import ./go.nix;
      };
    in rec {
      packages = import ./packages.nix { inherit pkgs;};
      defaultPackage = packages.tc_exporter;
      devShell = pkgs.mkShell rec {
        buildInputs = [
          pkgs.go
          pkgs.gofumpt
          pkgs.gotools
          pkgs.go-tools
          pkgs.gotestsum
          pkgs.golangci-lint
          pkgs.git
          pkgs.nix
          pkgs.nfpm
          pkgs.goreleaser
        ];
      };
    });
}
