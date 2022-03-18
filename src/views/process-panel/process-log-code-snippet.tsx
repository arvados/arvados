// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MuiThemeProvider, createMuiTheme, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { CodeSnippet } from 'components/code-snippet/code-snippet';
import grey from '@material-ui/core/colors/grey';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'wordWrap' | 'codeSnippetContainer';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    wordWrap: {
        whiteSpace: 'pre-wrap',
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
    fontSize: number;
    wordWrap?: boolean;
}

export const ProcessLogCodeSnippet = withStyles(styles)(
    (props: ProcessLogCodeSnippetProps & WithStyles<CssRules>) =>
        <MuiThemeProvider theme={theme}>
            <CodeSnippet lines={props.lines} fontSize={props.fontSize}
                className={props.wordWrap ? props.classes.wordWrap : undefined}
                containerClassName={props.classes.codeSnippetContainer} />
        </MuiThemeProvider>);