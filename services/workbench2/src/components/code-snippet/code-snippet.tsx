// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, Typography, withStyles, Link } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import classNames from 'classnames';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { FederationConfig, getNavUrl } from 'routes/routes';
import { Dispatch } from 'redux';
import { navigationNotAvailable } from 'store/navigation/navigation-action';

type CssRules = 'root' | 'inlineRoot' | 'space' | 'inline';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        overflow: 'auto',
        padding: theme.spacing.unit,
    },
    inlineRoot: {
        padding: "3px",
        display: "inline",
    },
    space: {
        marginLeft: '15px',
    },
    inline: {
        display: 'inline',
    },
});

export interface CodeSnippetDataProps {
    lines: string[];
    className?: string;
    apiResponse?: boolean;
    linked?: boolean;
    children?: JSX.Element;
    inline?: boolean;
}

interface CodeSnippetAuthProps {
    auth: FederationConfig;
}

type CodeSnippetProps = CodeSnippetDataProps & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): CodeSnippetAuthProps => ({
    auth: state.auth,
});

export const CodeSnippet = withStyles(styles)(connect(mapStateToProps)(
    ({ classes, lines, linked, className, apiResponse, dispatch, auth, children, inline }: CodeSnippetProps & CodeSnippetAuthProps & DispatchProp) =>
        <Typography
            component="div"
            className={classNames([classes.root, className, inline ? classes.inlineRoot : undefined])}>
            <Typography className={apiResponse ? classes.space : classNames([className, inline ? classes.inline : undefined])} component="pre">
                {children}
                {linked ?
                    lines.map((line, index) => <React.Fragment key={index}>{renderLinks(auth, dispatch)(line)}{`\n`}</React.Fragment>) :
                    lines.join('\n')
                }
            </Typography>
        </Typography>
));

export const renderLinks = (auth: FederationConfig, dispatch: Dispatch) => (text: string): JSX.Element => {
    // Matches UUIDs & PDHs
    const REGEX = /[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}|[0-9a-f]{32}\+\d+/g;
    const links = text.match(REGEX);
    if (!links) {
        return <>{text}</>;
    }
    return <>
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
    </>;
};
