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
import { DefaultCodeSnippet } from 'components/default-code-snippet/default-code-snippet';
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
        paddingBottom: theme.spacing.unit,
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing.unit * 0.5
    },
    content: {
        padding: theme.spacing.unit * 1.0,
        paddingTop: theme.spacing.unit * 0.5,
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 1,
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5
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
    const command = process.containerRequest.command.map((v) =>
      shellescape([v]) // Escape each arg separately
    );

    let formattedCommand = [...command];
    formattedCommand.forEach((item, i, arr) => {
      // Indent lines after the first
      const indent = i > 0 ? '  ' : '';
      // Escape newlines on every non-last arg when there are multiple lines
      const lineBreak = arr.length > 1 && i < arr.length - 1 ? ' \\' : '';
      arr[i] = `${indent}${item}${lineBreak}`;
    });

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
                <Tooltip title="Copy to clipboard" disableFocusListener>
                  <IconButton>
                    <CopyToClipboard
                      text={command.join(" ")}
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
          <DefaultCodeSnippet lines={formattedCommand} linked />
        </CardContent>
      </Card>
    );
  }
);
