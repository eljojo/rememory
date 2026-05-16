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

        npmDeps = pkgs.fetchNpmDeps {
          src = ./.;
          hash = "sha256-t4PWD3/YZZx82useT3YPGVRC/19noCmyfp6lLm47Lso=";
        };

        # Build TypeScript + WASM assets with the native toolchain. These are
        # architecture-independent and can be reused across cross builds.
        # Uses buildGoModule so Go dependencies are available in the sandbox.
        wasmAssets = pkgs.buildGoModule {
          pname = "rememory-wasm-assets";
          version = versionFile;
          src = ./.;

          vendorHash = "sha256-XR3XoBcyEou/L3QXGBUl8IS1u0lpEBS+GTrD574OWh8=";
          proxyVendor = true;

          overrideModAttrs = old: {
            preBuild = null;
          };

          nativeBuildInputs = [ pkgs.esbuild pkgs.gnumake pkgs.nodejs pkgs.cacert ];

          inherit npmDeps;

          # Patch go.mod to match nixpkgs Go version (nixpkgs may lag behind)
          prePatch = ''
            sed -i "s/^go .*/go ${pkgs.go.version}/" go.mod
          '';

          # Build only the WASM and TypeScript — skip the native Go binary
          preBuild = ''
            export HOME=$TMPDIR
            npm config set cache "$npmDeps"
            npm ci --ignore-scripts --prefer-offline
            rm -f node_modules/.bin/esbuild
            export PATH="$PWD/node_modules/.bin:$PATH"
            make wasm
          '';

          # Skip the normal Go build — we only want the WASM/JS artifacts
          buildPhase = ''
            runHook preBuild
          '';

          installPhase = ''
            mkdir -p $out
            cp internal/html/assets/*.js internal/html/assets/*.wasm $out/
          '';
        };

        # Shared builder for the rememory binary. Accepts a target Go package set
        # (for cross-compilation) and optional pre-built WASM assets to avoid
        # rebuilding them with a cross compiler that can't target js/wasm.
        mkRememory = { goPkgs ? pkgs, enableManPages ? true, prebuiltAssets ? null }:
          goPkgs.buildGoModule {
            pname = "rememory";
            version = versionFile;
            src = ./.;

            vendorHash = "sha256-XR3XoBcyEou/L3QXGBUl8IS1u0lpEBS+GTrD574OWh8=";
            proxyVendor = true; # Download deps during build instead of vendoring

            # The go-modules derivation only fetches Go deps — skip TS/WASM build there
            overrideModAttrs = old: {
              preBuild = null;
            };

            nativeBuildInputs = [ pkgs.esbuild pkgs.gnumake pkgs.nodejs pkgs.cacert ];

            inherit npmDeps;

            # Patch go.mod to match nixpkgs Go version (nixpkgs may lag behind)
            prePatch = ''
              sed -i "s/^go .*/go ${goPkgs.go.version}/" go.mod
            '';

            # Install npm deps and build TypeScript + WASM.
            # For cross builds, pre-built assets are copied in instead of running
            # `make wasm`, because the cross Go linker can't target js/wasm.
            preBuild = if prebuiltAssets != null then ''
              cp ${prebuiltAssets}/*.js internal/html/assets/
              cp ${prebuiltAssets}/*.wasm internal/html/assets/
            '' else ''
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

            # Generate and install man pages (only for native builds —
            # cross-compiled binaries can't run on the build host)
            postInstall = pkgs.lib.optionalString enableManPages ''
              mkdir -p $out/share/man/man1
              $out/bin/rememory doc $out/share/man/man1
            '';

            subPackages = [ "cmd/rememory" ];

            ldflags = [ "-s" "-w" "-X main.version=${versionFile}" ];
          };

        rememory = mkRememory { };

        mkDockerImage = { rememoryPkg, tag ? "latest", arch ? null }:
          pkgs.dockerTools.buildImage ({
            name = "rememory";
            inherit tag;
            copyToRoot = pkgs.buildEnv {
              name = "rememory-root";
              paths = [
                rememoryPkg
                pkgs.dockerTools.fakeNss
              ];
            };
            runAsRoot = ''
              mkdir -p /data
              chown 65534:65534 /data
            '';
            config = {
              Cmd = [ "${rememoryPkg}/bin/rememory" "serve" "--host" "0.0.0.0" "--port" "8080" "--data" "/data" ];
              ExposedPorts = { "8080/tcp" = { }; };
              Volumes = { "/data" = { }; };
              User = "65534:65534";
            };
          } // pkgs.lib.optionalAttrs (arch != null) { architecture = arch; });

        # Cross-compiled arm64 packages (only available on x86_64-linux,
        # where CI runs — used to produce multi-arch Docker images)
        crossPackages = pkgs.lib.optionalAttrs (system == "x86_64-linux") (
          let
            pkgsCrossArm64 = import nixpkgs {
              localSystem = "x86_64-linux";
              crossSystem = "aarch64-linux";
            };
            rememory-arm64 = mkRememory {
              goPkgs = pkgsCrossArm64;
              enableManPages = false;
              prebuiltAssets = wasmAssets;
            };
          in {
            rememory-arm64 = rememory-arm64;
            docker-arm64 = mkDockerImage {
              rememoryPkg = rememory-arm64;
              tag = "latest-arm64";
              arch = "arm64";
            };
          }
        );

      in
      {
        packages = {
          rememory = rememory;
          default = rememory;

          docker = mkDockerImage { rememoryPkg = rememory; };

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
        } // crossPackages;

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
