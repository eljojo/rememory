{
  description = "ReMemory - a digital safe with multiple keys, held by people you trust";

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

        versionFile = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);

        rememory = pkgs.buildGoModule {
          pname = "rememory";
          version = versionFile;
          src = ./.;

          vendorHash = "sha256-NmfFjRZASEzIXnvJ3yU0LrudG/wFUiF+37myhn1GjNM=";
          proxyVendor = true; # Download deps during build instead of vendoring

          # The go-modules derivation only fetches Go deps — skip TS/WASM build there
          overrideModAttrs = old: {
            preBuild = null;
          };

          nativeBuildInputs = [ pkgs.esbuild pkgs.gnumake pkgs.nodejs pkgs.cacert ];

          npmDeps = pkgs.fetchNpmDeps {
            src = ./.;
            hash = "sha256-xpvOTAZb2lpMQV9IK5yOMqvgJtcPAHspt9OHAufFKtc=";
          };

          # Patch go.mod to match nixpkgs Go version (nixpkgs may lag behind)
          prePatch = ''
            sed -i "s/^go .*/go ${pkgs.go.version}/" go.mod
          '';

          # Install npm deps and build TypeScript + WASM
          preBuild = ''
            export HOME=$TMPDIR
            npm config set cache "$npmDeps"
            npm ci --ignore-scripts --prefer-offline
            # Remove broken esbuild from node_modules — npm ci --ignore-scripts
            # skips the postinstall that downloads the platform binary, leaving a
            # broken wrapper that shadows the working esbuild from nativeBuildInputs.
            rm -f node_modules/.bin/esbuild
            export PATH="$PWD/node_modules/.bin:$PATH"
            make wasm
          '';

          # Generate and install man pages
          postInstall = ''
            mkdir -p $out/share/man/man1
            $out/bin/rememory doc $out/share/man/man1
          '';

          subPackages = [ "cmd/rememory" ];

          ldflags = [ "-s" "-w" "-X main.version=${versionFile}" ];
        };

      in
      {
        packages = {
          rememory = rememory;
          default = rememory;

          docker = pkgs.dockerTools.buildImage {
            name = "rememory";
            tag = "latest";
            copyToRoot = pkgs.buildEnv {
              name = "rememory-root";
              paths = [ rememory ];
            };
            config = {
              Cmd = [ "${rememory}/bin/rememory" "serve" "--host" "0.0.0.0" "--port" "8080" "--data" "/data" ];
              ExposedPorts = { "8080/tcp" = { }; };
              Volumes = { "/data" = { }; };
            };
          };

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

        apps = {
          rememory = flake-utils.lib.mkApp { drv = rememory; };
          default = flake-utils.lib.mkApp { drv = rememory; };
        };

        checks = {
          go-tests = rememory;
          # e2e-tests = self.packages.${system}.e2e-tests; # Enable after setup
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.nodejs
            pkgs.esbuild
            pkgs.playwright-test
            pkgs.poppler-utils
          ];
          shellHook = ''
            export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
            export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
          '';
        };
      }
    );
}
