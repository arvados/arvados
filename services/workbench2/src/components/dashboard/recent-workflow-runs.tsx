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
import { ArvadosTheme } from 'common/custom-theme';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { DashboardItemRow, DashboardColumnNames, DashboardItemRowStyles } from 'components/dashboard/dashboard-item-row';
import { ResourceStatus } from 'views-components/data-explorer/renderers';
import { loadRecentWorkflows } from 'store/recent-wf-runs/recent-wf-runs-action';
import { ProcessResource } from 'models/process';

type CssRules = 'root' | 'subHeader' | 'titleBar' | 'headers' | 'startedAtHead' | 'lastModDate' | 'hr' | 'list' | 'item';

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
    const selection = (state.dataExplorer.recentWorkflowRuns?.items || []);
    const recents = selection.map(uuid => state.resources[uuid] as ProcessResource).slice(0, 12);;
    return {
        items: recents
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<RecentWorkflowRunsProps, 'loadRecentWorkflows'> => ({
    loadRecentWorkflows: () => dispatch<any>(loadRecentWorkflows()),
});

type RecentWorkflowRunsProps = {
    items: ProcessResource[];
    loadRecentWorkflows: () => void;
};

export const RecentWorkflowRunsSection = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(({items, loadRecentWorkflows, classes}: RecentWorkflowRunsProps & WithStyles<CssRules>) => {
        useEffect(() => {
            loadRecentWorkflows();
        }, [loadRecentWorkflows]);

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
                                <div className={classes.startedAtHead}>started at</div>
                            </span>}
                    </span>
                    <hr className={classes.hr} />
                </div>
                <Collapse in={isOpen}>
                    <ul className={classes.list}>
                        {items.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
                            .map(item =>
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