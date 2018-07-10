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

### Configuration
You can customize project global variables using env variables. Default values are placed in the `.env` file.

Example:
```
REACT_APP_ARVADOS_API_HOST=localhost:8000 yarn start
```

### Licensing

Arvados is Free Software. See COPYING for information about Arvados Free
Software licenses.
