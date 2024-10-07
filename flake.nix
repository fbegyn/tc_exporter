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
        config = import ./go.nix;
        overlays = [ devshell.overlays.default ];
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
        vendorHash = "sha256-ZRnfVXV6/QJ98EsgTswxMeIroGfxJ2D436WzCwcVvbU";
        ldflags = [
          "-s" "-w"
        ];
      };
      devShells.default = pkgs.devshell.mkShell rec {
        name = "tc-exporer";
        packages = with pkgs; [
          go_1_23
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
