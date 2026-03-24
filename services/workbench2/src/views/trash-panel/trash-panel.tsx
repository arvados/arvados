// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ResourceKind, TrashableResource } from 'models/resource';
import { ArvadosTheme } from 'common/custom-theme';
import { TrashIcon } from 'components/icon/icon';
import { TRASH_PANEL_ID } from "store/trash-panel/trash-panel-action";
import { openContextMenuAndSelect } from "store/context-menu/context-menu-actions";
import { getResource, ResourcesState } from "store/resources/resources";
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { ContextMenuKind } from 'store/context-menu/context-menu';
import { toggleOne } from 'store/multiselect/multiselect-actions';

type CssRules = "toolbar" | "button" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing(3),
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing(1)
    },
    root: {
        width: '100%',
        boxShadow: "0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)",
    },
});

export interface TrashPanelFilter extends DataTableFilterItem {
    type: ResourceKind;
}

interface TrashPanelDataProps {
    resources: ResourcesState;
}

type TrashPanelProps = TrashPanelDataProps & DispatchProp & WithStyles<CssRules>;

export const TrashPanel = withStyles(styles)(
    connect((state: RootState) => ({
        resources: state.resources
    }))(
        class extends React.Component<TrashPanelProps> {
            render() {
                return <div className={this.props.classes.root}><DataExplorer
                    id={TRASH_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={false}
                    defaultViewIcon={TrashIcon}
                    defaultViewMessages={['Your trash list is empty.']} />
                </div>;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const resource = getResource<TrashableResource>(resourceUuid)(this.props.resources);
                if (resource) {
                    this.props.dispatch<any>(openContextMenuAndSelect(event, {
                        name: '',
                        uuid: resource.uuid,
                        ownerUuid: resource.ownerUuid,
                        isTrashed: resource.isTrashed,
                        kind: resource.kind,
                        menuKind: ContextMenuKind.TRASH
                    }));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(toggleOne(uuid))
            }
        }
    )
);
