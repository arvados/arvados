// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from 'react';
import withStyles from '@mui/styles/withStyles';
import { Dispatch } from 'redux';
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
import { toggleOne } from 'store/multiselect/multiselect-actions';
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
import { ProjectResource } from 'models/project';
import { projectPanelDataActions, projectPanelRunActions } from 'store/project-panel/project-panel-action-bind';

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

interface ProjectPanelActionProps {
    resetPagination: () => void;
}

type ProjectPanelProps = ProjectPanelDataProps & ProjectPanelActionProps & DispatchProp & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

const mapStateToProps = (state: RootState): ProjectPanelDataProps => {
    const currentItemId = getProjectPanelCurrentUuid(state);
    const resource = getResource<ProjectResource>(currentItemId)(state.resources);
    return {
        currentItemId,
        resources: state.resources,
        isAdmin: state.auth.user!.isAdmin,
        defaultTab: state.auth.user?.prefs.wb?.default_project_tab,
        isRootProject: isUserResource(resource) || currentItemId === state.auth.user?.uuid ,
    };
}

const mapDispatchToProps = (dispatch: Dispatch): ProjectPanelActionProps & DispatchProp => ({
    resetPagination: () => {
        dispatch(projectPanelDataActions.RESET_PAGINATION());
        dispatch(projectPanelRunActions.RESET_PAGINATION());
    },
    dispatch,
});

export const ProjectPanel = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(
    React.memo((props: ProjectPanelProps) => {
        const { classes, isRootProject, currentItemId, resetPagination } = props;

        // Reset all data explorer tab pagination on uuid change
        useEffect(() => {
            resetPagination();
        }, [currentItemId, resetPagination]);

        // Root project doesn't have Overview Panel
        const tabSet = isRootProject ? RootProjectPanelTabLabels : ProjectPanelTabLabels;
        // Default to Data tab if no user preference
        const defaultTab = props.defaultTab || tabSet.DATA;
        // Apply user preference or default to initial state
        const initialPanelState: MPVPanelState[] = Object.keys(tabSet).map(key => ({
                name: tabSet[key],
                visible: tabSet[key] === defaultTab,
        }));

        const handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
            const { resources, isAdmin, currentItemId } = props;
            const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
            // When viewing the contents of a filter group, all contents should be treated as read only.
            let readonly = false;
            const project = currentItemId ? getResource<GroupResource>(currentItemId)(resources) : undefined;
            if (project && project.groupClass === GroupClass.FILTER) {
                readonly = true;
            }

            const menuKind = props.dispatch<any>(resourceToMenuKind(resourceUuid, readonly));
            if (menuKind && resource) {
                props.dispatch<any>(
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
            props.dispatch<any>(loadDetailsPanel(resourceUuid));
        };

        const handleRowDoubleClick = (uuid: string) => {
            props.dispatch<any>(navigateTo(uuid));
        };

        const handleRowClick = (uuid: string) => {
            props.dispatch<any>(toggleOne(uuid))
        };

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
                    data-cy="project-data"
                    className={classes.dataExplorer}>
                    <ProjectPanelData
                        onRowClick={handleRowClick}
                        onRowDoubleClick={handleRowDoubleClick}
                        onContextMenu={handleContextMenu}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs="auto"
                    item
                    data-cy="project-run"
                    className={classes.dataExplorer}>
                    <ProjectPanelRun
                        onRowClick={handleRowClick}
                        onRowDoubleClick={handleRowDoubleClick}
                        onContextMenu={handleContextMenu}
                    />
                </MPVPanelContent>
            </MPVContainer>
        </div>;
    }, preventRerender)
));

function preventRerender(prevProps: ProjectPanelProps, nextProps: ProjectPanelProps) {
    if (!isEqual(prevProps.resources, nextProps.resources)) {
        return false;
    }
    if (prevProps.currentItemId !== nextProps.currentItemId) {
        return false;
    }
    return true;
}
