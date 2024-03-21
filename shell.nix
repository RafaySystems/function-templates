{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/4e50404e2f3403b020ac59986fcb517a0e7b119f.tar.gz") { } }:
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
