// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import withStyles from '@material-ui/core/styles/withStyles';
import { DispatchProp, connect } from 'react-redux';
import { RouteComponentProps } from 'react-router';
import { StyleRulesCallback, WithStyles } from '@material-ui/core';
import { RootState } from 'store/store';
import { Resource } from 'models/resource';
import { ResourcesState, getResource } from 'store/resources/resources';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { getProperty } from 'store/properties/properties';
import { PROJECT_PANEL_CURRENT_UUID } from 'store/project-panel/project-panel-action';
import { ArvadosTheme } from 'common/custom-theme';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { GroupClass, GroupResource } from 'models/group';
import { CollectionResource } from 'models/collection';
import { resourceIsFrozen } from 'common/frozen-resources';
import { deselectAllOthers, toggleOne } from 'store/multiselect/multiselect-actions';
import { DetailsCardRoot } from 'views-components/details-card/details-card-root';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { ProjectPanelData } from './project-panel-data';
import { ProjectPanelRun } from './project-panel-run';

type CssRules = 'root' | 'button' | 'mpvRoot' | 'dataExplorer';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
    },
    button: {
        marginLeft: theme.spacing.unit,
    },
    mpvRoot: {
        flexGrow: 1,
        display: 'flex',
        flexDirection: 'column',
        '& > div': {
            height: '100%',
        },
    },
    dataExplorer: {
        height: "100%",
    },
});

const panelsData: MPVPanelState[] = [
    { name: "Data", visible: true },
    { name: "Workflow Runs", visible: false },
];

interface ProjectPanelDataProps {
    currentItemId: string;
    resources: ResourcesState;
    project: GroupResource;
    isAdmin: boolean;
    userUuid: string;
    dataExplorerItems: any;
    working: boolean;
}

type ProjectPanelProps = ProjectPanelDataProps & DispatchProp & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

const mapStateToProps = (state: RootState) => {
    const currentItemId = getProperty<string>(PROJECT_PANEL_CURRENT_UUID)(state.properties);
    const project = getResource<GroupResource>(currentItemId || "")(state.resources);
    return {
        currentItemId,
        project,
        resources: state.resources,
        userUuid: state.auth.user!.uuid,
    };
}

export const ProjectPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<ProjectPanelProps> {

            render() {
                const { classes } = this.props;
                return <div data-cy='project-panel' className={classes.root}>
                    <DetailsCardRoot />
                    <MPVContainer
                        className={classes.mpvRoot}
                        spacing={8}
                        panelStates={panelsData}
                        mutuallyExclusive
                        justify-content="flex-start"
                        direction="column"
                        wrap="nowrap">
                        <MPVPanelContent
                            forwardProps
                            xs="auto"
                            data-cy="process-data"
                            className={classes.dataExplorer}>
                            <ProjectPanelData />
                        </MPVPanelContent>
                        <MPVPanelContent
                            forwardProps
                            xs="auto"
                            data-cy="process-run"
                            className={classes.dataExplorer}>
                            <ProjectPanelRun />
                        </MPVPanelContent>
                    </MPVContainer>
                </div>
            }

            isCurrentItemChild = (resource: Resource) => {
                return resource.ownerUuid === this.props.currentItemId;
            };

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const { resources, isAdmin } = this.props;
                const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
                // When viewing the contents of a filter group, all contents should be treated as read only.
                let readonly = false;
                const project = getResource<GroupResource>(this.props.currentItemId)(resources);
                if (project && project.groupClass === GroupClass.FILTER) {
                    readonly = true;
                }

                const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(resourceUuid, readonly));
                if (menuKind && resource) {
                    this.props.dispatch<any>(
                        openContextMenu(event, {
                            name: resource.name,
                            uuid: resource.uuid,
                            ownerUuid: resource.ownerUuid,
                            isTrashed: 'isTrashed' in resource ? resource.isTrashed : false,
                            kind: resource.kind,
                            menuKind,
                            isAdmin,
                            isFrozen: resourceIsFrozen(resource, resources),
                            description: resource.description,
                            storageClassesDesired: (resource as CollectionResource).storageClassesDesired,
                            properties: 'properties' in resource ? resource.properties : {},
                        })
                    );
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            };

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            };

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(toggleOne(uuid))
                this.props.dispatch<any>(deselectAllOthers(uuid))
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            };
        }
    )
);
