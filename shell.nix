let
  pkgs = import <nixpkgs> {};
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    go_1_19
    nfpm
  ];
}
