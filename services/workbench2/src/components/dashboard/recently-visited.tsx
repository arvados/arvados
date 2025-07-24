// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useEffect } from 'react';
import { Dispatch } from 'redux';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { Collapse } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { ArvadosTheme } from 'common/custom-theme';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { ResourcesState, getPopulatedResources } from 'store/resources/resources';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { loadRecentlyVisitedPanel } from 'store/recently-visited/recently-visited-actions';

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
    return {
        recents: state.auth.user?.prefs?.wb?.recentUuids || [],
        resources: state.resources,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<RecentlyVisitedProps, 'loadRecentlyVisitedPanel'> => ({
    loadRecentlyVisitedPanel: () => dispatch<any>(loadRecentlyVisitedPanel()),
});

type RecentlyVisitedProps = {
    recents: string[],
    resources: ResourcesState
    loadRecentlyVisitedPanel: () => void;
};

export const RecentlyVisitedSection = connect(mapStateToProps, mapDispatchToProps)
    (withStyles(styles)(
        ({recents, resources, loadRecentlyVisitedPanel, classes}: RecentlyVisitedProps & WithStyles<CssRules>) => {

            const [items, setItems] = useState<GroupContentsResource[]>([]);
            const [isOpen, setIsOpen] = useState(true);

            useEffect(() => {
                loadRecentlyVisitedPanel();
                // eslint-disable-next-line react-hooks/exhaustive-deps
            }, []);

            useEffect(() => {
                setItems(getPopulatedResources(recents, resources));
            }, [recents, resources]);

            return (
                <div className={classes.root}>
                    <div className={classes.subHeader} onClick={() => setIsOpen(!isOpen)}>
                        <span className={classes.titleBar}>
                            <span>
                                <span>Recently Visited</span>
                                <ExpandChevronRight expanded={isOpen} />
                            </span>
                            {isOpen &&<span className={classes.lastModHead}>last modified</span>}
                        </span>
                        <hr className={classes.hr} />
                    </div>
                    <Collapse in={isOpen}>
                        <ul className={classes.list}>
                            {items.map(item => <RecentlyVisitedItem item={item} classes={classes} />)}
                        </ul>
                    </Collapse>
                </div>
            )
        })
    );

type ItemProps = {
    item: { name: string, uuid: string, modifiedAt: string }
} & WithStyles<CssRules>;


const RecentlyVisitedItem = ({item, classes}: ItemProps) => {
    return (
        <div className={classes.item}>
            <span>
                <ResourceName uuid={item.uuid} />
            </span>
            <span>
                <div className={classes.lastModDate}>{new Date(item.modifiedAt).toLocaleString()}</div>
            </span>
        </div>
    );
}