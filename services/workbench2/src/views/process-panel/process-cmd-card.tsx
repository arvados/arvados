// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardHeader,
    IconButton,
    CardContent,
    Tooltip,
    Typography,
    Grid,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon, CommandIcon, CopyIcon } from 'components/icon/icon';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { DefaultVirtualCodeSnippet } from 'components/default-code-snippet/default-virtual-code-snippet';
import { Process } from 'store/processes/process';
import shellescape from 'shell-escape';
import CopyToClipboard from 'react-copy-to-clipboard';

type CssRules = 'card' | 'content' | 'title' | 'header' | 'avatar' | 'iconHeader';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: 0,
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing.unit * 0.5
    },
    content: {
        height: `calc(100% - ${theme.spacing.unit * 6}px)`,
        padding: theme.spacing.unit * 1.0,
        paddingTop: 0,
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 1,
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5,
        color: theme.customs.colors.greyD,
        fontSize: '1.875rem'
    },
});

interface ProcessCmdCardDataProps {
  process: Process;
  onCopy: (text: string) => void;
}

type ProcessCmdCardProps = ProcessCmdCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessCmdCard = withStyles(styles)(
  ({
    process,
    onCopy,
    classes,
    doHidePanel,
  }: ProcessCmdCardProps) => {

    const formatLine = (lines: string[], index: number): string => {
      // Escape each arg separately
      let line = shellescape([lines[index]])
      // Indent lines after the first
      const indent = index > 0 ? '  ' : '';
      // Add backslash "escaped linebreak"
      const lineBreak = lines.length > 1 && index < lines.length - 1 ? ' \\' : '';

      return `${indent}${line}${lineBreak}`;
    };

    return (
      <Card className={classes.card}>
        <CardHeader
          className={classes.header}
          classes={{
            content: classes.title,
            avatar: classes.avatar,
          }}
          avatar={<CommandIcon className={classes.iconHeader} />}
          title={
            <Typography noWrap variant="h6" color="inherit">
              Command
            </Typography>
          }
          action={
            <Grid container direction="row" alignItems="center">
              <Grid item>
                <Tooltip title="Copy link to clipboard" disableFocusListener>
                  <IconButton>
                    <CopyToClipboard
                      text={" "}
                      onCopy={() => onCopy("Command copied to clipboard")}
                    >
                      <CopyIcon />
                    </CopyToClipboard>
                  </IconButton>
                </Tooltip>
              </Grid>
              <Grid item>
                {doHidePanel && (
                  <Tooltip
                    title={`Close Command Panel`}
                    disableFocusListener
                  >
                    <IconButton onClick={doHidePanel}>
                      <CloseIcon />
                    </IconButton>
                  </Tooltip>
                )}
              </Grid>
            </Grid>
          }
        />
        <CardContent className={classes.content}>
          <DefaultVirtualCodeSnippet
            lines={process.containerRequest.command}
            lineFormatter={formatLine}
            linked
          />
        </CardContent>
      </Card>
    );
  }
);
