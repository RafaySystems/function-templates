{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/24.11.tar.gz") { } }:
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
    go_1_23
    go-bindata
    
    gotools
    goimports-reviser

    git
    #python
    # python311

    gnumake
  ];

  GOPRIVATE = "github.com/RafaySystems/*";
}
