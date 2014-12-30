NPMBIN:=$(shell npm bin)

all: npmdeps build-dist

clean:
	rm -rvf dist

npmdeps:
	for x in bootstrap bower browserify chai chai-jquery jquery mithril mithril-query mocha mocha-phantomjs sinon; do [ -d "$(NPMBIN)/../$$x" ] || npm install $$x --save-dev; done

# This uses --debug to enable source maps.
build-dist:
	mkdir -p dist
	$(NPMBIN)/browserify --debug -o dist/app.js app/app.js
	$(NPMBIN)/browserify --debug -o dist/test.js test/runner.js
	rsync -a $(NPMBIN)/../bootstrap/dist/ dist/bootstrap/

server: build-dist
	[ -e $(NPMBIN)/harp ] || npm install harp
	$(NPMBIN)/harp server --port 9000
