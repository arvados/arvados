# Install

## Prerequisites

```
sudo apt-get install nodejs
```

## Install dependencies and build dist files

```
make
```

# Develop

## Update dist files and run a dev server

```
make server
```

## Update dependencies

```
npm update && make
```

# Test

## Run test suite

This uses mocha, node.js, and jsdom.

```
make test
```

Run mocha in "watch" mode to re-run tests whenever you change a source file.

```
make test-watch
```

## Run test suite using phantomjs

```
make test-phantomjs
```

## Run test suite using a real browser

```
make server
```

Point your browser at http://localhost:9000/test.html
