// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import withStyles from '@mui/styles/withStyles';
import { DispatchProp, connect } from 'react-redux';
import { RouteComponentProps } from 'react-router';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { RootState } from 'store/store';
import { ResourcesState, getResource } from 'store/resources/resources';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { openContextMenuAndSelect } from 'store/context-menu/context-menu-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { getProjectPanelCurrentUuid } from "store/project-panel/project-panel";
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
import { isEqual } from 'lodash';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';
import { ProjectPanelTabLabels, RootProjectPanelTabLabels } from 'store/project-panel/project-panel-action';
import { OverviewPanel } from 'components/overview-panel/overview-panel';
import { ProjectAttributes } from './project-attributes';
import { isUserResource } from 'models/user';

type CssRules = 'root' | 'button' | 'mpvRoot' | 'dataExplorer';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
    },
    button: {
        marginLeft: theme.spacing(1),
    },
    mpvRoot: {
        flexGrow: 1,
        display: 'flex',
        flexDirection: 'column',
        flexWrap: 'nowrap',
        minHeight: "500px",
        '& > div': {
            height: '100%',
        },
    },
    dataExplorer: {
        height: "100%",
    },
});

interface ProjectPanelDataProps {
    currentItemId: string | undefined;
    resources: ResourcesState;
    isAdmin: boolean;
    defaultTab?: string;
    isRootProject: boolean;
}

type ProjectPanelProps = ProjectPanelDataProps & DispatchProp & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

const mapStateToProps = (state: RootState): ProjectPanelDataProps => {
    const currentItemId = getProjectPanelCurrentUuid(state);
    const resource = getResource<any>(currentItemId)(state.resources);
    return {
        currentItemId,
        resources: state.resources,
        isAdmin: state.auth.user!.isAdmin,
        defaultTab: state.auth.user?.prefs.wb?.default_project_tab,
        isRootProject: (resource && isUserResource(resource)) || currentItemId === state.auth.user?.uuid ,
    };
}

export const ProjectPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<ProjectPanelProps> {

            shouldComponentUpdate( nextProps: Readonly<ProjectPanelProps>, nextState: Readonly<{}>, nextContext: any ): boolean {
                return !isEqual(nextProps.resources, this.props.resources)
            }

            render() {
                const { classes, isRootProject } = this.props;
                // Root project doesn't have Overview Panel
                const tabSet = isRootProject ? RootProjectPanelTabLabels : ProjectPanelTabLabels;
                // Default to first tab if no user preference
                const defaultTab = this.props.defaultTab || Object.keys(tabSet)[0];
                // Apply user preference or default to initial state
                const initialPanelState: MPVPanelState[] = Object.keys(tabSet).map(key => ({
                        name: tabSet[key],
                        visible: tabSet[key] === defaultTab,
                }));

                return <div data-cy='project-panel' className={classes.root}>
                    <DetailsCardRoot />
                    <MPVContainer
                        className={classes.mpvRoot}
                        panelStates={initialPanelState}
                        justify-content="flex-start"
                        style={{flexWrap: 'nowrap'}}>
                        {isRootProject ? null : <MPVPanelContent
                            forwardProps
                            xs="auto"
                            item
                            data-cy="project-details"
                            className={classes.dataExplorer}>
                            <OverviewPanel detailsElement={<ProjectAttributes />} />
                        </MPVPanelContent>}
                        <MPVPanelContent
                            forwardProps
                            xs="auto"
                            item
                            data-cy="process-data"
                            className={classes.dataExplorer}>
                            <ProjectPanelData
                                onRowClick={this.handleRowClick}
                                onRowDoubleClick={this.handleRowDoubleClick}
                                onContextMenu={this.handleContextMenu}
                            />
                        </MPVPanelContent>
                        <MPVPanelContent
                            forwardProps
                            xs="auto"
                            item
                            data-cy="process-run"
                            className={classes.dataExplorer}>
                            <ProjectPanelRun
                                onRowClick={this.handleRowClick}
                                onRowDoubleClick={this.handleRowDoubleClick}
                                onContextMenu={this.handleContextMenu}
                            />
                        </MPVPanelContent>
                    </MPVContainer>
                </div>
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const { resources, isAdmin, currentItemId } = this.props;
                const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
                // When viewing the contents of a filter group, all contents should be treated as read only.
                let readonly = false;
                const project = currentItemId ? getResource<GroupResource>(currentItemId)(resources) : undefined;
                if (project && project.groupClass === GroupClass.FILTER) {
                    readonly = true;
                }

                const menuKind = this.props.dispatch<any>(resourceToMenuKind(resourceUuid, readonly));
                if (menuKind && resource) {
                    this.props.dispatch<any>(
                        openContextMenuAndSelect(event, {
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
