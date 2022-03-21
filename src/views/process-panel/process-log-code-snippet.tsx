// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MuiThemeProvider, createMuiTheme, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { CodeSnippet } from 'components/code-snippet/code-snippet';
import grey from '@material-ui/core/colors/grey';
import { ArvadosTheme } from 'common/custom-theme';
import { Link, Typography } from '@material-ui/core';
import { navigateTo } from 'store/navigation/navigation-action';
import { Dispatch } from 'redux';
import { connect, DispatchProp } from 'react-redux';

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

const renderLinks = (fontSize: number, dispatch: Dispatch) => (text: string) => {
    // Matches UUIDs & PDHs
    const REGEX = /[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}|[0-9a-f]{32}\+\d+/g;
    const links = text.match(REGEX);
    if (!links) {
        return <Typography style={{ fontSize: fontSize }}>{text}</Typography>;
    }
    return <Typography style={{ fontSize: fontSize }}>
        {text.split(REGEX).map((part, index) =>
        <React.Fragment key={index}>
            {part}
            {links[index] &&
            <Link onClick={() => dispatch<any>(navigateTo(links[index]))}
                style={ {cursor: 'pointer'} }>
                {links[index]}
            </Link>}
        </React.Fragment>
        )}
    </Typography>;
};

export const ProcessLogCodeSnippet = withStyles(styles)(connect()(
    (props: ProcessLogCodeSnippetProps & WithStyles<CssRules> & DispatchProp) =>
        <MuiThemeProvider theme={theme}>
            <CodeSnippet lines={props.lines} fontSize={props.fontSize}
                customRenderer={renderLinks(props.fontSize, props.dispatch)}
                className={props.wordWrap ? props.classes.wordWrap : undefined}
                containerClassName={props.classes.codeSnippetContainer} />
        </MuiThemeProvider>));