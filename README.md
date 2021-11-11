[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Arvados Workbench 2

## Setup
```
npm install yarn
yarn install
```

Install [redux-devtools-extension](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd)

## Start project for development
```
yarn start
```

## Start project for development inside Docker container

```
make workbench2-build-image
# (create public/config.json, see "Run time configuration" below)
docker run -ti -v$PWD:$PWD -p 3000:3000 -w$PWD workbench2-build /bin/bash
# (inside docker container)
yarn install
yarn start
```

## Run unit tests
```
make unit-tests
```

## Run end-to-end tests

```
make integration-tests
```

## Run end-to-end tests in a Docker container

```
make integration-tests-in-docker
```

## Run tests interactively in container

```
xhost +local:root
ARVADOS_DIR=/path/to/arvados
docker run -ti -v$PWD:$PWD -v$ARVADOS_DIR:/usr/src/arvados -w$PWD --env="DISPLAY" --volume="/tmp/.X11-unix:/tmp/.X11-unix:rw" workbench2-build /bin/bash
(inside container)
yarn run cypress install
tools/run-integration-tests.sh -i -a /usr/src/arvados
```

## Production build
```
yarn build
```

## Package build
```
make packages
```

## Build time configuration
You can customize project global variables using env variables. Default values are placed in the `.env` file.

Example:
```
REACT_APP_ARVADOS_CONFIG_URL=config.json yarn build
```

## Run time configuration
The app will fetch runtime configuration when starting. By default it will try to fetch `/config.json`.  In development mode, this can be found in the `public` directory.
You can customize this url using build time configuration.

Currently this configuration schema is supported:
```
{
    "API_HOST": "string",
    "FILE_VIEWERS_CONFIG_URL": "string",
}
```

### API_HOST

The Arvados base URL.

The `REACT_APP_ARVADOS_API_HOST` environment variable can be used to set the default URL if the run time configuration is unreachable.

## FILE_VIEWERS_CONFIG_URL
Local path, or any URL that allows cross-origin requests. See:

[File viewers config file example](public/file-viewers-example.json)

[File viewers config scheme](src/models/file-viewers-config.ts)

To use the URL defined in the Arvados cluster configuration, remove the entire `FILE_VIEWERS_CONFIG_URL` entry from the runtime configuration. Found in `/config.json` by default.

## Plugin support

Workbench supports plugins to add new functionality to the user
interface.  For information about installing plugins, the provided
example plugins, see [src/plugins/README.md](src/plugins/README.md).


## Licensing

Arvados is Free Software. See COPYING for information about Arvados Free
Software licenses.
