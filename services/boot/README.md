# arvados-boot

Coordinates Arvados system services.

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
