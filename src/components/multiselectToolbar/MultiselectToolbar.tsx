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
import { extractUuidKind } from 'models/resource';
import { openMoveProcessDialog } from 'store/processes/process-move-actions';
import { openCopyProcessDialog, openCopyManyProcessesDialog } from 'store/processes/process-copy-actions';
import { getResource } from 'store/resources/resources';
import { ResourcesState } from 'store/resources/resources';
import { getProcess } from 'store/processes/process';
import { CopyProcessDialog, CopyManyProcessesDialog } from 'views-components/dialog-forms/copy-process-dialog';
import { collectionActionSet } from 'views-components/context-menu/action-sets/collection-action-set';
import { ContextMenuAction, ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { TrashIcon } from 'components/icon/icon';

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

export type MultiselectToolbarProps = {
    isVisible: boolean;
    checkedList: TCheckedList;
    resources: ResourcesState;
    executeMulti: (fn, checkedList: TCheckedList, resources: ResourcesState) => void;
};

const collectionMSActionsFilter = {
    MAKE_A_COPY: 'Make a copy',
    MOVE_TO: 'Move to',
    TOGGLE_TRASH_ACTION: 'ToggleTrashAction',
};

const multiselectActionsFilters = {
    'arvados#collection': [collectionActionSet, collectionMSActionsFilter],
};

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        const { classes, isVisible, checkedList, resources } = props;
        const currentResourceKinds = Array.from(selectedToKindSet(checkedList));

        const buttons = selectActionsByKind(currentResourceKinds, multiselectActionsFilters);

        return (
            <Toolbar className={classes.root} style={{ width: `${buttons.length * 3.5}rem` }}>
                {buttons.length ? (
                    buttons.map((btn, i) => (
                        <Tooltip title={btn.name} key={i} disableFocusListener>
                            <IconButton onClick={() => props.executeMulti(btn.execute, checkedList, props.resources)}>
                                {btn.icon ? (
                                    btn.icon({ className: 'foo' })
                                ) : btn.name === 'ToggleTrashAction' ? (
                                    <TrashIcon />
                                ) : (
                                    <>error</>
                                )}
                            </IconButton>
                        </Tooltip>
                    ))
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

function filterActions(actionArray: ContextMenuActionSet, filters: Record<string, string>): Array<ContextMenuAction> {
    return actionArray[0].filter((action) => Object.values(filters).includes(action.name as string));
}

function selectActionsByKind(resourceKinds: Array<string>, filterSet: any) {
    const result: Array<ContextMenuAction> = [];
    resourceKinds.forEach((kind) => result.push(...filterActions(filterSet[kind][0], filterSet[kind][1])));
    return result;
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
        executeMulti: (fn, checkedList: TCheckedList, resources: ResourcesState) =>
            selectedToArray(checkedList).forEach((uuid) => {
                console.log(uuid);
                fn(dispatch, getResource(uuid)(resources));
            }),
    };
}
