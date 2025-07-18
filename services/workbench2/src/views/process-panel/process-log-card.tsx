// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState } from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { CardHeader, IconButton, CardContent, Tooltip, Grid, Typography } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { useAsyncInterval } from 'common/use-async-interval';
import { ArvadosTheme } from 'common/custom-theme';
import {
    CollectionIcon,
    CopyIcon,
    LogIcon,
    TextDecreaseIcon,
    TextIncreaseIcon,
    WordWrapOffIcon,
    WordWrapOnIcon,
} from 'components/icon/icon';
import { Process, isProcessRunning, isProcessQueued } from 'store/processes/process';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import {
    FilterOption,
    ProcessLogForm
} from 'views/process-panel/process-log-form';
import { ProcessLogCodeSnippet } from 'views/process-panel/process-log-code-snippet';
import { DefaultView } from 'components/default-view/default-view';
import { CodeSnippetDataProps } from 'components/code-snippet/code-snippet';
import CopyToClipboard from 'react-copy-to-clipboard';

type CssRules = 'card' | 'content' | 'title' | 'iconHeader' | 'header' | 'root' | 'logViewer' | 'logViewerContainer';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing(1),
        paddingBottom: theme.spacing(1),
    },
    content: {
        padding: theme.spacing(0),
        height: '100%',
    },
    logViewer: {
        height: '100%',
        overflowY: 'scroll', // Required for MacOS's Safari -- See #19687
    },
    logViewerContainer: {
        height: '100%',
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing(0.5),
        color: theme.customs.colors.greyD
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL
    },
    root: {
        height: '100%',
        overflow: 'hidden',
    },
});

export interface ProcessLogsCardDataProps {
    process: Process;
    selectedFilter: FilterOption;
    filters: FilterOption[];
}

export interface ProcessLogsCardActionProps {
    onLogFilterChange: (filter: FilterOption) => void;
    navigateToLog: (uuid: string) => void;
    onCopy: (text: string) => void;
    pollProcessLogs: (processUuid: string) => Promise<void>;
}

type ProcessLogsCardProps = ProcessLogsCardDataProps
    & ProcessLogsCardActionProps
    & CodeSnippetDataProps
    & WithStyles<CssRules>
    & MPVPanelProps;

export const ProcessLogsCard = withStyles(styles)(
    ({ classes, process, filters, selectedFilter, lines,
       onLogFilterChange, navigateToLog, onCopy, pollProcessLogs, panelName }: ProcessLogsCardProps) => {
           const [wordWrap, setWordWrap] = useState<boolean>(true);
           const [fontSize, setFontSize] = useState<number>(3);
           const fontBaseSize = 10;
           const fontStepSize = 1;

           useAsyncInterval(() => (
               pollProcessLogs(process.containerRequest.uuid)
           ), isProcessQueued(process) ? 20000 : (isProcessRunning(process) ? 2000 : null));

           return (
               <Grid item className={classes.root} xs={12}>
                   <section className={classes.card}>
                       <CardHeader className={classes.header}
                                   avatar={<LogIcon className={classes.iconHeader} />}
                                   action={<Grid container direction='row' alignItems='center'>
                                       <Grid item>
                                           <ProcessLogForm selectedFilter={selectedFilter}
                                                           filters={filters} onChange={onLogFilterChange} />
                                       </Grid>
                                       <Grid item>
                                           <Tooltip title="Decrease font size" disableFocusListener>
                                               <IconButton onClick={() => fontSize > 1 && setFontSize(fontSize-1)} size="large">
                                                   <TextDecreaseIcon />
                                               </IconButton>
                                           </Tooltip>
                                       </Grid>
                                       <Grid item>
                                           <Tooltip title="Increase font size" disableFocusListener>
                                               <IconButton onClick={() => fontSize < 5 && setFontSize(fontSize+1)} size="large">
                                                   <TextIncreaseIcon />
                                               </IconButton>
                                           </Tooltip>
                                       </Grid>
                                       <Grid item>
                                           <Tooltip title="Copy link to clipboard" disableFocusListener>
                                               <IconButton size="large">
                                                   <CopyToClipboard text={lines.join()} onCopy={() => onCopy("Log copied to clipboard")}>
                                                       <CopyIcon />
                                                   </CopyToClipboard>
                                               </IconButton>
                                           </Tooltip>
                                       </Grid>
                                       <Grid item>
                                           <Tooltip title={`${wordWrap ? 'Disable' : 'Enable'} word wrapping`} disableFocusListener>
                                               <IconButton onClick={() => setWordWrap(!wordWrap)} size="large">
                                                                   {wordWrap ? <WordWrapOffIcon /> : <WordWrapOnIcon />}
                                               </IconButton>
                                           </Tooltip>
                                       </Grid>
                                       <Grid item>
                                           <Tooltip title="Go to Log collection" disableFocusListener>
                                               <IconButton
                                                   onClick={() => navigateToLog(process.containerRequest.logUuid!)}
                                                   size="large">
                                                   <CollectionIcon />
                                               </IconButton>
                                           </Tooltip>
                                       </Grid>
                                   </Grid>}
                                   title={
                                       <Typography noWrap variant='h6' className={classes.title}>
                                                                   Logs
                                       </Typography>}
                       />
                       <CardContent className={classes.content}>
                           {lines.length > 0
                                         ? < Grid
                           className={classes.logViewerContainer}
                           container
                           spacing={3}
                           direction='column'>
                           <Grid className={classes.logViewer} item xs>
                               <ProcessLogCodeSnippet fontSize={fontBaseSize+(fontStepSize*fontSize)} wordWrap={wordWrap} lines={lines} />
                           </Grid>
                           </Grid>
                            : <DefaultView
                                  icon={LogIcon}
                                  messages={['No logs yet']} />
                           }
                       </CardContent>
                   </section>
               </Grid >
           );
});
