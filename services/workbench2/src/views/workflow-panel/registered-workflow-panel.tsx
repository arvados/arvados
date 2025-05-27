// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { CardHeader, CardContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { WorkflowIcon } from 'components/icon/icon';
import { WorkflowResource } from 'models/workflow';
import { ProcessOutputCollectionFiles } from 'views/process-panel/process-output-collection-files';
import { WorkflowDetailsAttributes, RegisteredWorkflowPanelDataProps, getRegisteredWorkflowPanelData } from 'views-components/details-panel/workflow-details';
import { getResource } from 'store/resources/resources';
import { openContextMenuAndSelect } from 'store/context-menu/context-menu-actions';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { ProcessIOCard, ProcessIOCardType } from 'views/process-panel/process-io-card';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { WorkflowProcessesPanel } from './workflow-processes-panel';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';
import { DetailsCardRoot } from 'views-components/details-card/details-card-root';

type CssRules =
    'root'
    | 'mpvRoot'
    | 'button'
    | 'infoCard'
    | 'filesCard'
    | 'label'
    | 'content'
    | 'subHeader';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'none',
    },
    mpvRoot: {
        width: '100%',
        height: '100%',
    },
    button: {
        cursor: 'pointer'
    },
    infoCard: {
    },
    filesCard: {
        padding: 0,
    },
    subHeader: {
        color: theme.customs.colors.greyD
    },
    label: {
        fontSize: '0.875rem',
    },
    content: {
        padding: theme.spacing(1),
        paddingTop: theme.spacing(0.5),
        '&:last-child': {
            paddingBottom: theme.spacing(1),
        }
    }
});

type RegisteredWorkflowPanelProps = RegisteredWorkflowPanelDataProps & DispatchProp & WithStyles<CssRules>

export const RegisteredWorkflowPanel = withStyles(styles)(connect(
    (state: RootState, props: RouteComponentProps<{ id: string }>) => {
        const item = getResource<WorkflowResource>(props.match.params.id)(state.resources);
        if (item) {
            return getRegisteredWorkflowPanelData(item, state.auth);
        }
        return { item, inputParams: [], outputParams: [], workflowCollection: "", gitprops: {} };
    })(
        class extends React.Component<RegisteredWorkflowPanelProps> {
            render() {
                const { classes, item, inputParams, outputParams, workflowCollection } = this.props;
                const panelsData: MPVPanelState[] = [{ name: 'Overview' }, { name: 'Runs' }, { name: 'Outputs' }, { name: 'Inputs' }, { name: 'Definition' }];
                return item ? (
                    <section className={classes.root}>
                        <DetailsCardRoot />
                        <MPVContainer
                            className={classes.mpvRoot}
                            spacing={1}
                            justifyContent='flex-start'
                            panelStates={panelsData}>
                            <MPVPanelContent
                                xs='auto'
                                data-cy='registered-workflow-info-panel'>
                                <section className={classes.infoCard}>
                                    <CardContent className={classes.content}>
                                        <WorkflowDetailsAttributes workflow={item} />
                                    </CardContent>
                                </section>
                            </MPVPanelContent>
                            <MPVPanelContent
                                forwardProps
                                xs
                                maxHeight='100%'>
                                <WorkflowProcessesPanel />
                            </MPVPanelContent>
                            <MPVPanelContent
                                forwardProps
                                xs
                                data-cy='process-outputs'
                                maxHeight='100%'>
                                <ProcessIOCard
                                    label={ProcessIOCardType.OUTPUT}
                                    params={outputParams}
                                    raw={{}}
                                    forceShowParams={true}/>
                            </MPVPanelContent>
                            <MPVPanelContent
                                forwardProps
                                xs
                                data-cy='process-inputs'
                                maxHeight='100%'>
                                <ProcessIOCard
                                    label={ProcessIOCardType.INPUT}
                                    params={inputParams}
                                    raw={{}}
                                    forceShowParams={true}/>
                            </MPVPanelContent>
                            <MPVPanelContent
                                xs
                                maxHeight='100%'>
                                <section className={classes.filesCard}>
                                    <CardHeader title='Workflow Definition' />
                                    <ProcessOutputCollectionFiles
                                        isWritable={false}
                                        currentItemUuid={workflowCollection}
                                    />
                                </section>
                            </MPVPanelContent>
                        </MPVContainer>
                    </section>
                ) : (
                    <NotFoundView
                        icon={WorkflowIcon}
                        messages={['Workflow not found']}
                    />
                );
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const { uuid, ownerUuid, name, description,
                    kind } = this.props.item;
                const menuKind = this.props.dispatch<any>(resourceToMenuKind(uuid));
                const resource = {
                    uuid,
                    ownerUuid,
                    name,
                    description,
                    kind,
                    menuKind,
                };
                // Avoid expanding/collapsing the panel
                event.stopPropagation();
                this.props.dispatch<any>(openContextMenuAndSelect(event, resource));
            }
        }
    )
);
