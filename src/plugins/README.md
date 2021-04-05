[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Plugin support

Workbench supports plugins to add new functionality to the user
interface.  It is also possible to remove the majority of standard UI
elements and replace them with your own, enabling you to use workbench
as a basis for developing essentially new applications for Arvados.

## Installing plugins

1. Check out the source of your plugin into a directory under `arvados-workbench2/src/plugins`

2. Register the plugin by editing `arvados-workbench2/src/plugins/plugins.tsx`.
It will look something like this:

```
import { register as examplePluginRegister } from '~/plugins/example/index';
examplePluginRegister(pluginConfig);
```

3. Rebuild Workbench 2

For testing/development: `yarn start`

For production: `APP_NAME=arvados-workbench2-with-custom-plugins make packages`

Set `APP_NAME=` to whatever you like, but it is important to name it
differently from the standard `arvados-workbench2` to avoid confusion.

## Existing plugins

### example

This is an example plugin showing how to add a new navigation tree
item, displaying a new center panel, as well as adding account menu
and "New" menu items, and showing how to use SET_PROPERTY and
getProperty() for state.

### blank

This deletes all of the existing user interface.  If you want the
application to only display your plugin's UI elements and none of the
standard elements, you would load and register this first.

### root-redirect

This helper takes a path when registered.  It tweaks the navigation
behavior so that the default starting location when the application
loads will be the path you provide, instead of "Projects".

### sample-tracker

This is a a new set of user interface screens that assist with
clinical sample tracking and analysis.  It is intended as a demo of
how a real-world application can built using the Workbench 2
plug-in interface.  It can be found at
https://github.com/arvados/sample-tracker .

## Developing plugins

For information about the plugin API, see
[../common/plugintypes.ts](src/common/plugintypes.ts).
