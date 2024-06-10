// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useRef, useState } from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {
    MuiThemeProvider,
    createTheme,
    withStyles,
    WithStyles
} from '@material-ui/core/styles';
import grey from '@material-ui/core/colors/grey';
import { ArvadosTheme } from 'common/custom-theme';
import { Link, Typography } from '@material-ui/core';
import { navigationNotAvailable } from 'store/navigation/navigation-action';
import { Dispatch } from 'redux';
import { connect, DispatchProp } from 'react-redux';
import classNames from 'classnames';
import { FederationConfig, getNavUrl } from 'routes/routes';
import { RootState } from 'store/store';

type CssRules = 'root' | 'wordWrapOn' | 'wordWrapOff' | 'logText';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        overflow: 'auto',
        backgroundColor: '#000',
        height: `calc(100% - ${theme.spacing(4)}px)`, // so that horizontal scollbar is visible
        "& a": {
            color: theme.palette.primary.main,
        },
    },
    logText: {
        padding: `0 ${theme.spacing(0.5)}px`,
    },
    wordWrapOn: {
        overflowWrap: 'anywhere',
    },
    wordWrapOff: {
        whiteSpace: 'nowrap',
    },
});

const theme = createTheme({
    overrides: {
        MuiTypography: {
            body2: {
                color: grey["200"]
            }
        }
    },
    typography: {
        fontFamily: 'monospace',
    }
});

interface ProcessLogCodeSnippetProps {
    lines: string[];
    fontSize: number;
    wordWrap?: boolean;
}

interface ProcessLogCodeSnippetAuthProps {
    auth: FederationConfig;
}

const renderLinks = (fontSize: number, auth: FederationConfig, dispatch: Dispatch) => (text: string) => {
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
                    <Link onClick={() => {
                        const url = getNavUrl(links[index], auth)
                        if (url) {
                            window.open(`${window.location.origin}${url}`, '_blank', "noopener");
                        } else {
                            dispatch(navigationNotAvailable(links[index]));
                        }
                    }}
                        style={{ cursor: 'pointer' }}>
                        {links[index]}
                    </Link>}
            </React.Fragment>
        )}
    </Typography>;
};

const mapStateToProps = (state: RootState): ProcessLogCodeSnippetAuthProps => ({
    auth: state.auth,
});

export const ProcessLogCodeSnippet = withStyles(styles)(connect(mapStateToProps)(
    ({ classes, lines, fontSize, auth, dispatch, wordWrap }: ProcessLogCodeSnippetProps & WithStyles<CssRules> & ProcessLogCodeSnippetAuthProps & DispatchProp) => {
        const [followMode, setFollowMode] = useState<boolean>(true);
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
                    if (elem.scrollTop + (elem.clientHeight * 1.1) >= elem.scrollHeight) {
                        setFollowMode(true);
                    } else {
                        setFollowMode(false);
                    }
                }}>
                {lines.map((line: string, index: number) =>
                    <Typography key={index} component="span"
                        className={classNames(classes.logText, wordWrap ? classes.wordWrapOn : classes.wordWrapOff)}>
                        {renderLinks(fontSize, auth, dispatch)(line)}
                    </Typography>
                )}
            </div>
        </MuiThemeProvider>
    }));
