{
  description = "Folder bundler and server";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {
          inherit system;
        };
      in {
        formatter = pkgs.alejandra;
        packages = {
          default = self.outputs.packages.${system}.serverfi;
          serverfi = pkgs.buildGoModule rec {
            pname = "serverfi";
            version = "0.1.0";

            src = ./.;
            vendorHash = null;
            # We're doing horrible things, obviously safety's gotta be off
            allowGoReference = true;
            CGO_ENABLED = 0;

            buildPhase = ''
              cp -r ${pkgs.go}/share/go .
              ${pkgs.zip}/bin/zip -r go.zip . -i "./go/*"
              cp go.zip static/
              mkdir -p $out/bin
              go build -o $out/bin/serverfi main.go
            '';
          };
        };
      }
    );
}
