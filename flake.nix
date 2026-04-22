{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
    devshell = {
      url = "github:numtide/devshell";
      inputs = {
        nixpkgs.follows = "nixpkgs";
      };
    };
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    devshell,
    ...
  } @inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ devshell.overlays.default ];
      };
    in rec {
      defaultPackage = pkgs.buildGoModule {
        name = "tc_exporter";
        src = ./.;
        env.CGO_ENABLED = 0;
        doCheck = false;
        checkPhase = ''
          staticcheck ./cmd/tc_exporter/
          staticcheck ./collector/
          go test -exec sudo ./collector/ -v
        '';
        vendorHash = "sha256-Ldpe/+4HajH3PjbA4eSUA9aeG1stzd5UDLZ2VbG5ADE=";
        ldflags = [
          "-s" "-w"
        ];
      };
      devShells.default = pkgs.devshell.mkShell rec {
        name = "tc-exporer";
        packages = with pkgs; [
          go_1_25
          gofumpt
          go-tools
          git
          nix
          nfpm
          goreleaser
          gnumake
          graphviz
        ];
      };
    });
}
