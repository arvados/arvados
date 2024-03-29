// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from "react";
import { connect } from "react-redux";
import { StyleRulesCallback, withStyles, WithStyles, Toolbar, Tooltip, IconButton } from "@material-ui/core";
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
import { isExactlyOneSelected } from "store/multiselect/multiselect-actions";
import { IntersectionObserverWrapper } from "./ms-toolbar-overflow-wrapper";
import { ContextMenuKind, sortMenuItems } from 'views-components/context-menu/menu-item-sort';
import { sortByProperty } from "common/array-utils";

const WIDTH_TRANSITION = 150

type CssRules = "root" | "transition" | "button" | "iconContainer" | "icon";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: "flex",
        flexDirection: "row",
        width: 0,
        height: '2.7rem',
        padding: 0,
        margin: "1rem auto auto 0.3rem",
        transition: `width ${WIDTH_TRANSITION}ms`,
        overflow: 'hidden',
    },
    transition: {
        display: "flex",
        flexDirection: "row",
        height: '2.7rem',
        padding: 0,
        margin: "1rem auto auto 0.3rem",
        overflow: 'hidden',
        transition: `width ${WIDTH_TRANSITION}ms`,
    },
    button: {
        width: "2.5rem",
        height: "2.5rem ",
        paddingLeft: 0,
        border: "1px solid transparent",
    },
    iconContainer: {
        height: '100%',
    },
    icon: {
        marginLeft: '-0.5rem',
    }
});

export type MultiselectToolbarProps = {
    checkedList: TCheckedList;
    singleSelectedUuid: string | null
    iconProps: IconProps
    user: User | null
    disabledButtons: Set<string>
    executeMulti: (action: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState) => void;
};

type IconProps = {
    resources: ResourcesState;
    favorites: FavoritesState;
    publicFavorites: PublicFavoritesState;
}

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        const { classes, checkedList, singleSelectedUuid, iconProps, user, disabledButtons } = props;
        const singleResourceKind = singleSelectedUuid ? [resourceToMsResourceKind(singleSelectedUuid, iconProps.resources, user)] : null
        console.log(singleResourceKind)
        const currentResourceKinds = singleResourceKind ? singleResourceKind : Array.from(selectedToKindSet(checkedList));
        const currentPathIsTrash = window.location.pathname === "/trash";
        const [isTransitioning, setIsTransitioning] = useState(false);
        
        const handleTransition = () => {
            setIsTransitioning(true)
            setTimeout(() => {
                setIsTransitioning(false)
            }, WIDTH_TRANSITION);
        }
        
        useEffect(()=>{
                handleTransition()
        }, [checkedList])

        const rawActions =
            currentPathIsTrash && selectedToKindSet(checkedList).size
                ? [msToggleTrashAction]
                : selectActionsByKind(currentResourceKinds as string[], multiselectActionsFilters).filter((action) =>
                        singleSelectedUuid === null ? action.isForMulti : true
                    );
                    
        const actions =
            singleResourceKind && singleResourceKind.length ? sortMenuItems(singleResourceKind[0] as ContextMenuKind, rawActions) : rawActions.sort(sortByProperty('name'));

        return (
            <React.Fragment>
                <Toolbar
                    className={isTransitioning ? classes.transition: classes.root}
                    style={{ width: `${(actions.length * 2.5) + 6}rem`}}
                    data-cy='multiselect-toolbar'
                    >
                    {actions.length ? (
                        <IntersectionObserverWrapper menuLength={actions.length}>
                            {actions.map((action, i) =>{
                                const { hasAlts, useAlts, name, altName, icon, altIcon } = action;
                            return hasAlts ? (
                                <Tooltip
                                    className={classes.button}
                                    data-targetid={name}
                                    title={currentPathIsTrash || (useAlts && useAlts(singleSelectedUuid, iconProps)) ? altName : name}
                                    key={i}
                                    disableFocusListener
                                    >
                                    <span className={classes.iconContainer}>
                                        <IconButton
                                            data-cy='multiselect-button'
                                            disabled={disabledButtons.has(name)}
                                            onClick={() => props.executeMulti(action, checkedList, iconProps.resources)}
                                            className={classes.icon}
                                        >
                                            {currentPathIsTrash || (useAlts && useAlts(singleSelectedUuid, iconProps)) ? altIcon && altIcon({}) : icon({})}
                                        </IconButton>
                                    </span>
                                </Tooltip>
                            ) : (
                                <Tooltip
                                    className={classes.button}
                                    data-targetid={name}
                                    title={action.name}
                                    key={i}
                                    disableFocusListener
                                    >
                                    <span className={classes.iconContainer}>
                                        <IconButton
                                            data-cy='multiselect-button'
                                            onClick={() => props.executeMulti(action, checkedList, iconProps.resources)}
                                            className={classes.icon}
                                        >
                                            {action.icon({})}
                                        </IconButton>
                                    </span>
                                </Tooltip>
                            );
                            })}
                        </IntersectionObserverWrapper>
                    ) : (
                        <></>
                    )}
                </Toolbar>
            </React.Fragment>
        )
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

export function selectedToKindSet(checkedList: TCheckedList): Set<string> {
    const setifiedList = new Set<string>();
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            setifiedList.add(extractUuidKind(key) as string);
        }
    }
    return setifiedList;
}

function groupByKind(checkedList: TCheckedList, resources: ResourcesState): Record<string, ContextMenuResource[]> {
    const result = {};
    selectedToArray(checkedList).forEach(uuid => {
        const resource = getResource(uuid)(resources) as ContainerRequestResource | Resource;
        if (!result[resource.kind]) result[resource.kind] = [];
        result[resource.kind].push(resource);
    });
    return result;
}

function filterActions(actionArray: MultiSelectMenuActionSet, filters: Set<string>): Array<MultiSelectMenuAction> {
    return actionArray[0].filter(action => filters.has(action.name as string));
}

const resourceToMsResourceKind = (uuid: string, resources: ResourcesState, user: User | null, readonly = false): (ContextMenuKind | ResourceKind) | undefined => {
    if (!user) return;
    const resource = getResourceWithEditableStatus<GroupResource & EditableResource>(uuid, user.uuid)(resources);
    const { isAdmin } = user;
    const kind = extractUuidKind(uuid);

    const isFrozen = resourceIsFrozen(resource, resources);
    const isEditable = (user.isAdmin || (resource || ({} as EditableResource)).isEditable) && !readonly && !isFrozen;

    switch (kind) {
        case ResourceKind.PROJECT:
            if (isFrozen) {
                return isAdmin ? ContextMenuKind.FROZEN_PROJECT_ADMIN : ContextMenuKind.FROZEN_PROJECT;
            }

            return isAdmin && !readonly
                ? resource && resource.groupClass !== GroupClass.FILTER
                    ? ContextMenuKind.PROJECT_ADMIN
                    : ContextMenuKind.FILTER_GROUP_ADMIN
                : isEditable
                ? resource && resource.groupClass !== GroupClass.FILTER
                    ? ContextMenuKind.PROJECT
                    : ContextMenuKind.FILTER_GROUP
                : ContextMenuKind.READONLY_PROJECT;
        case ResourceKind.COLLECTION:
            const c = getResource<CollectionResource>(uuid)(resources);
            if (c === undefined) {
                return;
            }
            const isOldVersion = c.uuid !== c.currentVersionUuid;
            const isTrashed = c.isTrashed;
            return isOldVersion
                ? ContextMenuKind.OLD_VERSION_COLLECTION
                : isTrashed && isEditable
                ? ContextMenuKind.TRASHED_COLLECTION
                : isAdmin && isEditable
                ? ContextMenuKind.COLLECTION_ADMIN
                : isEditable
                ? ContextMenuKind.COLLECTION
                : ContextMenuKind.READONLY_COLLECTION;
        case ResourceKind.PROCESS:
            return isAdmin && isEditable
                ? resource && isProcessCancelable(getProcess(resource.uuid)(resources) as Process)
                    ? ContextMenuKind.RUNNING_PROCESS_ADMIN
                    : ContextMenuKind.PROCESS_ADMIN
                : readonly
                ? ContextMenuKind.READONLY_PROCESS_RESOURCE
                : resource && isProcessCancelable(getProcess(resource.uuid)(resources) as Process)
                ? ContextMenuKind.RUNNING_PROCESS_RESOURCE
                : ContextMenuKind.PROCESS_RESOURCE;
        case ResourceKind.USER:
            return ContextMenuKind.ROOT_PROJECT;
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

function mapStateToProps({auth, multiselect, resources, favorites, publicFavorites}: RootState) {
    return {
        checkedList: multiselect.checkedList as TCheckedList,
        singleSelectedUuid: isExactlyOneSelected(multiselect.checkedList),
        user: auth && auth.user ? auth.user : null,
        disabledButtons: new Set<string>(multiselect.disabledButtons),
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
            switch (selectedAction.name) {
                case ContextMenuActionNames.MOVE_TO:
                case ContextMenuActionNames.REMOVE:
                    const firstResource = getResource(selectedToArray(checkedList)[0])(resources) as ContainerRequestResource | Resource;
                    const action = findActionByName(selectedAction.name as string, kindToActionSet[firstResource.kind]);
                    if (action) action.execute(dispatch, kindGroups[firstResource.kind]);
                    break;
                case ContextMenuActionNames.COPY_TO_CLIPBOARD:
                    const selectedResources = selectedToArray(checkedList).map(uuid => getResource(uuid)(resources));
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
