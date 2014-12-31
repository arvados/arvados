NPMBIN:=$(shell npm bin)

all: npmdeps build

clean:
	rm -rvf dist

npmdeps:
	for x in bootstrap bower browserify chai chai-jquery jquery mithril mithril-query mocha mocha-phantomjs sinon; do [ -d "$(NPMBIN)/../$$x" ] || npm install $$x --save-dev; done

# This uses --debug to enable source maps.
build: build-app build-test build-css
build-app:
	mkdir -p dist
	$(NPMBIN)/browserify --debug -o dist/app.js app/app.js
build-test:
	mkdir -p dist
	$(NPMBIN)/browserify --debug -o dist/test.js test/runner.js
build-css:
	mkdir -p dist
	rsync -a $(NPMBIN)/../bootstrap/dist/ dist/bootstrap/

server: build
	[ -e $(NPMBIN)/harp ] || npm install harp
	$(NPMBIN)/harp server --port 9000

test: build-test build-css
	$(NPMBIN)/mocha-phantomjs test.html
