{
  description = "Claude Code Ntfy - Transparent wrapper for Claude Code with notifications";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        claude-code-ntfy = pkgs.buildGoModule rec {
          pname = "claude-code-ntfy";
          version = "0.1.0";
          
          src = ./.;
          
          vendorHash = "sha256-T0KmN00q6lGo8LHKiBdmSveSMSLykCWi1pxCqkqE1Rs=";
          
          preBuild = ''
            echo "=== Build environment debug ==="
            echo "pwd: $(pwd)"
            echo "GOOS: $GOOS"
            echo "GOARCH: $GOARCH"
            echo "Contents of cmd/claude-code-ntfy/:"
            ls -la cmd/claude-code-ntfy/
            echo "Go version:"
            go version
            echo "Go env GOOS/GOARCH:"
            go env GOOS GOARCH
            echo "=== End debug ==="
          '';
          
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];
          
          # Create a direct claude symlink/wrapper
          postInstall = ''
            # Rename the binary to claude so it can be called directly
            mv $out/bin/claude-code-ntfy $out/bin/claude
          '';
          
          meta = with pkgs.lib; {
            description = "Transparent wrapper for Claude Code with ntfy.sh notifications";
            homepage = "https://github.com/Veraticus/claude-code-ntfy";
            license = licenses.mit;
            maintainers = [ ];
            platforms = platforms.linux ++ platforms.darwin;
          };
        };
      in
      {
        packages = {
          default = claude-code-ntfy;
          claude-code-ntfy = claude-code-ntfy;
        };
        
        apps.default = flake-utils.lib.mkApp {
          drv = claude-code-ntfy;
          name = "claude";
        };
        
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_21
            gnumake
            golangci-lint
            gopls
            gotools
          ];
          
          shellHook = ''
            echo "Claude Code Ntfy development environment"
            echo "Run 'make build' to build the project"
            echo "Run 'make test' to run tests"
            echo "Run 'make help' to see all available commands"
          '';
        };
      }
    ) // {
      # System-agnostic outputs
      overlays.default = final: prev: {
        claude-code-ntfy = self.packages.${final.system}.default;
      };

      # NixOS module for system-wide installation
      nixosModules = {
        default = { config, lib, pkgs, ... }:
          with lib;
          let
            cfg = config.programs.claude-code-ntfy;
          in
          {
            options.programs.claude-code-ntfy = {
              enable = mkEnableOption (lib.mdDoc ''
                Claude Code Ntfy wrapper.
                
                This installs a wrapper around Claude Code that monitors output
                and sends notifications via ntfy.sh based on configurable patterns.
                
                Note: Claude Code must be installed separately via npm.
              '');
              
              package = mkOption {
                type = types.package;
                default = self.packages.${pkgs.system}.default;
                defaultText = literalExpression "pkgs.claude-code-ntfy";
                description = lib.mdDoc "The claude-code-ntfy package to use";
              };
            };
            
            config = mkIf cfg.enable {
              environment.systemPackages = [ cfg.package ];
              
              # Ensure the wrapper has higher priority than npm claude
              environment.pathsToLink = [ "/bin" ];
              environment.extraInit = ''
                # Ensure our claude wrapper comes first in PATH
                export PATH="${cfg.package}/bin:$PATH"
              '';
            };
          };
      };

      # Home Manager module
      homeManagerModules = {
        default = { config, lib, pkgs, ... }:
          with lib;
          let
            cfg = config.programs.claude-code-ntfy;
          in
          {
            options.programs.claude-code-ntfy = {
              enable = mkEnableOption (lib.mdDoc ''
                Claude Code Ntfy wrapper.
                
                This installs a wrapper around Claude Code that monitors output
                and sends notifications via ntfy.sh based on configurable patterns.
                
                Note: Claude Code must be installed separately via npm.
              '');
              
              package = mkOption {
                type = types.package;
                default = self.packages.${pkgs.system}.default;
                defaultText = literalExpression "pkgs.claude-code-ntfy";
                description = lib.mdDoc "The claude-code-ntfy package to use";
              };

              settings = mkOption {
                type = types.attrs;
                default = {};
                example = literalExpression ''
                  {
                    ntfy_topic = "my-claude-notifications";
                    ntfy_server = "https://ntfy.sh";
                    idle_timeout = "2m";
                    patterns = [
                      {
                        name = "error";
                        regex = "(?i)(error|failed|exception)";
                        enabled = true;
                      }
                    ];
                  }
                '';
                description = lib.mdDoc ''
                  Configuration for claude-code-ntfy.
                  This will be written to ~/.config/claude-code-ntfy/config.yaml
                '';
              };
            };
            
            config = mkIf cfg.enable {
              home.packages = [ cfg.package ];
              
              # Create config file if settings are provided
              xdg.configFile."claude-code-ntfy/config.yaml" = mkIf (cfg.settings != {}) {
                text = pkgs.lib.generators.toYAML {} cfg.settings;
              };
              
              # Ensure our wrapper comes first in PATH
              home.sessionPath = [ "${cfg.package}/bin" ];
              home.sessionVariables = {
                # Prepend to PATH to ensure our wrapper is found first
                PATH = "${cfg.package}/bin:$PATH";
              };
            };
          };
      };
    };
}
