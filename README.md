[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

## Arvados Workbench 2

### Setup
<pre>
brew install yarn
yarn install
</pre>
Install [redux-devtools-extension](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd)

### Start project
<code>yarn start</code>

### Run tests
<pre>
yarn install
yarn test
</pre>

### Production build
<pre>
yarn install
yarn build
</pre>

### Build time configuration
You can customize project global variables using env variables. Default values are placed in the `.env` file.

Example:
```
REACT_APP_ARVADOS_CONFIG_URL=config.json yarn build
```

### Run time configuration
The app will fetch runtime configuration when starting. By default it will try to fetch `/config.json`. You can customize this url using build time configuration.

Currently this configuration schema is supported:
```
{
    "API_HOST": "string"
}
```

### Licensing

Arvados is Free Software. See COPYING for information about Arvados Free
Software licenses.
