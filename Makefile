JAVY ?= bin/javy

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
  ifeq ($(UNAME_M),arm64)
    JAVY_PATTERN := javy-arm-macos-*.gz
  else
    JAVY_PATTERN := javy-x86_64-macos-*.gz
  endif
else
  ifeq ($(UNAME_M),aarch64)
    JAVY_PATTERN := javy-arm-linux-*.gz
  else
    JAVY_PATTERN := javy-x86_64-linux-*.gz
  endif
endif

# NOTE: uses `shasum -a 256` (macOS/BSD + most Linux distros ship this via
# perl-Digest-SHA). If `shasum` is unavailable on a future Linux CI image,
# swap in `sha256sum` (e.g. `sha256sum js/dist/flint-javy.js | cut -d' ' -f1)`).
.PHONY: wasm
wasm: $(JAVY)
	cd js && npm run build
	$(JAVY) build -o compile/flint.wasm js/dist/flint-javy.js
	@printf 'javy: %s\nflint-chart: %s\nbundle-sha256: %s\nbundle-bytes: %s\nwasm-sha256: %s\n' \
	  "$$($(JAVY) --version)" \
	  "$$(cd js && node -p "JSON.parse(require('fs').readFileSync('node_modules/flint-chart/package.json','utf8')).version")" \
	  "$$(shasum -a 256 js/dist/flint-javy.js | cut -d' ' -f1)" \
	  "$$(wc -c < js/dist/flint-javy.js | tr -d ' ')" \
	  "$$(shasum -a 256 compile/flint.wasm | cut -d' ' -f1)" \
	  > compile/flint.wasm.buildinfo
	@ls -la compile/flint.wasm
	@cat compile/flint.wasm.buildinfo

$(JAVY):
	mkdir -p bin
	gh release download -R bytecodealliance/javy --pattern "$(JAVY_PATTERN)" -O bin/javy.gz --clobber
	gunzip -f bin/javy.gz
	chmod +x bin/javy

.PHONY: browser-shim
browser-shim:
	$(eval FLINT_VER := $(shell cd js && node -e 'const fs=require("fs");console.log(JSON.parse(fs.readFileSync("node_modules/flint-chart/package.json")).version)'))
	cd js && npx esbuild src/entry-browser.ts --bundle --format=esm --platform=browser --target=es2020 --outfile=dist/flintchart-shim.mjs --define:__FLINTCHART_VERSION__='"flint-chart@$(FLINT_VER)"'
	@printf 'source: github.com/NimbleMarkets/flint-ntcharts (local)\ncommit: %s\nentry: js/src/entry-browser.ts\nflint-chart: %s\nsha256: %s\nbytes: %s\n' \
	  "$$(git rev-parse HEAD)" \
	  "$(FLINT_VER)" \
	  "$$(shasum -a 256 js/dist/flintchart-shim.mjs | cut -d' ' -f1)" \
	  "$$(wc -c < js/dist/flintchart-shim.mjs | tr -d ' ')" \
	  > js/dist/flintchart-shim.PROVENANCE.txt
	@cat js/dist/flintchart-shim.PROVENANCE.txt
# Note: `npm run build:browser` (js/package.json) is for dev builds only and
# stamps a fixed "dev" version; this Makefile target is the one used for
# vendoring/release, and stamps the real flint-chart package version above.
