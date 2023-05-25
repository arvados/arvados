// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar, Button } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { CopyToClipboardSnackbar } from 'components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar';
import { TCheckedList } from 'components/data-table/data-table';
import { openRemoveProcessDialog, openRemoveManyProcessesDialog } from 'store/processes/processes-actions';
import { processResourceActionSet } from '../../views-components/context-menu/action-sets/process-resource-action-set';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { ResourceKind, extractUuidKind } from 'models/resource';
import { openMoveProcessDialog } from 'store/processes/process-move-actions';
import { openCopyProcessDialog, openCopyManyProcessesDialog } from 'store/processes/process-copy-actions';
import { getResource } from 'store/resources/resources';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { ProcessResource } from 'models/process';
import { ResourcesState } from 'store/resources/resources';
import { Resource } from 'models/resource';
import { getProcess } from 'store/processes/process';
import { CopyProcessDialog, CopyManyProcessesDialog } from 'views-components/dialog-forms/copy-process-dialog';

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
    },
    button: {
        backgroundColor: '#017ead',
        color: 'white',
        fontSize: '0.75rem',
        width: 'auto',
        margin: 'auto',
        padding: '1px',
    },
});

type MultiselectToolbarAction = {
    name: string;
    funcName: string;
    relevantKinds: Set<ResourceKind>;
};

//gleaned from src/views-components/context-menu/action-sets
export const defaultActions: Array<MultiselectToolbarAction> = [
    {
        name: 'copy and re-run',
        funcName: 'copySelected',
        relevantKinds: new Set([ResourceKind.PROCESS]),
    },
    {
        name: 'move',
        funcName: 'moveSelected',
        relevantKinds: new Set([ResourceKind.PROCESS, ResourceKind.PROJECT]),
    },
    {
        name: 'remove',
        funcName: 'removeSelected',
        relevantKinds: new Set([ResourceKind.PROCESS, ResourceKind.COLLECTION]),
    },
    {
        name: 'favorite',
        funcName: 'favoriteSelected',
        relevantKinds: new Set([ResourceKind.PROCESS, ResourceKind.PROJECT, ResourceKind.COLLECTION]),
    },
];

export type MultiselectToolbarProps = {
    actions: Array<MultiselectToolbarAction>;
    isVisible: boolean;
    checkedList: TCheckedList;
    resources: ResourcesState;
    copySelected: (checkedList: TCheckedList, resources: ResourcesState) => void;
    moveSelected: (resource) => void;
    barSelected: () => void;
    removeSelected: (checkedList: TCheckedList) => void;
};

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        // console.log(props);
        const { classes, actions, isVisible, checkedList, resources } = props;

        const currentResourceKinds = Array.from(selectedToKindSet(checkedList));
        const buttons = actions.filter((action) => currentResourceKinds.length && currentResourceKinds.every((kind) => action.relevantKinds.has(kind as ResourceKind)));

        return (
            <Toolbar className={classes.root} style={{ width: `${buttons.length * 5.5}rem` }}>
                {buttons.length ? (
                    buttons.map((btn) => (
                        <Button key={btn.name} className={classes.button} onClick={() => props[btn.funcName](checkedList)}>
                            {/* {<CopyProcessDialog>{btn.name}</CopyProcessDialog>} */}
                            {btn.name}
                        </Button>
                    ))
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
        copySelected: (checkedList: TCheckedList, resources: ResourcesState) => copyMoveMany(dispatch, checkedList),
        moveSelected: (checkedList: TCheckedList) => {},
        barSelected: () => {},
        removeSelected: (checkedList: TCheckedList) => removeMultiProcesses(dispatch, checkedList),
    };
}

function copyMoveMany(dispatch: Dispatch, checkedList: TCheckedList) {
    const selectedList: Array<string> = selectedToArray(checkedList);
    const uuid = selectedList[0];
    // dispatch<any>(openCopyManyProcessesDialog(uuid));
    dispatch<any>(openCopyManyProcessesDialog(selectedList));
}

function moveMultiProcesses(dispatch: Dispatch, checkedList: TCheckedList): void {
    const selectedList: Array<string> = selectedToArray(checkedList);
    // if (selectedList.length === 1) dispatch<any>(openMoveProcessDialog(selectedList[0]));
}

const RemoveFunctions = {
    ONE_PROCESS: (uuid: string) => openRemoveProcessDialog(uuid),
    MANY_PROCESSES: (list: Array<string>) => openRemoveManyProcessesDialog(list),
};

function removeMultiProcesses(dispatch: Dispatch, checkedList: TCheckedList): void {
    const selectedList: Array<string> = selectedToArray(checkedList);
    dispatch<any>(selectedList.length === 1 ? RemoveFunctions.ONE_PROCESS(selectedList[0]) : RemoveFunctions.MANY_PROCESSES(selectedList));
}

//onRemove
//go through the array of selected and choose the appropriate removal function
//have a constant with [ResourceKind]: different removal methods
