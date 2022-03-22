// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useRef, useState } from 'react';
import {
    MuiThemeProvider,
    createMuiTheme,
    StyleRulesCallback,
    withStyles,
    WithStyles
} from '@material-ui/core/styles';
import grey from '@material-ui/core/colors/grey';
import { ArvadosTheme } from 'common/custom-theme';
import { Link, Typography } from '@material-ui/core';
import { navigateTo } from 'store/navigation/navigation-action';
import { Dispatch } from 'redux';
import { connect, DispatchProp } from 'react-redux';
import classNames from 'classnames';

type CssRules = 'root' | 'wordWrap' | 'logText';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        overflow: 'auto',
        backgroundColor: '#000',
        height: `calc(100% - ${theme.spacing.unit * 4}px)`, // so that horizontal scollbar is visible
    },
    logText: {
        padding: theme.spacing.unit * 0.5,
    },
    wordWrap: {
        whiteSpace: 'pre-wrap',
    },
});

const theme = createMuiTheme({
    overrides: {
        MuiTypography: {
            body2: {
                color: grey["200"]
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
    ({classes, lines, fontSize, dispatch, wordWrap}: ProcessLogCodeSnippetProps & WithStyles<CssRules> & DispatchProp) => {
        const [followMode, setFollowMode] = useState<boolean>(false);
        const scrollRef = useRef<HTMLDivElement>(null);

        useEffect(() => {
            if (followMode && scrollRef.current && lines.length > 0) {
                // Scroll to bottom
                scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
            }
        }, [followMode, lines, scrollRef]);

        return <MuiThemeProvider theme={theme}>
            <div ref={scrollRef} className={classes.root}
                onScroll={(e) => {
                    const elem = e.target as HTMLDivElement;
                    if (elem.scrollTop + elem.clientHeight >= elem.scrollHeight) {
                        setFollowMode(true);
                    } else {
                        setFollowMode(false);
                    }
                }}>
                { lines.map((line: string, index: number) =>
                <Typography key={index} component="pre"
                    className={classNames(classes.logText, wordWrap ? classes.wordWrap : undefined)}>
                    {renderLinks(fontSize, dispatch)(line)}
                </Typography>
                ) }
            </div>
        </MuiThemeProvider>
    }));