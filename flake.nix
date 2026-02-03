{
  description = "ReMemory - encrypt secrets and split access among trusted friends";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    playwright.url = "github:pietdevries94/playwright-web-flake";
    playwright.inputs.nixpkgs.follows = "nixpkgs";
    playwright.inputs.flake-utils.follows = "flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, playwright }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (final: prev: {
              inherit (playwright.packages.${system}) playwright-test playwright-driver;
            })
          ];
        };

        rememory = pkgs.buildGoModule {
          pname = "rememory";
          version = "0.1.0";
          src = ./.;

          vendorHash = null; # Uses go.sum

          # Build WASM first
          preBuild = ''
            mkdir -p internal/html/assets
            GOOS=js GOARCH=wasm go build -o internal/html/assets/recover.wasm ./internal/wasm
            cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
            cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || true
          '';

          subPackages = [ "cmd/rememory" ];

          ldflags = [ "-s" "-w" "-X main.version=${self.shortRev or "dev"}" ];
        };

      in
      {
        packages = {
          default = rememory;

          e2e-tests = pkgs.buildNpmPackage {
            pname = "rememory-e2e";
            version = "1.0.0";
            src = ./.;

            npmDepsHash = pkgs.lib.fakeHash; # Update after first build

            nativeBuildInputs = [
              rememory
              pkgs.playwright-test
              pkgs.playwright-driver
            ];

            env = {
              PLAYWRIGHT_BROWSERS_PATH = "${pkgs.playwright-driver.browsers}";
              PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1";
            };

            dontNpmBuild = true;

            buildPhase = ''
              # Remove npm-installed playwright to avoid conflicts
              rm -rf node_modules/@playwright node_modules/.bin/playwright 2>/dev/null || true
              mkdir -p node_modules/.bin
              ln -s ${pkgs.playwright-test}/bin/playwright node_modules/.bin/playwright

              # Create test fixtures
              ln -s ${rememory}/bin/rememory rememory

              echo "Running Playwright E2E tests..."
              ${pkgs.playwright-test}/bin/playwright test
            '';

            installPhase = ''
              mkdir -p $out
              if [ -d e2e/playwright-report ]; then
                cp -r e2e/playwright-report/* $out/
              fi
            '';
          };
        };

        checks = {
          go-tests = rememory;
          # e2e-tests = self.packages.${system}.e2e-tests; # Enable after setup
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.nodejs
            pkgs.playwright-test
          ];
          shellHook = ''
            export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
            export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
          '';
        };
      }
    );
}
