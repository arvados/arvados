// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar, Tooltip, IconButton } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { TCheckedList } from 'components/data-table/data-table';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { Resource, extractUuidKind } from 'models/resource';
import { getResource } from 'store/resources/resources';
import { ResourcesState } from 'store/resources/resources';
import { ContextMenuAction, ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { RestoreFromTrashIcon, TrashIcon } from 'components/icon/icon';
import { multiselectActionsFilters, TMultiselectActionsFilters } from './ms-toolbar-action-filters';
import { kindToActionSet, findActionByName } from './ms-kind-action-differentiator';
import { toggleTrashAction } from 'views-components/context-menu/action-sets/project-action-set';

type CssRules = 'root' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
        width: 0,
        padding: 0,
        margin: '1rem auto auto 0.5rem',
        overflow: 'hidden',
        transition: 'width 150ms',
        // borderBottom: '1px solid gray',
    },
    button: {
        width: '1rem',
        margin: 'auto 5px',
    },
});

export type MultiselectToolbarProps = {
    isVisible: boolean;
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

        const currentPathIsTrash = window.location.pathname === '/trash';
        const buttons =
            currentPathIsTrash && selectedToKindSet(checkedList).size
                ? [toggleTrashAction]
                : selectActionsByKind(currentResourceKinds, multiselectActionsFilters);

        return (
            <Toolbar className={classes.root} style={{ width: `${buttons.length * 2.12}rem` }}>
                {buttons.length ? (
                    buttons.map((btn, i) =>
                        btn.name === 'ToggleTrashAction' ? (
                            <Tooltip className={classes.button} title={currentPathIsTrash ? 'Restore' : 'Move to trash'} key={i} disableFocusListener>
                                <IconButton onClick={() => props.executeMulti(btn, checkedList, props.resources)}>
                                    {currentPathIsTrash ? <RestoreFromTrashIcon /> : <TrashIcon />}
                                </IconButton>
                            </Tooltip>
                        ) : (
                            <Tooltip className={classes.button} title={btn.name} key={i} disableFocusListener>
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
        );
    })
);

function selectedToArray(checkedList: TCheckedList): Array<string> {
    const arrayifiedSelectedList: Array<string> = [];
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            arrayifiedSelectedList.push(key);
        }
    }
    return arrayifiedSelectedList;
}

function selectedToKindSet(checkedList: TCheckedList): Set<string> {
    const setifiedList = new Set<string>();
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            setifiedList.add(extractUuidKind(key) as string);
        }
    }
    return setifiedList;
}

function filterActions(actionArray: ContextMenuActionSet, filters: Set<string>): Array<ContextMenuAction> {
    return actionArray[0].filter((action) => filters.has(action.name as string));
}

function selectActionsByKind(currentResourceKinds: Array<string>, filterSet: TMultiselectActionsFilters) {
    const rawResult: Set<ContextMenuAction> = new Set();
    const resultNames = new Set();
    const allFiltersArray: ContextMenuAction[][] = [];
    currentResourceKinds.forEach((kind) => {
        if (filterSet[kind]) {
            const actions = filterActions(...filterSet[kind]);
            allFiltersArray.push(actions);
            actions.forEach((action) => {
                if (!resultNames.has(action.name)) {
                    rawResult.add(action);
                    resultNames.add(action.name);
                }
            });
        }
    });

    const filteredNameSet = allFiltersArray.map((filterArray) => {
        const resultSet = new Set();
        filterArray.forEach((action) => resultSet.add(action.name || ''));
        return resultSet;
    });

    const filteredResult = Array.from(rawResult).filter((action) => {
        for (let i = 0; i < filteredNameSet.length; i++) {
            if (!filteredNameSet[i].has(action.name)) return false;
        }
        return true;
    });

    return filteredResult.sort((a, b) => {
        const nameA = a.name || '';
        const nameB = b.name || '';
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
    const { isVisible, checkedList } = state.multiselect;
    return {
        isVisible: isVisible,
        checkedList: checkedList as TCheckedList,
        resources: state.resources,
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        executeMulti: (selectedAction: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState): void => {
            const kindGroups = groupByKind(checkedList, resources);
            for (const kind in kindGroups) {
                const actionSet = kindToActionSet[kind];
                const action = findActionByName(selectedAction.name as string, actionSet);

                if (action) action.execute(dispatch, kindGroups[kind]);
                // if (action && action.name === 'ToggleTrashAction') action.execute(dispatch, kindGroups[kind]);
            }
        },
    };
}

function groupByKind(checkedList: TCheckedList, resources: ResourcesState): Record<string, ContextMenuResource[]> {
    const result = {};
    selectedToArray(checkedList).forEach((uuid) => {
        const resource = getResource(uuid)(resources) as Resource;
        if (!result[resource.kind]) result[resource.kind] = [];
        result[resource.kind].push(resource);
    });
    return result;
}
