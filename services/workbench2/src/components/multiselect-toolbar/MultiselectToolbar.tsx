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
// import { kindToActionSet, findActionByName } from "./ms-kind-action-differentiator";
import { msToggleTrashAction } from "views-components/multiselect-toolbar/ms-project-action-set";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { ContainerRequestResource } from "models/container-request";
import { isUserGroup } from "models/group";
import { AuthState } from "store/auth/auth-reducer";
import { IntersectionObserverWrapper } from "./ms-toolbar-overflow-wrapper";
import classNames from "classnames";
import { ContextMenuKind, sortMenuItems, menuDirection } from 'views-components/context-menu/menu-item-sort';
import { resourceToMenuKind } from "common/resource-to-menu-kind";
import { getMenuActionSetByKind } from "common/menu-action-set-actions";
import { intersection } from "lodash";

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
    getAllMenukinds: (checkedList: TCheckedList) => ContextMenuKind[];
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
        const currentResourceKinds = singleResourceKind && !!singleResourceKind[0] ? singleResourceKind : props.getAllMenukinds(checkedList);
        const currentPathIsTrash = window.location.pathname === "/trash";

        const rawActions =
            currentPathIsTrash && selectedToKindSet(checkedList).size
                ? [msToggleTrashAction]
                : selectActionsByKind(currentResourceKinds as ContextMenuKind[]).filter((action) =>
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

function groupByKind(dispatch: Dispatch, checkedList: TCheckedList, resources: ResourcesState): [Record<string, ContextMenuResource[]>, ContextMenuKind | undefined] {
    const result = {};
    let firstResourceKind: ContextMenuKind | undefined;
    selectedToArray(checkedList).forEach((uuid, i) => {
        const menuKind = dispatch<any>(resourceToMenuKind(uuid));
        const resource = getResource(uuid)(resources) as ContainerRequestResource | Resource;
        const kind = isRoleGroupResource(uuid, resources) ? ContextMenuKind.GROUPS : menuKind;
        if (i === 0) {
            firstResourceKind = kind;
        }
        if (!result[kind]) result[kind] = [];
        result[kind].push(resource);
    });
    return [result, firstResourceKind];
}

function selectActionsByKind(currentResourceKinds: ContextMenuKind[]): MultiSelectMenuAction[] {
    if (currentResourceKinds.length === 0) return [];
    const allMenuActionSets = currentResourceKinds.map(kind => getMenuActionSetByKind(kind)).map(actionSetArray => actionSetArray[0]);
    //if only one selected, return all actions
    if (currentResourceKinds.length === 1) return allMenuActionSets[0];
    const actionNames = allMenuActionSets.map(actionSet => actionSet.map(action => action.name));
    const commonNames = new Set(intersection(...actionNames));
    const commonActions = allMenuActionSets
                            .reduce((prev, next) => prev.concat(next), [])
                            .filter(action => commonNames.has(action.name) && action.isForMulti);

    return Array.from(new Set(commonActions));
}

function findActionByName(name: string, actionSet: MultiSelectMenuActionSet) {
    return actionSet[0].find(action => action.name === name);
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
        getAllMenukinds: (checkedList: TCheckedList) => selectedToArray(checkedList).map(uuid => dispatch<any>(resourceToMenuKind(uuid))).filter(kind => !!kind),
        resourceToMenukind: (uuid: string)=> dispatch<any>(resourceToMenuKind(uuid)),
        executeComponent: (fn: (dispatch: Dispatch, res: any[]) => void, resources: any[]) => fn(dispatch, resources),
        executeMulti: (selectedAction: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState): void => {
            const selectedResources = selectedToArray(checkedList).map(uuid => getResource(uuid)(resources)).filter(resource => !!resource);
            const allMenuKinds: ContextMenuKind[] = selectedToArray(checkedList).map(uuid => dispatch<any>(resourceToMenuKind(uuid))).filter(kind => !!kind) as ContextMenuKind[];
            const groupedActionSets = allMenuKinds.reduce((result, menuKind: ContextMenuKind): Record<string, ContextMenuAction[]> => {
                    if (!result[menuKind]) {result[menuKind] = []};
                    result[menuKind].push(findActionByName(selectedAction.name, getMenuActionSetByKind(menuKind)));
                    return result;
                }, {});
            if (selectedAction.name === ContextMenuActionNames.MOVE_TO || selectedAction.name === ContextMenuActionNames.REMOVE) {
                const [kindGroups, firstResourceKind] = groupByKind(dispatch, checkedList, resources);
                if (firstResourceKind) {
                    const action = findActionByName(selectedAction.name, [selectActionsByKind([firstResourceKind])]);
                    if (action) action.execute(dispatch, kindGroups[firstResourceKind]);
                }
            }
            selectedResources.forEach(resource => {
                if (!resource) return;
                const corrsepondingActionSet = groupedActionSets[dispatch<any>(resourceToMenuKind(resource.uuid))!];
                if (!corrsepondingActionSet) return;
                corrsepondingActionSet.forEach(action => action.execute(dispatch, [resource]));
            })
        },
    };
}
