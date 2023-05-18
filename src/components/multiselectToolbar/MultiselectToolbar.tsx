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

type CssRules = 'root' | 'expanded' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
        justifyContent: 'start',
        width: '0px',
        padding: 0,
        margin: '1rem auto auto 0.5rem',
        overflow: 'hidden',
        transition: 'width 150ms',
        transitionTimingFunction: 'ease',
    },
    expanded: {
        transition: 'width 150ms',
        transitionTimingFunction: 'ease-in',
        width: '40%',
    },
    button: {
        backgroundColor: '#017ead',
        color: 'white',
        fontSize: '0.75rem',
        width: 'fit-content',
        margin: '2px',
        padding: '1px',
    },
});

type MultiselectToolbarAction = {
    name: string;
    action: string;
};

export const defaultActions: Array<MultiselectToolbarAction> = [
    {
        name: 'copy',
        action: 'copySelected',
    },
    {
        name: 'move',
        action: 'moveSelected',
    },
    {
        name: 'remove',
        action: 'removeSelected',
    },
];

export type MultiselectToolbarProps = {
    buttons: Array<MultiselectToolbarAction>;
    isVisible: boolean;
    checkedList: TCheckedList;
    copySelected: () => void;
    moveSelected: () => void;
    removeSelected: (selectedList: TCheckedList) => void;
};

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        // console.log(props);
        const { classes, buttons, isVisible, checkedList } = props;
        return (
            <Toolbar className={isVisible ? `${classes.root} ${classes.expanded}` : classes.root}>
                {buttons.map((btn) => (
                    <Button key={btn.name} className={`${classes.button} ${classes.expanded}`} onClick={() => props[btn.action](checkedList)}>
                        {btn.name}
                    </Button>
                ))}
            </Toolbar>
        );
    })
);

function selectedToString(checkedList: TCheckedList) {
    let stringifiedSelectedList: string = '';
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            stringifiedSelectedList += key + ',';
        }
    }
    return stringifiedSelectedList.slice(0, -1);
}

function selectedToArray<T>(checkedList: TCheckedList): Array<T | string> {
    const arrayifiedSelectedList: Array<T | string> = [];
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            arrayifiedSelectedList.push(key);
        }
    }
    return arrayifiedSelectedList;
}

function mapStateToProps(state: RootState) {
    // console.log(state.resources, state.multiselect.checkedList);
    const { isVisible, checkedList } = state.multiselect;
    return {
        isVisible: isVisible,
        checkedList: checkedList as TCheckedList,
        // selectedList: state.multiselect.checkedList.forEach(processUUID=>containerRequestUUID)
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        copySelected: () => {},
        moveSelected: () => {},
        removeSelected: (checkedList: TCheckedList) => removeMulti(dispatch, checkedList),
    };
}

function removeMulti(dispatch: Dispatch, checkedList: TCheckedList): void {
    const list: Array<string> = selectedToArray(checkedList);
    dispatch<any>(list.length === 1 ? openRemoveProcessDialog(list[0]) : openRemoveManyProcessesDialog(list));
}
