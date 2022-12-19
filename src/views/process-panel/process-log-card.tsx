// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState } from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardHeader,
    IconButton,
    CardContent,
    Tooltip,
    Grid,
    Typography,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import {
    CloseIcon,
    CollectionIcon,
    CopyIcon,
    LogIcon,
    MaximizeIcon,
    UnMaximizeIcon,
    TextDecreaseIcon,
    TextIncreaseIcon,
    WordWrapOffIcon,
    WordWrapOnIcon,
} from 'components/icon/icon';
import { Process } from 'store/processes/process';
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

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    content: {
        padding: theme.spacing.unit * 0,
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
        paddingTop: theme.spacing.unit * 0.5,
        color: theme.customs.colors.greyD
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL
    },
    root: {
        height: '100%',
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
}

type ProcessLogsCardProps = ProcessLogsCardDataProps
    & ProcessLogsCardActionProps
    & CodeSnippetDataProps
    & WithStyles<CssRules>
    & MPVPanelProps;

export const ProcessLogsCard = withStyles(styles)(
    ({ classes, process, filters, selectedFilter, lines,
        onLogFilterChange, navigateToLog, onCopy,
        doHidePanel, doMaximizePanel, doUnMaximizePanel, panelMaximized, panelName }: ProcessLogsCardProps) => {
        const [wordWrap, setWordWrap] = useState<boolean>(true);
        const [fontSize, setFontSize] = useState<number>(3);
        const fontBaseSize = 10;
        const fontStepSize = 1;

        return <Grid item className={classes.root} xs={12}>
            <Card className={classes.card}>
                <CardHeader className={classes.header}
                    avatar={<LogIcon className={classes.iconHeader} />}
                    action={<Grid container direction='row' alignItems='center'>
                        <Grid item>
                            <ProcessLogForm selectedFilter={selectedFilter}
                                filters={filters} onChange={onLogFilterChange} />
                        </Grid>
                        <Grid item>
                            <Tooltip title="Decrease font size" disableFocusListener>
                                <IconButton onClick={() => fontSize > 1 && setFontSize(fontSize-1)}>
                                    <TextDecreaseIcon />
                                </IconButton>
                            </Tooltip>
                        </Grid>
                        <Grid item>
                            <Tooltip title="Increase font size" disableFocusListener>
                                <IconButton onClick={() => fontSize < 5 && setFontSize(fontSize+1)}>
                                    <TextIncreaseIcon />
                                </IconButton>
                            </Tooltip>
                        </Grid>
                        <Grid item>
                            <Tooltip title="Copy to clipboard" disableFocusListener>
                                <IconButton>
                                    <CopyToClipboard text={lines.join()} onCopy={() => onCopy("Log copied to clipboard")}>
                                        <CopyIcon />
                                    </CopyToClipboard>
                                </IconButton>
                            </Tooltip>
                        </Grid>
                        <Grid item>
                            <Tooltip title={`${wordWrap ? 'Disable' : 'Enable'} word wrapping`} disableFocusListener>
                                <IconButton onClick={() => setWordWrap(!wordWrap)}>
                                    {wordWrap ? <WordWrapOffIcon /> : <WordWrapOnIcon />}
                                </IconButton>
                            </Tooltip>
                        </Grid>
                        <Grid item>
                            <Tooltip title="Go to Log collection" disableFocusListener>
                                <IconButton onClick={() => navigateToLog(process.containerRequest.logUuid!)}>
                                    <CollectionIcon />
                                </IconButton>
                            </Tooltip>
                        </Grid>
                        { doUnMaximizePanel && panelMaximized &&
                        <Tooltip title={`Unmaximize ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton onClick={doUnMaximizePanel}><UnMaximizeIcon /></IconButton>
                        </Tooltip> }
                        { doMaximizePanel && !panelMaximized &&
                        <Tooltip title={`Maximize ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton onClick={doMaximizePanel}><MaximizeIcon /></IconButton>
                        </Tooltip> }
                        { doHidePanel &&
                        <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton disabled={panelMaximized} onClick={doHidePanel}><CloseIcon /></IconButton>
                        </Tooltip> }
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
                            spacing={24}
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
            </Card>
        </Grid >
});

