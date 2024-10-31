// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useMemo } from "react";
import { connect } from "react-redux";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Toolbar, IconButton } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from "common/custom-theme";
import { RootState } from "store/store";
import { Dispatch } from "redux";
import { TCheckedList } from "components/data-table/data-table";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { Resource, ResourceKind, extractUuidKind } from "models/resource";
import { getResource } from "store/resources/resources";
import { ResourcesState } from "store/resources/resources";
import { MultiSelectMenuAction, MultiSelectMenuActionSet } from "views-components/multiselect-toolbar/ms-menu-actions";
import { ContextMenuAction, ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { multiselectActionsFilters, TMultiselectActionsFilters } from "./ms-toolbar-action-filters";
import { kindToActionSet, findActionByName } from "./ms-kind-action-differentiator";
import { msToggleTrashAction } from "views-components/multiselect-toolbar/ms-project-action-set";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { ContainerRequestResource } from "models/container-request";
import { FavoritesState } from "store/favorites/favorites-reducer";
import { resourceIsFrozen } from "common/frozen-resources";
import { getResourceWithEditableStatus } from "store/resources/resources";
import { GroupResource } from "models/group";
import { EditableResource } from "models/resource";
import { User } from "models/user";
import { GroupClass } from "models/group";
import { isProcessCancelable } from "store/processes/process";
import { CollectionResource } from "models/collection";
import { getProcess } from "store/processes/process";
import { Process } from "store/processes/process";
import { PublicFavoritesState } from "store/public-favorites/public-favorites-reducer";
import { AuthState } from "store/auth/auth-reducer";
import { IntersectionObserverWrapper } from "./ms-toolbar-overflow-wrapper";
import classNames from "classnames";
import { ContextMenuKind, sortMenuItems, menuDirection } from 'views-components/context-menu/menu-item-sort';

type CssRules = "root" | "iconContainer" | "icon" | "divider";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: "flex",
        flexDirection: "row",
        height: '2.5rem',
        width: 0,
        padding: 0,
        margin: 0,
        overflow: 'hidden',
    },
    iconContainer: {
        height: '100%',
    },
    icon: {
        marginLeft: '-5px',
    },
    divider: {
        marginTop: '5px',
        width: '2rem',
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
    },
});

export type MultiselectToolbarProps = {
    checkedList: TCheckedList;
    selectedResourceUuid: string | null;
    iconProps: IconProps
    user: User | null
    disabledButtons: Set<string>
    auth: AuthState;
    location: string;
    forceMultiSelectMode?: boolean;
    injectedStyles?: string;
    unfreezeRequiresAdmin?: boolean;
    executeMulti: (action: ContextMenuAction | MultiSelectMenuAction, checkedList: TCheckedList, resources: ResourcesState) => void;
};

type IconProps = {
    resources: ResourcesState;
    favorites: FavoritesState;
    publicFavorites: PublicFavoritesState;
}

const detailsCardPaths = [
    '/projects',
]

export const usesDetailsCard = (location: string): boolean => {
    return detailsCardPaths.some(path => location.includes(path))
}

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        const { classes, checkedList, iconProps, user, disabledButtons, location, forceMultiSelectMode, injectedStyles, unfreezeRequiresAdmin } = props;
        const selectedResourceArray = selectedToArray(checkedList);
        const selectedResourceUuid = usesDetailsCard(location) ? props.selectedResourceUuid : selectedResourceArray.length === 1 ? selectedResourceArray[0] : null;
        const singleResourceKind = selectedResourceUuid && !forceMultiSelectMode ? [resourceToMsResourceKind(selectedResourceUuid, iconProps.resources, user, !!unfreezeRequiresAdmin)] : null
        const currentResourceKinds = singleResourceKind ? singleResourceKind : Array.from(selectedToKindSet(checkedList, iconProps.resources));
        const currentPathIsTrash = window.location.pathname === "/trash";

        const rawActions =
            currentPathIsTrash && selectedToKindSet(checkedList).size
                ? [msToggleTrashAction]
                : selectActionsByKind(currentResourceKinds as string[], multiselectActionsFilters).filter((action) =>
                        selectedResourceUuid === null ? action.isForMulti : true
                    );

        const actions: ContextMenuAction[] | MultiSelectMenuAction[] = sortMenuItems(
            singleResourceKind && singleResourceKind.length ? (singleResourceKind[0] as ContextMenuKind) : ContextMenuKind.MULTI,
            rawActions,
            menuDirection.HORIZONTAL
        );

        // eslint-disable-next-line
        const memoizedActions = useMemo(() => actions, [currentResourceKinds, currentPathIsTrash, selectedResourceUuid]);

        const targetResources = selectedResourceUuid ? {[selectedResourceUuid]: true} as TCheckedList : checkedList

        return (
            <React.Fragment>
                <Toolbar
                    className={classNames(classes.root, injectedStyles)}
                    style={{ width: `${(memoizedActions.length * 2.5) + 2}rem`, height: '2.5rem'}}
                    data-cy='multiselect-toolbar'
                    >
                    {memoizedActions.length ? (
                        <IntersectionObserverWrapper 
                            menuLength={memoizedActions.length}
                            key={actions.map(a => a.name).join(',')}
                            >
                            {memoizedActions.map((action, i) =>{
                                const { hasAlts, useAlts, name, altName, icon, altIcon } = action;
                            return action.name === ContextMenuActionNames.DIVIDER ? (
                                action.component && (
                                    <div
                                        className={classes.divider}
                                        data-targetid={`${name}${i}`}
                                        key={i}
                                    >
                                        <action.component />
                                    </div>
                                )
                            ) : hasAlts ? (
                                <span className={classes.iconContainer} data-targetid={name} data-title={(useAlts && useAlts(selectedResourceUuid, iconProps)) ? altName : name}>
                                    <IconButton
                                        data-cy='multiselect-button'
                                        disabled={disabledButtons.has(name)}
                                        onClick={() => props.executeMulti(action, targetResources, iconProps.resources)}
                                        className={classes.icon}
                                        size="large">
                                        {currentPathIsTrash || (useAlts && useAlts(selectedResourceUuid, iconProps)) ? altIcon && altIcon({}) : icon({})}
                                    </IconButton>
                                </span>
                            ) : (
                                //data-targetid is used to determine what goes to the overflow menu
                                //data-title is used to display the tooltip text
                                <span className={classes.iconContainer} data-targetid={name} data-title={name}>
                                    <IconButton
                                        data-cy='multiselect-button'
                                        onClick={() => {
                                            props.executeMulti(action, targetResources, iconProps.resources)}}
                                        className={classes.icon}
                                        size="large">
                                        {action.icon({})}
                                    </IconButton>
                                </span>
                            );
                            })}
                        </IntersectionObserverWrapper>
                    ) : (
                        <></>
                    )}
                </Toolbar>
            </React.Fragment>
        );
    })
);

export function selectedToArray(checkedList: TCheckedList): Array<string> {
    const arrayifiedSelectedList: Array<string> = [];
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            arrayifiedSelectedList.push(key);
        }
    }
    return arrayifiedSelectedList;
}

export function selectedToKindSet(checkedList: TCheckedList, resources: ResourcesState = {}): Set<string> {
    const setifiedList = new Set<string>();
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            isGroupResource(key, resources) ? setifiedList.add(ContextMenuKind.GROUPS) : setifiedList.add(extractUuidKind(key) as string);
        }
    }
    return setifiedList;
}

export const isGroupResource = (uuid: string, resources: ResourcesState): boolean => {
    const resource = getResource(uuid)(resources);
    if(!resource) return false;
    return resource.kind === ResourceKind.PROJECT && (resource as GroupResource).groupClass === GroupClass.ROLE;
};

function groupByKind(checkedList: TCheckedList, resources: ResourcesState): Record<string, ContextMenuResource[]> {
    const result = {};
    selectedToArray(checkedList).forEach(uuid => {
        const resource = getResource(uuid)(resources) as ContainerRequestResource | Resource;
        const kind = isGroupResource(uuid, resources) ? ContextMenuKind.GROUPS : resource.kind;
        if (!result[kind]) result[kind] = [];
        result[kind].push(resource);
    });
    return result;
}

function filterActions(actionArray: MultiSelectMenuActionSet, filters: Set<string>): Array<MultiSelectMenuAction> {
    return actionArray[0].filter(action => filters.has(action.name as string));
}

const resourceToMsResourceKind = (uuid: string, resources: ResourcesState, user: User | null, unfreezeRequiresAdmin: boolean, readonly = false ): (ContextMenuKind | ResourceKind) | undefined => {
    if (!user) return;
    const resource = getResourceWithEditableStatus<GroupResource & EditableResource>(uuid, user.uuid)(resources);
    const { isAdmin } = user;
    const kind = extractUuidKind(uuid);

    const isFrozen = resource?.kind && resource.kind === ResourceKind.PROJECT ? resourceIsFrozen(resource, resources) : false;
    const isEditable = (user.isAdmin || (resource || ({} as EditableResource)).isEditable) && !readonly && !isFrozen;
    const { canManage, canWrite } = resource || {};

    switch (kind) {
        case ResourceKind.PROJECT:
            if (isFrozen) {
                return isAdmin 
                ? ContextMenuKind.FROZEN_PROJECT_ADMIN 
                : canManage && !unfreezeRequiresAdmin
                    ? ContextMenuKind.MANAGEABLE_PROJECT
                    : isEditable 
                        ? ContextMenuKind.FROZEN_PROJECT
                        : ContextMenuKind.READONLY_PROJECT;
            }

            if (isAdmin && !readonly) {
                if (resource?.groupClass === GroupClass.FILTER) {
                    return ContextMenuKind.FILTER_GROUP_ADMIN;
                }
                if (resource?.groupClass === GroupClass.ROLE) {
                    return ContextMenuKind.GROUPS;
                }
                return ContextMenuKind.PROJECT_ADMIN;
            }

            if(canManage === false && canWrite === true) return ContextMenuKind.WRITEABLE_PROJECT;

            return isEditable
                ? resource && resource.groupClass !== GroupClass.FILTER
                    ? ContextMenuKind.PROJECT
                    : ContextMenuKind.FILTER_GROUP
                : ContextMenuKind.READONLY_PROJECT;
        case ResourceKind.COLLECTION:
            const c = getResource<CollectionResource>(uuid)(resources);
            if (c === undefined) {
                return;
            }
            const parent = getResource<GroupResource>(c.ownerUuid)(resources);
            const isWriteable = parent?.canWrite === true && parent.canManage === false;
            const isOldVersion = c.uuid !== c.currentVersionUuid;
            const isTrashed = c.isTrashed;
            return isOldVersion
                ? ContextMenuKind.OLD_VERSION_COLLECTION
                : isTrashed && isEditable
                    ? ContextMenuKind.TRASHED_COLLECTION
                    : isAdmin && isEditable
                        ? ContextMenuKind.COLLECTION_ADMIN
                        : isEditable 
                            ? isWriteable
                                ? ContextMenuKind.WRITEABLE_COLLECTION 
                                : ContextMenuKind.COLLECTION
                            : ContextMenuKind.READONLY_COLLECTION;
        case ResourceKind.PROCESS:
            return !isEditable
                ? ContextMenuKind.READONLY_PROCESS_RESOURCE
                : isAdmin
                    ? resource && isProcessCancelable(getProcess(resource.uuid)(resources) as Process)
                        ? ContextMenuKind.RUNNING_PROCESS_ADMIN
                        : ContextMenuKind.PROCESS_ADMIN
                    : resource && isProcessCancelable(getProcess(resource.uuid)(resources) as Process)
                        ? ContextMenuKind.RUNNING_PROCESS_RESOURCE
                        : ContextMenuKind.PROCESS_RESOURCE;
        case ResourceKind.USER:
            return isAdmin ? ContextMenuKind.ROOT_PROJECT_ADMIN : ContextMenuKind.ROOT_PROJECT;
        case ResourceKind.LINK:
            return ContextMenuKind.LINK;
        case ResourceKind.WORKFLOW:
            return isEditable ? ContextMenuKind.WORKFLOW : ContextMenuKind.READONLY_WORKFLOW;
        default:
            return;
    }
};

function selectActionsByKind(currentResourceKinds: Array<string>, filterSet: TMultiselectActionsFilters): MultiSelectMenuAction[] {
    const rawResult: Set<MultiSelectMenuAction> = new Set();
    const resultNames = new Set();
    const allFiltersArray: MultiSelectMenuAction[][] = []
    currentResourceKinds.forEach(kind => {
        if (filterSet[kind]) {
            const actions = filterActions(...filterSet[kind]);
            allFiltersArray.push(actions);
            actions.forEach(action => {
                if (!resultNames.has(action.name)) {
                    rawResult.add(action);
                    resultNames.add(action.name);
                }
            });
        }
    });

    const filteredNameSet = allFiltersArray.map(filterArray => {
        const resultSet = new Set<string>();
        filterArray.forEach(action => resultSet.add(action.name as string || ""));
        return resultSet;
    });

    const filteredResult = Array.from(rawResult).filter(action => {
        for (let i = 0; i < filteredNameSet.length; i++) {
            if (!filteredNameSet[i].has(action.name as string)) return false;
        }
        return true;
    });

    return filteredResult;
}


//--------------------------------------------------//

function mapStateToProps({auth, multiselect, resources, favorites, publicFavorites, selectedResourceUuid}: RootState) {
    return {
        checkedList: multiselect.checkedList as TCheckedList,
        user: auth && auth.user ? auth.user : null,
        disabledButtons: new Set<string>(multiselect.disabledButtons),
        auth,
        selectedResourceUuid,
        location: window.location.pathname,
        unfreezeRequiresAdmin: auth.remoteHostsConfig[auth.homeCluster]?.clusterConfig?.API?.UnfreezeProjectRequiresAdmin,
        iconProps: {
            resources,
            favorites,
            publicFavorites
        }
    }
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        executeMulti: (selectedAction: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState): void => {
            const kindGroups = groupByKind(checkedList, resources);
            const currentList = selectedToArray(checkedList)
            switch (selectedAction.name) {
                case ContextMenuActionNames.MOVE_TO:
                case ContextMenuActionNames.REMOVE:
                    const firstResourceKind = isGroupResource(currentList[0], resources) 
                        ? ContextMenuKind.GROUPS 
                        : (getResource(currentList[0])(resources) as ContainerRequestResource | Resource).kind;
                    const action = findActionByName(selectedAction.name as string, kindToActionSet[firstResourceKind]);
                    if (action) action.execute(dispatch, kindGroups[firstResourceKind]);
                    break;
                case ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD:
                    const selectedResources = currentList.map(uuid => getResource(uuid)(resources));
                    dispatch<any>(copyToClipboardAction(selectedResources));
                    break;
                default:
                    for (const kind in kindGroups) {
                        const action = findActionByName(selectedAction.name as string, kindToActionSet[kind]);
                        if (action) action.execute(dispatch, kindGroups[kind]);
                    }
                    break;
            }
        },
    };
}
