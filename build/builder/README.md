# purpose

The arvados/builder image runs tests for a specified (recent or near future) version of arvados.

The image comes with most of the dependencies already installed, in order to reduce per-build time and network traffic.

(Most Ruby and Python dependencies can be updated at build time as needed. In that sense it's better, but usually not necessary, to keep builder up to date.)

# building

Make a docker image called "arvados/builder" suitable for using locally or pushing to dockerhub.

```
make builder
```

Build dependencies change over time, so sometimes it is necessary to build a new (or old) `arvados/builder` image that is compatible with the version of arvados you want to build.

Generate a new image:

```
git checkout master
make builder
```

Confirm the new image passes tests using its bundled version:

```
docker run -it arvados/builder
```

# using

Run the test suites using the source tree baked into the builder image itself (which might be out of date):

```
docker run -it arvados/builder
```

Run the test suites using a specific git commit (first fetching the latest commits from git.curoverse.com _unless_ the given commit is a full sha1 and is already present in the baked-in git history):

```
docker run -it arvados/builder master
```

# developer hacks

The focus of arvados/builder is running test suites in a well-defined environment in order to establish that a sha1-addressable version passes its tests.

There are some opportunities to use the builder image in other ways as part of your development cycle, though.

Run the test suites using a source tree on the host (which might have uncommitted local changes):

```
docker run -it -v /PATH/TO/LOCAL/ARVADOS:/src/arvados:ro arvados/builder
```

Add `-v /PATH/TO/LOCAL/CACHE:/tmp` to preserve installed bits between runs. For example, to test whether your `services/api` changes affect `services/arv-git-httpd` integration tests:

```
docker run -it -v /PATH/TO/LOCAL/CACHE:/tmp arvados/builder -v /HOST/SRC/ARVADOS:/src/arvados:ro "" --only install --only-install services/api
docker run -it -v /PATH/TO/LOCAL/CACHE:/tmp arvados/builder -v /HOST/SRC/ARVADOS:/src/arvados:ro "" --skip-install --only services/arv-git-httpd
```
