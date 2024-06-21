// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Tooltip,
    Typography,
    Card,
    CardHeader,
    CardContent,
    IconButton
} from '@material-ui/core';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { WorkflowIcon, MoreVerticalIcon } from 'components/icon/icon';
import { WorkflowResource } from 'models/workflow';
import { ProcessOutputCollectionFiles } from 'views/process-panel/process-output-collection-files';
import { WorkflowDetailsAttributes, RegisteredWorkflowPanelDataProps, getRegisteredWorkflowPanelData } from 'views-components/details-panel/workflow-details';
import { getResource } from 'store/resources/resources';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { ProcessIOCard, ProcessIOCardType } from 'views/process-panel/process-io-card';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { WorkflowProcessesPanel } from './workflow-processes-panel';

type CssRules =
    | 'button'
    | 'infoCard'
    | 'propertiesCard'
    | 'filesCard'
    | 'iconHeader'
    | 'tag'
    | 'label'
    | 'value'
    | 'link'
    | 'centeredLabel'
    | 'warningLabel'
    | 'collectionName'
    | 'readOnlyIcon'
    | 'header'
    | 'title'
    | 'avatar'
    | 'content';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    button: {
        cursor: 'pointer'
    },
    infoCard: {
    },
    propertiesCard: {
        padding: 0,
    },
    filesCard: {
        padding: 0,
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL
    },
    tag: {
        marginRight: theme.spacing.unit / 2,
        marginBottom: theme.spacing.unit / 2
    },
    label: {
        fontSize: '0.875rem',
    },
    centeredLabel: {
        fontSize: '0.875rem',
        textAlign: 'center'
    },
    warningLabel: {
        fontStyle: 'italic'
    },
    collectionName: {
        flexDirection: 'column',
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    },
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    readOnlyIcon: {
        marginLeft: theme.spacing.unit,
        fontSize: 'small',
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5,
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
                const panelsData: MPVPanelState[] = [
                    { name: "Details" },
                    { name: "Runs" },
                    { name: "Outputs" },
                    { name: "Inputs" },
                    { name: "Definition" },
                ];
                return item
                    ? <MPVContainer spacing={8} direction="column" justify-content="flex-start" wrap="nowrap" panelStates={panelsData}>
                        <MPVPanelContent xs="auto" data-cy='registered-workflow-info-panel'>
                            <Card className={classes.infoCard}>
                                <CardHeader
                                    className={classes.header}
                                    classes={{
                                        content: classes.title,
                                        avatar: classes.avatar,
                                    }}
                                    avatar={<WorkflowIcon className={classes.iconHeader} />}
                                    title={
                                        <Tooltip title={item.name} placement="bottom-start">
                                            <Typography noWrap variant='h6'>
                                                {item.name}
                                            </Typography>
                                        </Tooltip>
                                    }
                                    subheader={
                                        <Tooltip title={item.description || '(no-description)'} placement="bottom-start">
                                            <Typography noWrap variant='body1' color='inherit'>
                                                {item.description || '(no-description)'}
                                            </Typography>
                                        </Tooltip>}
                                    action={
                                        <Tooltip title="More options" disableFocusListener>
                                            <IconButton
                                                aria-label="More options"
                                                onClick={event => this.handleContextMenu(event)}>
                                                <MoreVerticalIcon />
                                            </IconButton>
                                        </Tooltip>}

                                />

                                <CardContent className={classes.content}>
                                    <WorkflowDetailsAttributes workflow={item} />
                                </CardContent>
                            </Card>
                        </MPVPanelContent>
                        <MPVPanelContent forwardProps xs maxHeight="100%">
                            <WorkflowProcessesPanel />
                        </MPVPanelContent>
                        <MPVPanelContent forwardProps xs data-cy="process-outputs" maxHeight="100%">
                            <ProcessIOCard
                                label={ProcessIOCardType.OUTPUT}
                                params={outputParams}
                                raw={{}}
                                forceShowParams={true}
                            />
                        </MPVPanelContent>
                        <MPVPanelContent forwardProps xs data-cy="process-inputs" maxHeight="100%">
                            <ProcessIOCard
                                label={ProcessIOCardType.INPUT}
                                params={inputParams}
                                raw={{}}
                                forceShowParams={true}
                            />
                        </MPVPanelContent>
                        <MPVPanelContent xs maxHeight="100%">
                            <Card className={classes.filesCard}>
                                <CardHeader title="Workflow Definition" />
                                <ProcessOutputCollectionFiles isWritable={false} currentItemUuid={workflowCollection} />
                            </Card>
                        </MPVPanelContent>
                    </MPVContainer>
                    :
                    <NotFoundView
                        icon={WorkflowIcon}
                        messages={["Workflow not found"]}
                    />
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const { uuid, ownerUuid, name, description,
                    kind } = this.props.item;
                const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(uuid));
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
                this.props.dispatch<any>(openContextMenu(event, resource));
            }
        }
    )
);
