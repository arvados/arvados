// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useMemo } from "react";
import { connect } from "react-redux";
import { CustomStyleRulesCallback, ArvadosTheme } from 'common/custom-theme';
import { Toolbar, IconButton } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { RootState } from "store/store";
import { Dispatch } from "redux";
import { TCheckedList } from "components/data-table/data-table";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { Resource, extractUuidKind } from "models/resource";
import { getResource, ResourcesState } from "store/resources/resources";
import { MultiSelectMenuAction, MultiSelectMenuActionSet } from "views-components/multiselect-toolbar/ms-menu-actions";
import { ContextMenuAction, ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { multiselectActionsFilters, TMultiselectActionsFilters } from "./ms-toolbar-action-filters";
import { kindToActionSet, findActionByName } from "./ms-kind-action-differentiator";
import { msToggleTrashAction } from "views-components/multiselect-toolbar/ms-project-action-set";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { ContainerRequestResource } from "models/container-request";
import { isUserGroup } from "models/group";
import { AuthState } from "store/auth/auth-reducer";
import { IntersectionObserverWrapper } from "./ms-toolbar-overflow-wrapper";
import classNames from "classnames";
import { ContextMenuKind, sortMenuItems, menuDirection } from 'views-components/context-menu/menu-item-sort';
import { resourceToMenuKind } from "common/resource-to-menu-kind";

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

export type MultiselectToolbarDataProps = {
    checkedList: TCheckedList;
    selectedResourceUuid: string | null;
    resources: ResourcesState;
    disabledButtons: Set<string>
    auth: AuthState;
    location: string;
};

type MultiselectToolbarActionProps = {
    executeComponent: (fn: (dispatch: Dispatch, res: any[]) => void, resources: any[]) => void;
    executeMulti: (action: ContextMenuAction | MultiSelectMenuAction, checkedList: TCheckedList, resources: ResourcesState) => void;
    resourceToMenukind: (uuid: string) => ContextMenuKind | undefined;
};

type MultiselectToolbarRecievedProps = {
    forceMultiSelectMode?: boolean;
    injectedStyles?: string;
}

const detailsCardPaths = [
    '/projects',
]

export const usesDetailsCard = (location: string): boolean => {
    return detailsCardPaths.some(path => location.includes(path))
}

type MultiselectToolbarProps = MultiselectToolbarDataProps & MultiselectToolbarActionProps & MultiselectToolbarRecievedProps & WithStyles<CssRules>;

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps) => {
        const { classes, checkedList, resources, location, forceMultiSelectMode, injectedStyles } = props;
        const selectedResourceArray = selectedToArray(checkedList);
        const selectedResourceUuid = usesDetailsCard(location) ? props.selectedResourceUuid : selectedResourceArray.length === 1 ? selectedResourceArray[0] : null;
        const singleResourceKind = selectedResourceUuid && !forceMultiSelectMode ? [props.resourceToMenukind(selectedResourceUuid)] : null
        const currentResourceKinds = singleResourceKind ? singleResourceKind : Array.from(selectedToKindSet(checkedList, resources));
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

        const fetchedResources = selectedToArray(targetResources).map(uuid => resources[uuid]);

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
                                const { name } = action;
                            return action.name === ContextMenuActionNames.DIVIDER ? (
                                action.component && (
                                    <div
                                        className={classes.divider}
                                        data-targetid={`${name}${i}`}
                                        key={`${name}${i}`}
                                    >
                                        <action.component />
                                    </div>
                                )
                            ) : action.component ? (
                                <span className={classes.iconContainer} key={`${name}${i}`} data-targetid={name} style={{color: 'blue'}}>
                                    <action.component isInToolbar={true} onClick={()=>props.executeComponent(action.execute, fetchedResources)} />
                                </span>
                            ) : (
                                //data-targetid is used to determine what goes to the overflow menu
                                //data-title is used to display the tooltip text
                                <span className={classes.iconContainer} key={`${name}${i}`} data-targetid={name} data-title={name}>
                                    <IconButton
                                        data-cy='multiselect-button'
                                        onClick={() => {
                                            props.executeMulti(action, targetResources, resources)}}
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
            isRoleGroupResource(key, resources) ? setifiedList.add(ContextMenuKind.GROUPS) : setifiedList.add(extractUuidKind(key) as string);
        }
    }
    return setifiedList;
}

export const isRoleGroupResource = (uuid: string, resources: ResourcesState): boolean => {
    const resource = getResource(uuid)(resources);
    return isUserGroup(resource);
};

function groupByKind(checkedList: TCheckedList, resources: ResourcesState): Record<string, ContextMenuResource[]> {
    const result = {};
    selectedToArray(checkedList).forEach(uuid => {
        const resource = getResource(uuid)(resources) as ContainerRequestResource | Resource;
        const kind = isRoleGroupResource(uuid, resources) ? ContextMenuKind.GROUPS : resource.kind;
        if (!result[kind]) result[kind] = [];
        result[kind].push(resource);
    });
    return result;
}

function filterActions(actionArray: MultiSelectMenuActionSet, filters: Set<string>): Array<MultiSelectMenuAction> {
    return actionArray[0].filter(action => filters.has(action.name as string));
}

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

function mapStateToProps({auth, multiselect, resources, selectedResourceUuid}: RootState): MultiselectToolbarDataProps {
    return {
        checkedList: multiselect.checkedList as TCheckedList,
        disabledButtons: new Set<string>(multiselect.disabledButtons),
        auth,
        selectedResourceUuid,
        location: window.location.pathname,
        resources,
    }
}

function mapDispatchToProps(dispatch: Dispatch): MultiselectToolbarActionProps {
    return {
        resourceToMenukind: (uuid: string)=> dispatch<any>(resourceToMenuKind(uuid)),
        executeComponent: (fn: (dispatch: Dispatch, res: any[]) => void, resources: any[]) => fn(dispatch, resources),
        executeMulti: (selectedAction: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState): void => {
            const kindGroups = groupByKind(checkedList, resources);
            const currentList = selectedToArray(checkedList)
            switch (selectedAction.name) {
                case ContextMenuActionNames.MOVE_TO:
                case ContextMenuActionNames.REMOVE:
                    const firstResourceKind = isRoleGroupResource(currentList[0], resources)
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
