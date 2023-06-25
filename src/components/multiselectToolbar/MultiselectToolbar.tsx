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
import { openRemoveProcessDialog, openRemoveManyProcessesDialog } from 'store/processes/processes-actions';
import { processResourceActionSet } from '../../views-components/context-menu/action-sets/process-resource-action-set';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { Resource, extractUuidKind } from 'models/resource';
import { openMoveProcessDialog } from 'store/processes/process-move-actions';
import { openCopyProcessDialog, openCopyManyProcessesDialog } from 'store/processes/process-copy-actions';
import { getResource } from 'store/resources/resources';
import { ResourcesState } from 'store/resources/resources';
import { getProcess } from 'store/processes/process';
import { CopyProcessDialog, CopyManyProcessesDialog } from 'views-components/dialog-forms/copy-process-dialog';
import { collectionActionSet } from 'views-components/context-menu/action-sets/collection-action-set';
import { ContextMenuAction, ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { TrashIcon } from 'components/icon/icon';
import {
    multiselectActionsFilters,
    TMultiselectActionsFilters,
    contextMenuActionConsts,
} from './ms-toolbar-action-filters';
import { kindToActionSet, findActionByName } from './ms-kind-action-differentiator';

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
        borderBottom: '1px solid gray',
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

        const buttons = selectActionsByKind(currentResourceKinds, multiselectActionsFilters);

        return (
            <Toolbar className={classes.root} style={{ width: `${buttons.length * 2.12}rem` }}>
                {buttons.length ? (
                    buttons.map((btn, i) =>
                        btn.name === 'ToggleTrashAction' ? (
                            <Tooltip className={classes.button} title={'Move to trash'} key={i} disableFocusListener>
                                <IconButton onClick={() => props.executeMulti(btn, checkedList, props.resources)}>
                                    <TrashIcon />
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

//todo: put these all in a /utils
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

//num of currentResourceKinds * num of actions (in ContextMenuActionSet) * num of filters
//worst case: 14 * n * n -oof
function filterActions(actionArray: ContextMenuActionSet, filters: Array<string>): Array<ContextMenuAction> {
    return actionArray[0].filter((action) => filters.includes(action.name as string));
}

//this might be the least efficient algorithm I've ever written
function selectActionsByKind(currentResourceKinds: Array<string>, filterSet: TMultiselectActionsFilters) {
    const result: Array<ContextMenuAction> = [];
    currentResourceKinds.forEach((kind) => {
        if (filterSet[kind]) {
            const actions = filterActions(...filterSet[kind]);
            actions.forEach((action) => {
                if (!result.some((item) => item.name === action.name)) {
                    result.push(action);
                }
            });
        }
    });
    return result.sort((a, b) => {
        a.name = a.name || '';
        b.name = b.name || '';
        if (a.name < b.name) {
            return -1;
        }
        if (a.name > b.name) {
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
        executeMulti: (action: ContextMenuAction, checkedList: TCheckedList, resources: ResourcesState) => {
            selectedToArray(checkedList).forEach((uuid) => {
                const resource = getResource(uuid)(resources);
                executeSpecific(dispatch, action.name, resource);
            });
        },
    };
}

function executeSpecific(dispatch: Dispatch, actionName, resource) {
    const actionSet = kindToActionSet[resource.kind];
    const action = findActionByName(actionName, actionSet);
    if (action) action.execute(dispatch, resource);
}
