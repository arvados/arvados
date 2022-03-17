// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MuiThemeProvider, createMuiTheme, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { CodeSnippet } from 'components/code-snippet/code-snippet';
import grey from '@material-ui/core/colors/grey';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'codeSnippet' | 'codeSnippetContainer';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    codeSnippet: {
    },
    codeSnippetContainer: {
        height: `calc(100% - ${theme.spacing.unit * 4}px)`, // so that horizontal scollbar is visible
    },
});

const theme = createMuiTheme({
    overrides: {
        MuiTypography: {
            body2: {
                color: grey["200"]
            },
            root: {
                backgroundColor: '#000'
            }
        }
    },
    typography: {
        fontFamily: 'monospace',
        useNextVariants: true,
    }
});

interface ProcessLogCodeSnippetProps {
    lines: string[];
}

export const ProcessLogCodeSnippet = withStyles(styles)(
    (props: ProcessLogCodeSnippetProps & WithStyles<CssRules>) =>
        <MuiThemeProvider theme={theme}>
            <CodeSnippet lines={props.lines} className={props.classes.codeSnippet}
                containerClassName={props.classes.codeSnippetContainer} />
        </MuiThemeProvider>);