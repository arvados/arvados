NPMBIN:=$(shell npm bin)

all: npmdeps build

clean:
	rm -rvf dist

npmdeps:
	ulimit -n 4000; for x in bootstrap bower browserify chai chai-jquery jquery jsdom lodash mithril mithril-query mocha mocha-phantomjs sinon; do [ -d "$(NPMBIN)/../$$x" ] || npm install $$x --save-dev; done

# This uses --debug to enable source maps.
build: build-app build-test build-assets
build-app:
	mkdir -p dist
	ulimit -n 4000; $(NPMBIN)/browserify --debug -o dist/app.js app/app.js
build-test:
	mkdir -p dist
	ulimit -n 4000; $(NPMBIN)/browserify --debug -o dist/test.js test/runner.js
build-assets: dist/bootstrap
dist/bootstrap:
	mkdir -p dist
	rsync -a $(NPMBIN)/../bootstrap/dist/ dist/bootstrap/

server: build
	[ -e $(NPMBIN)/harp ] || npm install harp
	$(NPMBIN)/harp server --port 9000

test-phantomjs: build-test build-assets
	$(NPMBIN)/mocha-phantomjs test.html
test-watch:
	$(NPMBIN)/mocha --watch
.PHONY: test
test:
	$(NPMBIN)/mocha
