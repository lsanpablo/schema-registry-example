{
  description = "Event Registry tools";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";

  outputs = { self, nixpkgs }: let
    forAllSystems = f: nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-darwin" ] (system:
      f {
        pkgs = nixpkgs.legacyPackages.${system};
        system = system;
      });
  in {
    packages = forAllSystems ({ pkgs, system }: {
      event-template = pkgs.buildGoModule {
        pname = "event-template";
        version = "0.1.0";
        src = self + /tools/event-template;
        vendorHash = null;
      };

      generate-index = pkgs.buildGoModule {
              pname = "generate-index";
              version = "0.1.0";
              src = self + /tools/generate-index;
              vendorHash = null;
      };
      generate-go-types = pkgs.buildGoModule {
              pname = "generate-go-types";
              version = "0.1.0";
              src = self + /tools/generate-go-types;
              vendorHash = null;
      };

    });

    devShells = forAllSystems ({ pkgs, ... }: {
      default = pkgs.mkShell {
        packages = [
        pkgs.protobuf_23
        pkgs.protoc-gen-go
        pkgs.protoc-gen-go-grpc
          self.packages.${pkgs.system}.event-template
          self.packages.${pkgs.system}.generate-index
          self.packages.${pkgs.system}.generate-go-types
        ];
      };
    });
  };
}
