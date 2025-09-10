// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useEffect } from 'react';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { Collapse } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { ResourcesState, getPopulatedResources } from 'store/resources/resources';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { DashboardItemRow, DashboardColumnNames, DashboardItemRowStyles } from 'components/dashboard/dashboard-item-row';
import { RecentUuid } from 'models/user';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { formatDateTime } from 'common/formatters';

type CssRules = 'root' | 'subHeader' | 'titleBar' | 'lastVisHead' | 'hr' | 'list';

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
        cursor: 'pointer',
    },
    lastVisHead: {
        fontSize: '0.875rem',
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
});

// pass any styles to child elements
const forwardStyles: DashboardItemRowStyles = {
    [DashboardColumnNames.LAST_VISITED]: {
        marginLeft: '2rem',
        width: '12rem',
        display: 'flex',
        justifyContent: 'flex-end',
    },
}

const mapStateToProps = (state: RootState) => {
    return {
        recents: state.auth.user?.prefs?.wb?.recentUuids || [],
        resources: state.resources,
    };
};

type RecentlyVisitedProps = {
    recents: RecentUuid[],
    resources: ResourcesState
};

export const RecentlyVisitedSection = connect(mapStateToProps)
    (withStyles(styles)(
        ({recents, resources, classes}: RecentlyVisitedProps & WithStyles<CssRules>) => {

            const [items, setItems] = useState<GroupContentsResource[]>([]);
            const [isOpen, setIsOpen] = useState(true);

            useEffect(() => {
                const recentUuids = recents.map(recent => recent.uuid);
                setItems(getPopulatedResources(recentUuids, resources));
            }, [recents, resources]);

            return (
                <div className={classes.root}>
                    <div className={classes.subHeader} onClick={() => setIsOpen(!isOpen)}>
                        <span className={classes.titleBar}>
                            <span>
                                <span>Recently Visited</span>
                                <ExpandChevronRight expanded={isOpen} />
                            </span>
                            {isOpen &&<span className={classes.lastVisHead}>last visited</span>}
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
                                            [DashboardColumnNames.LAST_VISITED]: <span>{getLastVisitedDate(item.uuid, recents)}</span>,
                                        }
                                    }
                                    forwardStyles={forwardStyles}
                                />
                            )}
                        </ul>
                    </Collapse>
                </div>
            )
        })
    );

const getLastVisitedDate = (targetUuid: string, recents: RecentUuid[]) => {
    const targetRecent = recents.find(recent => recent.uuid === targetUuid);
    if (targetRecent) {
        return formatDateTime(new Date(targetRecent.lastVisited).toISOString());
    }
    return '';
}
