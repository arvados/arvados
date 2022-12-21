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
    MemoryIcon,
    UnMaximizeIcon,
    ProcessIcon
} from 'components/icon/icon';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { connect } from 'react-redux';
import { Process } from 'store/processes/process';
import { NodeInstanceType } from 'store/process-panel/process-panel';
import { DefaultView } from 'components/default-view/default-view';
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { formatFileSize } from "common/formatters";
import { InputCollectionMount } from 'store/processes/processes-actions';
import { MountKind, TemporaryDirectoryMount } from 'models/mount-types';

interface ProcessResourceCardDataProps {
    process: Process;
    nodeInfo: NodeInstanceType | null;
}

type CssRules = "card" | "header" | "title" | "avatar" | "iconHeader" | "content" | "sectionH3";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingBottom: "0px"
    },
    title: {
        paddingTop: theme.spacing.unit * 0.5
    },
    avatar: {
        paddingTop: theme.spacing.unit * 0.5
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL,
    },
    content: {
        paddingTop: "0px",
        maxHeight: `calc(100% - ${theme.spacing.unit * 7.5}px)`,
        overflow: "auto"
    },
    sectionH3: {
        margin: "0.5em",
        color: theme.customs.colors.greyD,
        fontSize: "0.8125rem",
        textTransform: "uppercase",
    }
});

type ProcessResourceCardProps = ProcessResourceCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessResourceCard = withStyles(styles)(connect()(
    ({ classes, nodeInfo, doHidePanel, doMaximizePanel, doUnMaximizePanel, panelMaximized, panelName, process, }: ProcessResourceCardProps) => {

        const loading = false;

        let diskRequest = 0;
        if (process.container?.mounts) {
            for (const mnt in process.container.mounts) {
                const mp = process.container.mounts[mnt];
                if (mp.kind === MountKind.TEMPORARY_DIRECTORY) {
                    diskRequest += mp.capacity;
                }
            }
        }

        return <Card className={classes.card} data-cy="process-resources-card">
            <CardHeader
                className={classes.header}
                classes={{
                    content: classes.title,
                    avatar: classes.avatar,
                }}
                avatar={<MemoryIcon className={classes.iconHeader} />}
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
                <Grid container>
                    <Grid item xs={4}>
                        <h3 className={classes.sectionH3}>Requested Resources</h3>
                        <Grid container>
                            <Grid item xs={12}>
                                <DetailsAttribute label="Cores" value={process.container?.runtimeConstraints.vcpus} />
                            </Grid>
                            <Grid item xs={12}>
                                <DetailsAttribute label="RAM" value={formatFileSize(process.container?.runtimeConstraints.ram)} />
                            </Grid>
                            <Grid item xs={12}>
                                <DetailsAttribute label="Disk" value={formatFileSize(diskRequest)} />
                            </Grid>

                            {process.container?.runtimeConstraints.cuda &&
                                process.container?.runtimeConstraints.cuda.device_count > 0 ?
                                <>
                                    <Grid item xs={12}>
                                        <DetailsAttribute label="CUDA devices" value={process.container?.runtimeConstraints.cuda.device_count} />
                                    </Grid>
                                    <Grid item xs={12}>
                                        <DetailsAttribute label="CUDA driver version" value={process.container?.runtimeConstraints.cuda.driver_version} />
                                    </Grid>
                                    <Grid item xs={12}>
                                        <DetailsAttribute label="CUDA hardware capability" value={process.container?.runtimeConstraints.cuda.hardware_capability} />
                                    </Grid>
                                </> : null}

                            {process.container?.runtimeConstraints.keep_cache_ram &&
                                process.container?.runtimeConstraints.keep_cache_ram > 0 ?
                                <Grid item xs={12}>
                                    <DetailsAttribute label="Keep cache (RAM)" value={formatFileSize(process.container?.runtimeConstraints.keep_cache_ram)} />
                                </Grid> : null}

                            {process.container?.runtimeConstraints.keep_cache_disk &&
                                process.container?.runtimeConstraints.keep_cache_disk > 0 ?
                                <Grid item xs={12}>
                                    <DetailsAttribute label="Keep cache (disk)" value={formatFileSize(process.container?.runtimeConstraints.keep_cache_disk)} />
                                </Grid> : null}

                            {process.container?.runtimeConstraints.API ? <Grid item xs={12}>
                                <DetailsAttribute label="API access" value={process.container?.runtimeConstraints.API.toString()} />
                            </Grid> : null}

                        </Grid>
                    </Grid>


                    <Grid item xs={8}>
                        <h3 className={classes.sectionH3}>Assigned Instance Type</h3>
                        {nodeInfo === null ? <Grid item xs={8}>
                            No instance type recorded
                        </Grid>
                            :
                            <Grid container>
                                <Grid item xs={6}>
                                    <DetailsAttribute label="Cores" value={nodeInfo.VCPUs} />
                                </Grid>

                                <Grid item xs={6}>
                                    <DetailsAttribute label="Provider type" value={nodeInfo.ProviderType} />
                                </Grid>

                                <Grid item xs={6}>
                                    <DetailsAttribute label="RAM" value={formatFileSize(nodeInfo.RAM)} />
                                </Grid>

                                <Grid item xs={6}>
                                    <DetailsAttribute label="Price" value={"$" + nodeInfo.Price.toString()} />
                                </Grid>

                                <Grid item xs={6}>
                                    <DetailsAttribute label="Disk" value={formatFileSize(nodeInfo.IncludedScratch + nodeInfo.AddedScratch)} />
                                </Grid>

                                <Grid item xs={6}>
                                    <DetailsAttribute label="Preemptible" value={nodeInfo.Preemptible.toString()} />
                                </Grid>

                                {nodeInfo.CUDA && nodeInfo.CUDA.DeviceCount > 0 &&
                                    <>
                                        <Grid item xs={6}>
                                            <DetailsAttribute label="CUDA devices" value={nodeInfo.CUDA.DeviceCount} />
                                        </Grid>

                                        <Grid item xs={6}>
                                        </Grid>

                                        <Grid item xs={6}>
                                            <DetailsAttribute label="CUDA driver version" value={nodeInfo.CUDA.DriverVersion} />
                                        </Grid>

                                        <Grid item xs={6}>
                                        </Grid>

                                        <Grid item xs={6}>
                                            <DetailsAttribute label="CUDA hardware capability" value={nodeInfo.CUDA.HardwareCapability} />
                                        </Grid>
                                    </>
                                }
                            </Grid>}
                    </Grid>
                </Grid>
            </CardContent>
        </Card >;
    }
));
