// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
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
import { MultiSelectMenuAction, MultiSelectMenuActionSet, MultiSelectMenuActionNames } from "views-components/multiselect-toolbar/ms-menu-actions";
import { ContextMenuAction } from "views-components/context-menu/context-menu-action-set";
import { multiselectActionsFilters, TMultiselectActionsFilters, msResourceKind } from "./ms-toolbar-action-filters";
import { kindToActionSet, findActionByName } from "./ms-kind-action-differentiator";
import { msToggleTrashAction } from "views-components/multiselect-toolbar/ms-project-action-set";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { ContainerRequestResource } from "models/container-request";
import { FavoritesState } from "store/favorites/favorites-reducer";
import { resourceIsFrozen } from "common/frozen-resources";
import { ProjectResource } from "models/project";

type CssRules = "root" | "button";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: "flex",
        flexDirection: "row",
        width: 0,
        padding: 0,
        margin: "1rem auto auto 0.5rem",
        overflowY: 'scroll',
        transition: "width 150ms",
    },
    button: {
        width: "2.5rem",
        height: "2.5rem ",
    },
});

export type MultiselectToolbarProps = {
    checkedList: TCheckedList;
    selectedUuid: string | null
    iconProps: IconProps
    executeMulti: (action: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState) => void;
};

type IconProps = {
    resources: ResourcesState;
    favorites: FavoritesState
}

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        const { classes, checkedList, selectedUuid: singleSelectedUuid, iconProps } = props;
        const singleProjectKind = singleSelectedUuid ? resourceSubKind(singleSelectedUuid, iconProps.resources) : ''
        const currentResourceKinds = singleProjectKind ? singleProjectKind : Array.from(selectedToKindSet(checkedList));

        const currentPathIsTrash = window.location.pathname === "/trash";


        const actions =
            currentPathIsTrash && selectedToKindSet(checkedList).size
                ? [msToggleTrashAction]
                : selectActionsByKind(currentResourceKinds as string[], multiselectActionsFilters)
                .filter((action) => (singleSelectedUuid === null ? action.isForMulti : true));

        return (
            <React.Fragment>
                <Toolbar
                    className={classes.root}
                    style={{ width: `${actions.length * 2.5}rem` }}
                >
                    {actions.length ? (
                        actions.map((action, i) =>
                            action.hasAlts ? (
                                <Tooltip
                                    className={classes.button}
                                    title={currentPathIsTrash || action.useAlts(singleSelectedUuid, iconProps) ? action.altName : action.name}
                                    key={i}
                                    disableFocusListener
                                >
                                    <IconButton onClick={() => props.executeMulti(action, checkedList, iconProps.resources)}>
                                        {currentPathIsTrash || action.useAlts(singleSelectedUuid, iconProps) ? action.altIcon && action.altIcon({}) :  action.icon({})}
                                    </IconButton>
                                </Tooltip>
                            ) : (
                                <Tooltip
                                    className={classes.button}
                                    title={action.name}
                                    key={i}
                                    disableFocusListener
                                >
                                    <IconButton onClick={() => props.executeMulti(action, checkedList, iconProps.resources)}>{action.icon({})}</IconButton>
                                </Tooltip>
                            )
                        )
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

const resourceSubKind = (uuid: string, resources: ResourcesState) => {
    const resource = getResource(uuid)(resources) as ContainerRequestResource | Resource;
    switch (resource.kind) {
        case ResourceKind.PROJECT:
            if(resourceIsFrozen(resource, resources)) return [msResourceKind.PROJECT_FROZEN]
            if((resource as ProjectResource).canWrite === false) return [msResourceKind.PROJECT_READONLY]
            return [msResourceKind.PROJECT]
        default:
            return [resource.kind]
    }
}; 

function selectActionsByKind(currentResourceKinds: Array<string>, filterSet: TMultiselectActionsFilters) {
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

    return filteredResult.sort((a, b) => {
        const nameA = a.name || "";
        const nameB = b.name || "";
        if (nameA < nameB) {
            return -1;
        }
        if (nameA > nameB) {
            return 1;
        }
        return 0;
    });
}

export const isExactlyOneSelected = (checkedList: TCheckedList) => {
    let tally = 0;
    let current = '';
    for (const uuid in checkedList) {
        if (checkedList[uuid] === true) {
            tally++;
            current = uuid;
        }
    }
    return tally === 1 ? current : null
};

//--------------------------------------------------//

function mapStateToProps({multiselect, resources, favorites}: RootState) {
    return {
        checkedList: multiselect.checkedList as TCheckedList,
        selectedUuid: isExactlyOneSelected(multiselect.checkedList),
        iconProps: {
            resources,
            favorites
        }
    }
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        executeMulti: (selectedAction: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState): void => {
            const kindGroups = groupByKind(checkedList, resources);
            switch (selectedAction.name) {
                case MultiSelectMenuActionNames.MOVE_TO:
                case MultiSelectMenuActionNames.REMOVE:
                    const firstResource = getResource(selectedToArray(checkedList)[0])(resources) as ContainerRequestResource | Resource;
                    const action = findActionByName(selectedAction.name as string, kindToActionSet[firstResource.kind]);
                    if (action) action.execute(dispatch, kindGroups[firstResource.kind]);
                    break;
                case MultiSelectMenuActionNames.COPY_TO_CLIPBOARD:
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
