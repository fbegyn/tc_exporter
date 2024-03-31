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
        vendorHash = "sha256-Ua7fadV2l5J6qg6Irh/vF40FD4o6G5ug72ZcD63s7B0";
        ldflags = [
          "-s" "-w"
        ];
      };
      devShells.default = pkgs.mkShell rec {
        buildInputs = with pkgs; [
          go_1_22
          gofumpt
          go-tools
          git
          nix
          nfpm
          goreleaser
          gnumake
        ];
      };
    });
}
