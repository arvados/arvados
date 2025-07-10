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
import { ProcessStatus } from 'views-components/data-explorer/renderers';
import { ArvadosTheme } from 'common/custom-theme';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';

type CssRules = 'root' | 'subHeader' | 'titleBar' | 'lastModHead' | 'lastModDate' | 'hr' | 'list' | 'item';

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
    lastModHead: {
        fontSize: '0.875rem',
        marginRight: '1rem',
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
        // Additional styles for better appearance
        boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        '&:hover': {
            background: 'lightgray',
        },
    },
});

const mapStateToProps = (state: RootState) => {
    const selection = (state.dataExplorer.allProcessesPanel?.items || []).slice(0, 5);
    const recents = selection.map(uuid => state.resources[uuid]);
    return {
        items: recents
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    loadAllProcessesPanel: () => dispatch<any>(loadAllProcessesPanel()),
});

type RecentProcessesProps = {
    items: any[];
    loadAllProcessesPanel: () => void;
} & WithStyles<CssRules>;

export const RecentProcessesSection = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(({items, loadAllProcessesPanel, classes}: RecentProcessesProps) => {
    useEffect(() => {
        loadAllProcessesPanel();
    }, [loadAllProcessesPanel]);

    const [isOpen, setIsOpen] = useState(true);

    return (
        <div className={classes.root}>
            <div className={classes.subHeader} onClick={() => setIsOpen(!isOpen)}>
                <span className={classes.titleBar}>
                    <span>
                        <span>Recent Processes</span>
                        <ExpandChevronRight expanded={isOpen} />
                    </span>
                    {isOpen && <span className={classes.lastModHead}>last modified</span>}
                </span>
                <hr className={classes.hr} />
            </div>
            <Collapse in={isOpen}>
                <ul className={classes.list}>
                    {items.map(item => <RecentProcessItem item={item} classes={classes} />)}
                </ul>
            </Collapse>
        </div>
    )
}));

type ItemProps = {
    item: { name: string, uuid: string, modifiedAt: string }
} & WithStyles<CssRules>;


const RecentProcessItem = ({item, classes}: ItemProps) => {
    return (
        <div className={classes.item}>
            <span>
                <ResourceName uuid={item.uuid} />
            </span>
            <span style={{display: 'flex'}}>
                <span><ProcessStatus uuid={item.uuid} /></span>
                <div className={classes.lastModDate}>{new Date(item.modifiedAt).toLocaleString()}</div>
            </span>
        </div>
    );
}