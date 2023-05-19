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
import { toggleTrashed } from 'store/trash/trash-actions';
import { ResourceKind, extractUuidKind } from 'models/resource';

type CssRules = 'root' | 'expanded' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
        width: 0,
        padding: 0,
        margin: '1rem auto auto 0.5rem',
        overflow: 'hidden',
    },
    expanded: {
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
    action: string;
    relevantKinds: Set<ResourceKind>;
};

export const defaultActions: Array<MultiselectToolbarAction> = [
    {
        name: 'copy',
        action: 'copySelected',
        relevantKinds: new Set([ResourceKind.COLLECTION]),
    },
    {
        name: 'move',
        action: 'moveSelected',
        relevantKinds: new Set([ResourceKind.COLLECTION, ResourceKind.PROCESS]),
    },
    {
        name: 'remove',
        action: 'removeSelected',
        relevantKinds: new Set([ResourceKind.COLLECTION, ResourceKind.PROCESS, ResourceKind.PROJECT]),
    },
    {
        name: 'foo',
        action: 'barSelected',
        relevantKinds: new Set([ResourceKind.COLLECTION, ResourceKind.PROJECT]),
    },
];

export type MultiselectToolbarProps = {
    actions: Array<MultiselectToolbarAction>;
    isVisible: boolean;
    checkedList: TCheckedList;
    copySelected: () => void;
    moveSelected: () => void;
    barSelected: () => void;
    removeSelected: (selectedList: TCheckedList) => void;
};

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        // console.log(props);
        const { classes, actions, isVisible, checkedList } = props;
        const currentResourceKinds = Array.from(new Set(selectedToArray(checkedList).map((element) => extractUuidKind(element))));
        const buttons = actions.filter((action) => currentResourceKinds.length && currentResourceKinds.every((kind) => action.relevantKinds.has(kind as ResourceKind)));

        return (
            <Toolbar className={isVisible && buttons.length ? `${classes.root} ${classes.expanded}` : classes.root} style={{ width: `${buttons.length * 5.5}rem` }}>
                {buttons.length ? (
                    buttons.map((btn) => (
                        <Button key={btn.name} className={`${classes.button} ${classes.expanded}`} onClick={() => props[btn.action](checkedList)}>
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

function selectedToArray<T>(checkedList: TCheckedList): Array<string> {
    const arrayifiedSelectedList: Array<string> = [];
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            arrayifiedSelectedList.push(key);
        }
    }
    return arrayifiedSelectedList;
}

function mapStateToProps(state: RootState) {
    const { isVisible, checkedList } = state.multiselect;
    return {
        isVisible: isVisible,
        checkedList: checkedList as TCheckedList,
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        copySelected: () => {},
        moveSelected: () => {},
        barSelected: () => {},
        removeSelected: (checkedList: TCheckedList) => removeMulti(dispatch, checkedList),
    };
}

function removeMulti(dispatch: Dispatch, checkedList: TCheckedList): void {
    const list: Array<string> = selectedToArray(checkedList);
    dispatch<any>(list.length === 1 ? openRemoveProcessDialog(list[0]) : openRemoveManyProcessesDialog(list));
}
