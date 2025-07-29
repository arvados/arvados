// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { Collapse } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { loadAllProcessesPanel } from 'store/all-processes-panel/all-processes-panel-action';
import { ArvadosTheme } from 'common/custom-theme';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { DashboardItemRow, DashboardColumnNames, DashboardItemRowStyles } from 'components/dashboard/dashboard-item-row';
import { ResourceStatus } from 'views-components/data-explorer/renderers';
import { ResourceKind } from 'models/resource';

type CssRules = 'root' | 'subHeader' | 'titleBar' | 'headers' | 'statusHead' | 'startedAtHead' | 'lastModDate' | 'hr' | 'list' | 'item';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    subHeader: {
        margin: '0 1rem',
        padding: '4px',
    },
    titleBar: {
        display: 'flex',
        justifyContent: 'space-between',
    },
    headers: {
        display: 'flex',
    },
    statusHead: {
        minWidth: '12rem',
        fontSize: '0.875rem',
        marginRight: '2rem',
        textAlign: 'right',
    },
    startedAtHead: {
        minWidth: '12rem',
        fontSize: '0.875rem',
        marginRight: '1rem',
        textAlign: 'right',
    },
    lastModDate: {
        marginLeft: '2rem',
        width: '12rem',
        display: 'flex',
        justifyContent: 'flex-end'
    },
    hr: {
        marginTop: '0',
        marginBottom: '0',
    },
    list: {
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'flex-start',
        width: '100%',
        marginLeft: '-1rem',
    },
    item: {
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

// pass any styles to child elements
const forwardStyles: DashboardItemRowStyles = {
    [DashboardColumnNames.STATUS]: {
        marginRight: '1rem',
        width: '12rem',
        display: 'flex',
        justifyContent: 'flex-end',
    },
    [DashboardColumnNames.STARTED_AT]: {
        width: '12rem',
        display: 'flex',
        justifyContent: 'flex-end',
    },
}

const mapStateToProps = (state: RootState): Pick<RecentWorkflowRunsProps, 'items'> => {
    const selection = (state.dataExplorer.allProcessesPanel?.items || []);
    const recents = selection.map(uuid => state.resources[uuid]).filter(item => item.kind === ResourceKind.PROCESS).slice(0, 5);;
    return {
        items: recents
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<RecentWorkflowRunsProps, 'loadAllProcessesPanel'> => ({
    loadAllProcessesPanel: () => dispatch<any>(loadAllProcessesPanel()),
});

type RecentWorkflowRunsProps = {
    items: any[];
    loadAllProcessesPanel: () => void;
};

export const RecentWorkflowRunsSection = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(({items, loadAllProcessesPanel, classes}: RecentWorkflowRunsProps & WithStyles<CssRules>) => {
        useEffect(() => {
            loadAllProcessesPanel();
        }, [loadAllProcessesPanel]);

        const [isOpen, setIsOpen] = useState(true);

        return (
            <div className={classes.root}>
                <div className={classes.subHeader} onClick={() => setIsOpen(!isOpen)}>
                    <span className={classes.titleBar}>
                        <span>
                            <span>Recent Workflow Runs</span>
                            <ExpandChevronRight expanded={isOpen} />
                        </span>
                        {isOpen &&
                            <span className={classes.headers}>
                                <div className={classes.statusHead}>status</div>
                                <div className={classes.startedAtHead}>started at</div>
                            </span>}
                    </span>
                    <hr className={classes.hr} />
                </div>
                <Collapse in={isOpen}>
                    <ul className={classes.list}>
                        {items.map(item =>
                            <DashboardItemRow
                                item={item}
                                columns={
                                    {
                                        [DashboardColumnNames.NAME]: <ResourceName uuid={item.uuid} />,
                                        [DashboardColumnNames.STATUS]: <ResourceStatus uuid={item.uuid} />,
                                        [DashboardColumnNames.STARTED_AT]: <span>{new Date(item.createdAt).toLocaleString()}</span>,
                                    }
                                }
                                forwardStyles={forwardStyles}
                            />)}
                    </ul>
                </Collapse>
            </div>
        )
}));