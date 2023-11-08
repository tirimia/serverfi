{
  description = "Folder bundler and server";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  /*
  * Initial plan: hijack the bootstrapping process of GO to build my own version
  * learned how self-hosted languages work from watching enough tsoding
  * ******************
  * Second idea: use tinygo's internal libraries to build the binary
  * ******************
  * Third iteration: use nix to build tinygo, embed it in serverfi,
  * write it out to a file on run, exec it, clean up
  * Turns out some essential libraries don't work with tinygo
  * ******************
  * Fourth iteration: use nix to embed go itself in serverfi
  */
  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      imports = [];
      systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin"];
      perSystem = {
        config,
        self',
        inputs',
        pkgs,
        system,
        lib,
        ...
      }: {
        formatter = pkgs.alejandra;
        packages.default = let
          go-compiler = pkgs.go_1_21;
        in
          pkgs.buildGoModule rec {
            pname = "serverfi";
            version = "0.1.0";

            src = ./.;
            vendorHash = null;
            # We're doing horrible things, obviously safety's gotta be off
            allowGoReference = true;
            CGO_ENABLED = 0;

            buildPhase = ''
              cp -r ${go-compiler}/share/go .
              ${pkgs.zip}/bin/zip -r go.zip . -i "./go/*"
              cp go.zip static/
              mkdir -p $out/bin
              go build -o $out/bin/serverfi main.go
            '';
          };
      };
    };
}
