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
    CircularProgress,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import {
    CloseIcon,
    MaximizeIcon,
    UnMaximizeIcon,
    ProcessIcon
} from 'components/icon/icon';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { connect } from 'react-redux';
import { Process } from 'store/processes/process';
import { NodeInstanceType } from 'store/process-panel/process-panel';
import { DefaultView } from 'components/default-view/default-view';

interface ProcessResourceCardDataProps {
    process: Process;
    nodeInfo: NodeInstanceType | null;
}

type CssRules = "card" | "header" | "title" | "avatar" | "iconHeader" | "content";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {},
    header: {},
    title: {},
    avatar: {},
    iconHeader: {},
    content: {}
});

type ProcessResourceCardProps = ProcessResourceCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessResourceCard = withStyles(styles)(connect()(
    ({ classes, nodeInfo, doHidePanel, doMaximizePanel, doUnMaximizePanel, panelMaximized, panelName, process, }: ProcessResourceCardProps) => {

        const loading = nodeInfo === null;

        return <Card className={classes.card} data-cy="process-resources-card">
            <CardHeader
                className={classes.header}
                classes={{
                    content: classes.title,
                    avatar: classes.avatar,
                }}
                avatar={<ProcessIcon className={classes.iconHeader} />}
                title={
                    <Typography noWrap variant='h6' color='inherit'>
                        Resources
                    </Typography>
                }
                action={
                    <div>
                        {doUnMaximizePanel && panelMaximized &&
                            <Tooltip title={`Unmaximize ${panelName || 'panel'}`} disableFocusListener>
                                <IconButton onClick={doUnMaximizePanel}><UnMaximizeIcon /></IconButton>
                            </Tooltip>}
                        {doMaximizePanel && !panelMaximized &&
                            <Tooltip title={`Maximize ${panelName || 'panel'}`} disableFocusListener>
                                <IconButton onClick={doMaximizePanel}><MaximizeIcon /></IconButton>
                            </Tooltip>}
                        {doHidePanel &&
                            <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                                <IconButton disabled={panelMaximized} onClick={doHidePanel}><CloseIcon /></IconButton>
                            </Tooltip>}
                    </div>
                } />
            <CardContent className={classes.content}>
                <>
                    {/* raw is undefined until params are loaded */}
                    {loading && <Grid container item alignItems='center' justify='center'>
                        <CircularProgress />
                    </Grid>}
                    {/* Once loaded, either raw or params may still be empty
                      *   Raw when all params are empty
                      *   Params when raw is provided by containerRequest properties but workflow mount is absent for preview
                      */}
                    {!loading &&
                        <>
                            <div>
                                stuff
                            </div>
                        </>}
                    {!loading && <Grid container item alignItems='center' justify='center'>
                        <DefaultView messages={["No parameters found"]} />
                    </Grid>}
                </>
            </CardContent>
        </Card>;
    }
));
