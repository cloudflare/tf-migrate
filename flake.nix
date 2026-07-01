{
  description = "Cloudflare tf-migrate based on gomod2nix";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      gomod2nix,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
        };
      in
      {
        packages.default = pkgs.buildGoApplication {
          pname = "tf-migrate";
          version = "1.0.1";
          src = ./.;
          # gomod2nix generated this file from go.mod/go.sum
          modules = ./gomod2nix.toml;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            gomod2nix.packages.${system}.default
          ];
        };
      }
    );

}
