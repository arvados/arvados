// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { CSSProperties } from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { ArvadosTheme } from 'common/custom-theme';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { openContextMenuOnlyFromUuid } from 'store/context-menu/context-menu-actions';
import { navigateTo } from 'store/navigation/navigation-action';

export const DashboardColumnNames = {
    STATUS: 'status',
    NAME: 'Name',
    MODIFIED_AT: 'last modified',
    LAST_VISITED: 'last visited',
    TYPE: 'type',
    STARTED_AT: 'started at',
}

type CssRules = 'root' | 'columns';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        padding: '8px',
        margin: '4px 0',
        width: '100%',
        background: '#fafafa',
        borderRadius: '8px',
        boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        '&:hover': {
            background: 'lightgray',
        },
    },
    columns: {
        display: 'flex',
    },
});

const mapDispatchToProps = (dispatch: Dispatch): Pick<DashboardItemRowProps, 'navTo' | 'openContextMenu'> => ({
    navTo: (uuid: string) => dispatch<any>(navigateTo(uuid)),
    openContextMenu: (event: React.MouseEvent<HTMLElement>, uuid: string) => dispatch<any>(openContextMenuOnlyFromUuid(event, uuid)),
});

export type DashboardItemRowStyles = Partial<Record<keyof typeof DashboardColumnNames, CSSProperties>>;

type DashboardItemRowProps = {
    item: GroupContentsResource;
    columns: Partial<Record<keyof typeof DashboardColumnNames, React.ReactElement<any>>>;
    forwardStyles?: DashboardItemRowStyles;
    navTo: (uuid: string) => void,
    openContextMenu: (event: React.MouseEvent, uuid: string) => void;
};

export const DashboardItemRow = connect(null, mapDispatchToProps)(
    withStyles(styles)(({ item, columns, classes, forwardStyles, navTo, openContextMenu }: DashboardItemRowProps & WithStyles<CssRules>) => {

        const handleContextMenu = (event: React.MouseEvent) => {
                event.preventDefault();
                event.stopPropagation();
                openContextMenu(event, item.uuid);
            };

        return (
            <div className={classes.root} onContextMenu={handleContextMenu} onClick={() => navTo(item.uuid)} data-cy={'dashboard-item-row'}>
                <span>{columns[DashboardColumnNames.NAME]}</span>
                <span className={classes.columns}>
                    {Object.entries(columns).map(([key, element]) => {
                        if (key === DashboardColumnNames.NAME) return null;
                        return (<span key={key} style={forwardStyles ? forwardStyles[key] : undefined}>
                            {element}
                        </span>
                    )})}
                </span>
            </div>
        );
    })
);
