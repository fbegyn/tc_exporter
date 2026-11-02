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
      packages.default = pkgs.buildGoModule {
        name = "tc_exporter";
        src = ./.;
        env.CGO_ENABLED = 0;
        doCheck = false;
        checkPhase = ''
          staticcheck ./cmd/tc_exporter/
          staticcheck ./collector/
          go test -exec sudo ./collector/ -v
        '';
        vendorHash = "sha256-cNnbw9L5bx7tAyk9DImtJk6GMpOJnFcl1ReOessN/KE=";
        ldflags = [
          "-s" "-w"
        ];
      };

      # the collector tests need netns/netlink privileges, which the nix sandbox
      # doesn't offer, so the checks only cover build, static analysis and format
      checks = {
        build = packages.default;
        lint = packages.default.overrideAttrs (old: {
          nativeBuildInputs = (old.nativeBuildInputs or []) ++ [pkgs.go-tools];
          doCheck = false;
          buildPhase = ''
            runHook preBuild
            export HOME="$TMPDIR"
            go vet ./...
            staticcheck ./...
            runHook postBuild
          '';
          installPhase = "touch $out";
        });
        fmt = pkgs.runCommand "gofumpt-check" {nativeBuildInputs = [pkgs.gofumpt];} ''
          unformatted="$(gofumpt -l ${./.})"
          if [ -n "$unformatted" ]; then
            echo "gofumpt: needs formatting:" >&2
            echo "$unformatted" >&2
            exit 1
          fi
          touch $out
        '';
      };

      formatter = pkgs.writeShellScriptBin "tc-fmt" ''
        exec ${pkgs.gofumpt}/bin/gofumpt -w "''${@:-.}"
      '';

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
        ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux [pkgs.iproute2];
        commands = [
          {
            name = "tc-build";
            help = "build the tc_exporter binary into bin/";
            command = ''go build -o bin/tc_exporter -ldflags "-w -s" ./cmd/tc_exporter'';
          }
          {
            name = "tc-lint";
            help = "run go vet, staticcheck and a gofumpt format check";
            command = ''
              go vet ./... && staticcheck ./...
              unformatted="$(gofumpt -l .)"
              if [ -n "$unformatted" ]; then
                echo "gofumpt: needs formatting:" >&2
                echo "$unformatted" >&2
                exit 1
              fi
            '';
          }
          {
            name = "tc-test";
            help = "run the collector tests (needs sudo for netns/netlink)";
            command = "go test -exec sudo ./collector/";
          }
          {
            name = "tc-fmt";
            help = "format all go code with gofumpt";
            command = "gofumpt -w .";
          }
          {
            name = "tc-check";
            help = "run lint, build and test in one go (pre-push)";
            command = "tc-lint && tc-build && tc-test";
          }
          {
            name = "tc-run";
            help = "build and run the exporter with sudo against ./config.toml (extra args are passed on)";
            command = "tc-build && sudo bin/tc_exporter --config-file . \"$@\"";
          }
          {
            name = "tc-snapshot";
            help = "build a local goreleaser snapshot release";
            command = "goreleaser release --snapshot --clean";
          }
          {
            name = "tc-package";
            help = "build the deb and rpm packages with nfpm";
            command = "make package";
          }
          {
            name = "tc-tidy";
            help = "run go mod tidy (afterwards refresh vendorHash in flake.nix)";
            command = ''
              go mod tidy
              echo "if go.mod/go.sum changed, set vendorHash to lib.fakeHash in flake.nix and rebuild to get the new one"
            '';
          }
          {
            name = "tc-netns";
            help = "tc-netns [up|down|status] [netns=testing] [dev=dummy02]: scratch netns with a dummy iface, htb and fq_codel";
            command = ''
              action="''${1:-up}"
              ns="''${2:-testing}"
              dev="''${3:-dummy02}"
              case "$action" in
                up)
                  sudo ip netns add "$ns"
                  sudo ip -n "$ns" link add "$dev" type dummy
                  sudo ip -n "$ns" link set "$dev" up
                  sudo ip netns exec "$ns" tc qdisc add dev "$dev" root handle 1: htb default 10
                  sudo ip netns exec "$ns" tc class add dev "$dev" parent 1: classid 1:10 htb rate 10mbit
                  sudo ip netns exec "$ns" tc qdisc add dev "$dev" parent 1:10 handle 10: fq_codel
                  ;;
                down)
                  sudo ip netns del "$ns"
                  ;;
                status)
                  sudo ip netns exec "$ns" tc -s qdisc show dev "$dev"
                  ;;
                *)
                  echo "usage: tc-netns [up|down|status] [netns] [dev]" >&2
                  exit 1
                  ;;
              esac
            '';
          }
        ];
      };
    });
}
