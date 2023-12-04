// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Typography, Grid } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { InstanceTypeIcon } from 'components/icon/icon';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { ClusterConfigJSON } from 'common/config';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { formatCost, formatFileSize } from 'common/formatters';

type CssRules = 'root' | 'instanceType';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
       width: '100%',
       overflow: 'auto'
    },
    instanceType: {
        padding: "10px",
    },
});

type InstanceTypesPanelConnectedProps = {config: ClusterConfigJSON};

type InstanceTypesPanelRootProps = InstanceTypesPanelConnectedProps & WithStyles<CssRules>;

const mapStateToProps = ({auth}: RootState): InstanceTypesPanelConnectedProps => ({
    config: auth.config.clusterConfig,
});

export const InstanceTypesPanel = withStyles(styles)(connect(mapStateToProps)(
    ({ config, classes }: InstanceTypesPanelRootProps) => {

        const instances = config.InstanceTypes || {};

        return <Card className={classes.root}>
            <CardContent>
                <Grid container direction="row">
                    {Object.keys(instances).length > 0 ?
                        Object.keys(instances).map((instanceKey) => {
                            const instanceType = instances[instanceKey];

                            return <Grid data-cy={instanceKey} className={classes.instanceType} item sm={6} xs={12} key={instanceKey}>
                                <Card>
                                    <CardContent>
                                        <Typography variant="h6">
                                            {instanceKey}
                                        </Typography>
                                        <Typography>
                                            Provider type: {instanceType.ProviderType}
                                        </Typography>
                                        <Typography>
                                            Price: {formatCost(instanceType.Price)}
                                        </Typography>
                                        <Typography>
                                            Cores: {instanceType.VCPUs}
                                        </Typography>
                                        <Typography>
                                            Preemptible: {instanceType.Preemptible.toString()}
                                        </Typography>
                                        <Typography>
                                            Max disk request: {formatFileSize(instanceType.IncludedScratch)}
                                        </Typography>
                                        <Typography>
                                            Max ram request: {formatFileSize(instanceType.RAM - config.Containers.ReserveExtraRAM)}
                                        </Typography>
                                        {instanceType.CUDA && instanceType.CUDA.DeviceCount > 0 ?
                                            <>
                                                <Typography>
                                                    CUDA GPUs: {instanceType.CUDA.DeviceCount}
                                                </Typography>
                                                <Typography>
                                                    Hardware capability: {instanceType.CUDA.HardwareCapability}
                                                </Typography>
                                                <Typography>
                                                    Driver version: {instanceType.CUDA.DriverVersion}
                                                </Typography>
                                            </> : <></>
                                        }
                                    </CardContent>
                                </Card>
                            </Grid>
                        }) :
                        <NotFoundView
                            icon={InstanceTypeIcon}
                            messages={["No instances found"]}
                        />
                    }
                </Grid>
            </CardContent>
        </Card>
    }
));
