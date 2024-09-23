{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/917bb8ae5404879542d4c409091b1102637dc761.tar.gz") { } }:
pkgs.mkShell {
  hardeningDisable = [ "fortify" ]; # needed for dlv to work (https://github.com/NixOS/nixpkgs/issues/18995)
  # packages = [
  #   (pkgs.python311.withPackages (python-pkgs: [
  #     python-pkgs.pip
  #     python-pkgs.build
  #     python-pkgs.twine
  #   ]))
  # ];
  buildInputs = with pkgs; [
    # go
    go_1_22
    go-bindata
    gotools
    goimports-reviser

    #python
    # python311

    gnumake
  ];

  GOPRIVATE = "github.com/RafaySystems/*";
}
