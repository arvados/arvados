// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { CSSProperties } from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { RootState } from 'store/store';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { ArvadosTheme } from 'common/custom-theme';
import { GroupContentsResource } from 'services/groups-service/groups-service';

export const DashboardColumnNames = {
    STATUS: 'Status',
    NAME: 'name',
    MODIFIED_AT: 'modifiedAt',
    LAST_VISITED: 'last visited',
    TYPE: 'type',
}

type CssRules = 'root';

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
});

type DashboardItemRowProps = {
    item: GroupContentsResource;
    columns: Partial<Record<keyof typeof DashboardColumnNames, React.ReactElement<any>>>;
    forwardStyles?: Partial<Record<keyof typeof DashboardColumnNames, CSSProperties>>; //keys must be DashboardColumnNames
};

export const DashboardItemRow = withStyles(styles)(({ item, columns, classes, forwardStyles }: DashboardItemRowProps & WithStyles<CssRules>) => {
    return (
        <div className={classes.root}>
            {Object.entries(columns).map(([key, element]) => (
                <span key={key} style={forwardStyles ? forwardStyles[key] : undefined}>
                    {element}
                </span>
            ))}
        </div>
    );
});
