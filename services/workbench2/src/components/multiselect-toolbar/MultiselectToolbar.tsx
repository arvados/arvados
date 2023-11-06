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
import { Resource, extractUuidKind } from "models/resource";
import { getResource } from "store/resources/resources";
import { ResourcesState } from "store/resources/resources";
import { ContextMenuAction, ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { RestoreFromTrashIcon, TrashIcon } from "components/icon/icon";
import { multiselectActionsFilters, TMultiselectActionsFilters, contextMenuActionConsts } from "./ms-toolbar-action-filters";
import { kindToActionSet, findActionByName } from "./ms-kind-action-differentiator";
import { msToggleTrashAction } from "views-components/multiselect-toolbar/ms-project-action-set";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { ContainerRequestResource } from "models/container-request";

type CssRules = "root" | "button";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: "flex",
        flexDirection: "row",
        width: 0,
        padding: 0,
        margin: "1rem auto auto 0.5rem",
        overflow: "hidden",
        transition: "width 150ms",
    },
    button: {
        width: "2.5rem",
        height: "2.5rem ",
    },
});

export type MultiselectToolbarProps = {
    checkedList: TCheckedList;
    resources: ResourcesState;
    executeMulti: (action: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState) => void;
};

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        const { classes, checkedList } = props;
        const currentResourceKinds = Array.from(selectedToKindSet(checkedList));

        const currentPathIsTrash = window.location.pathname === "/trash";
        const buttons =
            currentPathIsTrash && selectedToKindSet(checkedList).size
                ? [msToggleTrashAction]
                : selectActionsByKind(currentResourceKinds, multiselectActionsFilters);

        return (
            <React.Fragment>
                <Toolbar
                    className={classes.root}
                    style={{ width: `${buttons.length * 2.5}rem` }}
                >
                    {buttons.length ? (
                        buttons.map((btn, i) =>
                            btn.name === "ToggleTrashAction" ? (
                                <Tooltip
                                    className={classes.button}
                                    title={currentPathIsTrash ? "Restore selected" : "Move to trash"}
                                    key={i}
                                    disableFocusListener
                                >
                                    <IconButton onClick={() => props.executeMulti(btn, checkedList, props.resources)}>
                                        {currentPathIsTrash ? <RestoreFromTrashIcon /> : <TrashIcon />}
                                    </IconButton>
                                </Tooltip>
                            ) : (
                                <Tooltip
                                    className={classes.button}
                                    title={btn.name}
                                    key={i}
                                    disableFocusListener
                                >
                                    <IconButton onClick={() => props.executeMulti(btn, checkedList, props.resources)}>
                                        {btn.icon ? btn.icon({}) : <></>}
                                    </IconButton>
                                </Tooltip>
                            )
                        )
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

function filterActions(actionArray: ContextMenuActionSet, filters: Set<string>): Array<ContextMenuAction> {
    return actionArray[0].filter(action => filters.has(action.name as string));
}

function selectActionsByKind(currentResourceKinds: Array<string>, filterSet: TMultiselectActionsFilters) {
    const rawResult: Set<ContextMenuAction> = new Set();
    const resultNames = new Set();
    const allFiltersArray: ContextMenuAction[][] = [];
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
        const resultSet = new Set();
        filterArray.forEach(action => resultSet.add(action.name || ""));
        return resultSet;
    });

    const filteredResult = Array.from(rawResult).filter(action => {
        for (let i = 0; i < filteredNameSet.length; i++) {
            if (!filteredNameSet[i].has(action.name)) return false;
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

//--------------------------------------------------//

function mapStateToProps(state: RootState) {
    return {
        checkedList: state.multiselect.checkedList as TCheckedList,
        resources: state.resources,
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        executeMulti: (selectedAction: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState): void => {
            const kindGroups = groupByKind(checkedList, resources);
            switch (selectedAction.name) {
                case contextMenuActionConsts.MOVE_TO:
                case contextMenuActionConsts.REMOVE:
                    const firstResource = getResource(selectedToArray(checkedList)[0])(resources) as ContainerRequestResource | Resource;
                    const action = findActionByName(selectedAction.name as string, kindToActionSet[firstResource.kind]);
                    if (action) action.execute(dispatch, kindGroups[firstResource.kind]);
                    break;
                case contextMenuActionConsts.COPY_TO_CLIPBOARD:
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
