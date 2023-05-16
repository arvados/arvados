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
import { openRemoveProcessDialog } from 'store/processes/processes-actions';
import { processResourceActionSet } from '../../views-components/context-menu/action-sets/process-resource-action-set';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { toggleTrashed } from 'store/trash/trash-actions';

type CssRules = 'root' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
    },
    button: {
        color: theme.palette.text.primary,
        // margin: '0.5rem',
    },
});

type MultiselectToolbarAction = {
    name: string;
    fn: string;
};

export const defaultActions: Array<MultiselectToolbarAction> = [
    // {
    //     name: 'copy',
    //     fn: (name, checkedList) => MSToolbarCopyButton(name, checkedList),
    // },
    {
        name: 'remove',
        fn: 'REMOVE',
    },
];

export type MultiselectToolbarProps = {
    buttons: Array<MultiselectToolbarAction>;
    checkedList: TCheckedList;
    copySelected: () => void;
    removeSelected: (selectedList: TCheckedList) => void;
};

export const MultiselectToolbar = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        console.log(props);
        const actions = {
            COPY: props.copySelected,
            REMOVE: props.removeSelected,
        };

        const { classes, buttons, checkedList } = props;
        return (
            <Toolbar className={classes.root}>
                {buttons.map((btn) => (
                    <Button key={btn.name} className={classes.button} onClick={() => actions[btn.fn](checkedList)}>
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
    const arrayifiedSelectedList: Array<string> = [];
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            arrayifiedSelectedList.push(key);
        }
    }
    return arrayifiedSelectedList;
}

function mapStateToProps(state: RootState) {
    // console.log(state.resources, state.multiselect.checkedList);
    return {
        checkedList: state.multiselect.checkedList as TCheckedList,
        // selectedList: state.multiselect.checkedList.forEach(processUUID=>containerRequestUUID)
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        copySelected: () => {},
        removeSelected: (selectedList) => removeMany(dispatch, selectedList),
    };
}

function removeMany(dispatch: Dispatch, checkedList: TCheckedList): void {
    selectedToArray(checkedList).forEach((uuid: string) => dispatch<any>(openRemoveProcessDialog(uuid)));
}
