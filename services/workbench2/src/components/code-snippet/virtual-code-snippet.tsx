// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, Typography, withStyles, Tooltip, IconButton } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import classNames from 'classnames';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { FederationConfig } from 'routes/routes';
import { renderLinks } from './code-snippet';
import { FixedSizeList } from 'react-window';
import AutoSizer from "react-virtualized-auto-sizer";
import CopyResultToClipboard from 'components/copy-to-clipboard/copy-result-to-clipboard';
import { CopyIcon } from 'components/icon/icon';
import { SnackbarKind, snackbarActions } from 'store/snackbar/snackbar-actions';
import { Dispatch } from "redux";

type CssRules = 'root' | 'space' | 'content' | 'copyButton' ;

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        position: 'relative',
        boxSizing: 'border-box',
        height: '100%',
        padding: theme.spacing.unit,
    },
    space: {
        marginLeft: '15px',
    },
    content: {
        maxHeight: '100%',
        height: '100vh',
    },
    copyButton: {
        position: 'absolute',
        top: '8px',
        right: '16px',
        zIndex: 100,
    },
});

export interface CodeSnippetDataProps {
    lines: string[];
    lineFormatter?: (lines: string[], index: number) => string;
    className?: string;
    apiResponse?: boolean;
    linked?: boolean;
    copyButton?: boolean;
}

export interface CodeSnippetActionProps {
    renderLinks: (auth: FederationConfig) => (text: string) => JSX.Element;
    onCopyToClipboard: () => void;
}

interface CodeSnippetAuthProps {
    auth: FederationConfig;
}

type CodeSnippetProps = CodeSnippetDataProps & CodeSnippetActionProps & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): CodeSnippetAuthProps => ({
    auth: state.auth,
});

const mapDispatchToProps = (dispatch: Dispatch): CodeSnippetActionProps => ({
    renderLinks: (auth: FederationConfig) => renderLinks(auth, dispatch),
    onCopyToClipboard: () => {
        dispatch<any>(
            snackbarActions.OPEN_SNACKBAR({
                message: "Contents copied to clipboard",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
    },
});

export const VirtualCodeSnippet = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(
    ({ classes, lines, lineFormatter, linked, copyButton, renderLinks, onCopyToClipboard, className, apiResponse, auth }: CodeSnippetProps & CodeSnippetAuthProps) => {
        const RenderRow = ({index, style}) => {
            const lineContents = lineFormatter ? lineFormatter(lines, index) : lines[index];
            return <span style={style}>{linked ? renderLinks(auth)(lineContents) : lineContents}</span>
        };

        const formatClipboardText = (lines: string[]) => () =>  {
            return lines.join('\n');
        };



        return <Typography
            component="div"
            className={classNames([classes.root, className])}>
            {copyButton && <span className={classes.copyButton}>
                <Tooltip title="Copy text to clipboard" disableFocusListener>
                    <IconButton>
                        <CopyResultToClipboard
                            getText={formatClipboardText(lines)}
                            onCopy={onCopyToClipboard}
                        >
                            <CopyIcon />
                        </CopyResultToClipboard>
                    </IconButton>
                </Tooltip>
            </span>}
            <Typography className={classNames(classes.content, apiResponse ? classes.space : className)} component="pre">
                <AutoSizer>
                    {({ height, width }) =>
                        <FixedSizeList
                            height={height}
                            width={width}
                            itemSize={21}
                            itemCount={lines.length}
                        >
                            {RenderRow}
                        </FixedSizeList>
                    }
                </AutoSizer>
            </Typography>
        </Typography>;
}));
