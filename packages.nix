{pkgs, ...}:
with pkgs; let
  basepkg = name:
    buildGoModule {
      name = name;
      src = stdenv.mkDerivation {
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
      vendorSha256 = "sha256-oUmFwNRJvbdlX3QzudPPcyAJ0gbBw8dGptLcb0ExzoY=";
      #vendorSha256 = pkgs.lib.fakeSha256;
      subPackages =
        if name == "tc_exporter"
        then []
        else ["./cmd/${name}"];

      ldflags = [
        "-X github.com/prometheus/common/version.Version=${pkgs.lib.removeSuffix "\n" (builtins.readFile ./VERSION)}"
        "-X github.com/prometheus/common/version.Branch=n/a"
        "-X github.com/prometheus/common/version.Revision=n/a"
        "-X github.com/prometheus/common/version.BuildUser=n/a"
        "-X github.com/prometheus/common/version.BuildDate=n/a"
      ];
    };
  packageList =
    builtins.mapAttrs
    (
      name: value:
        basepkg name
    )
    (builtins.readDir ./cmd);
  dockerPackageList =
    lib.mapAttrs'
    (name: value:
      lib.nameValuePair
      "docker-${name}"
      (pkgs.dockerTools.buildImage {
        name = name;
        tag = "latest";
        contents = [pkgs.bashInteractive (builtins.getAttr name packageList)];
        config = {
          Entrypoint = ["/bin/${name}"];
        };
      }))
    (builtins.readDir ./cmd);
in
  lib.recursiveUpdate
  (lib.recursiveUpdate packageList dockerPackageList)
  rec {
    tc_exporter = basepkg "tc_exporter";
    nfpmPackages = let
  }
