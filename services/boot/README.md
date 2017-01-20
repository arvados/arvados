# gowebapp

A basic skeleton web application server. Just add HTML, client-side JavaScript code, and server-side APIs.

Strategy:
* In development, use npm to install JavaScript libraries.
* At build time, use webpack on nodejs to compile JavaScript assets.
* Deploy with a single Go binary -- no nodejs, no asset files.

## dev/build dependencies

Go:

```
curl https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz \
     | sudo tar -C /usr/local -xzf - \
     && (cd /usr/local/bin && sudo ln -s ../go/bin/* .)
```

nodejs:

```sh
curl -sL https://deb.nodesource.com/setup_6.x | sudo bash -
sudo apt-get install nodejs
```

## add/edit static files

Everything in the `static` directory will be served at `/`.

```sh
echo foo > static/foo.txt
# http://webapp/foo.txt
```

## add/edit javascript files

A webpack will be built using the entry point `js/index.js`, and served at `/js.js`.

```sh
echo 'function foo() { console.log("foo") }' > js/foo.js
echo 'require("./foo"); foo()'               > js/index.js
```

The default entry point and published location can be changed by editing `webpack.config.js`. For example, to build separate packs from `js/` and `js-admin/` source directories and serve them at `/user.js` and `/admin.js`:

```javascript
module.exports = {
    entry: {
        admin: './js-admin',
        user: './js'
    },
    ...
```

## generate before commit

To make your project `go get`able, run `go generate` before committing. This updates `bindata_assetfs.go`. Consider doing this in `.git/hooks/pre-commit` in case you forget.

If you don't need `go get` to work, and you prefer to keep generated files out of your source tree, you can:

```sh
git rm bindata_assetfs.go
echo bindata_assetfs.go >>.gitignore
git add .gitignore
git commit -m 'remove generated data'
```

In this case, your build pipeline must run `go generate` before `go build`.

## run dev-mode server

This runs webpack, updates bindata_assetfs.go with the new filesystem, builds a new Go binary, and runs it:

```sh
npm run dev
```

To use a port other than the default 8000:

```sh
PORT=8888 npm run dev
```

In dev mode, source maps are served, and JS is not minified.

After changing any source code (including static content), `^C` and run `npm run dev` again.

## run tests

Use nodejs to run JavaScript unit tests in `js/**/*_test.js` (see `js/example_test.js`).

```sh
npm test
```

Run Go tests the usual way.

```sh
go test ./...
```

## build production-mode server

```sh
npm build
```

The server binary will be installed to `$GOPATH/bin/`.

## build & run production-mode server

```sh
npm start
```

## TODO

* live dev mode with fsnotify and `webpack --watch -d`
* etags
