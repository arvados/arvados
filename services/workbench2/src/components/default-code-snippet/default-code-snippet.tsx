// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MuiThemeProvider, createMuiTheme } from '@material-ui/core/styles';
import { CodeSnippet, CodeSnippetDataProps } from 'components/code-snippet/code-snippet';
import grey from '@material-ui/core/colors/grey';
import { themeOptions } from 'common/custom-theme';

const theme = createMuiTheme(Object.assign({}, themeOptions, {
    overrides: {
        MuiTypography: {
            body1: {
                color: grey["900"]
            },
            root: {
                backgroundColor: grey["200"]
            }
        }
    },
    typography: {
        fontFamily: 'monospace',
        useNextVariants: true,
    }
}));

export const DefaultCodeSnippet = (props: CodeSnippetDataProps) =>
    <MuiThemeProvider theme={theme}>
        <CodeSnippet {...props} />
    </MuiThemeProvider>;
