{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/4e50404e2f3403b020ac59986fcb517a0e7b119f.tar.gz") { } }:
pkgs.mkShell {
  hardeningDisable = [ "fortify" ]; # needed for dlv to work (https://github.com/NixOS/nixpkgs/issues/18995)
  buildInputs = with pkgs; [
    # go
    go_1_22
    go-bindata

    # buf
    # golangci-lint
    # go-migrate
    # protobuf    

    # test
    # moq

    #db schema
    # atlas

    # protoc plugins
    # protoc-gen-go
    # protoc-gen-go-grpc
    # grpc-gateway # adds protoc-gen-grpc-gateway and protoc-gen-openapiv2 

    # # debugging
    # delve

    # other
    gnumake
  ];

  GOPRIVATE = "github.com/RafaySystems/*";
}
