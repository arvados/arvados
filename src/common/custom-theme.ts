// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createMuiTheme } from '@material-ui/core/styles';
import { ThemeOptions, Theme } from '@material-ui/core/styles/createMuiTheme';
import purple from '@material-ui/core/colors/purple';
import blue from '@material-ui/core/colors/blue';
import grey from '@material-ui/core/colors/grey';

interface ArvadosThemeOptions extends ThemeOptions {
    customs: any;
}

export interface ArvadosTheme extends Theme {
    customs: any;
}

const purple900 = purple["900"];
const grey600 = grey["600"];
const themeOptions: ArvadosThemeOptions = {
    customs: {
        colors: {
            
        }
    },
    overrides: {
        MuiAppBar: {
            colorPrimary: {
                backgroundColor: purple900
            }
        },
        MuiTabs: {
            root: {
                color: grey600
            },
            indicator: {
                backgroundColor: purple900
            }
        },
        MuiTab: {
            selected: {
                fontWeight: 700,
                color: purple900
            }
        }
    },
    mixins: {
        toolbar: {
            minHeight: '48px'
        }
    },
    palette: {
        primary: {
            main: '#06C',
            dark: blue.A100
        }
    }
};

export const CustomTheme = createMuiTheme(themeOptions);