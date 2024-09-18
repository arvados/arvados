// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ThemeProvider, Theme, StyledEngineProvider, createTheme, adaptV4Theme } from '@mui/material/styles';
import { CodeSnippet, CodeSnippetDataProps } from 'components/code-snippet/code-snippet';
import { themeOptions } from 'common/custom-theme';
import { grey } from '@mui/material/colors';


declare module '@mui/styles/defaultTheme' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  interface DefaultTheme extends Theme {}
}


const theme = createTheme(adaptV4Theme(Object.assign({}, themeOptions, {
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
    }
})));

export const DefaultCodeSnippet = (props: CodeSnippetDataProps) =>
    <StyledEngineProvider injectFirst>
        <ThemeProvider theme={theme}>
            <CodeSnippet {...props} />
        </ThemeProvider>
    </StyledEngineProvider>;
