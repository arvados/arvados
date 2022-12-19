// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Breadcrumb, Breadcrumbs as BreadcrumbsComponent, BreadcrumbsProps } from 'components/breadcrumbs/breadcrumbs';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { navigateTo } from 'store/navigation/navigation-action';
import { getProperty } from '../../store/properties/properties';
import { BREADCRUMBS } from '../../store/breadcrumbs/breadcrumbs-actions';
import { openSidePanelContextMenu } from 'store/context-menu/context-menu-actions';

type BreadcrumbsDataProps = Pick<BreadcrumbsProps, 'items' | 'resources'>;
type BreadcrumbsActionProps = Pick<BreadcrumbsProps, 'onClick' | 'onContextMenu'>;

const mapStateToProps = () => ({ properties, resources }: RootState): BreadcrumbsDataProps => ({
    items: (getProperty<Breadcrumb[]>(BREADCRUMBS)(properties) || []),
    resources,
});

const mapDispatchToProps = (dispatch: Dispatch): BreadcrumbsActionProps => ({
    onClick: ({ uuid }: Breadcrumb) => {
        dispatch<any>(navigateTo(uuid));
    },
    onContextMenu: (event, breadcrumb: Breadcrumb) => {
        dispatch<any>(openSidePanelContextMenu(event, breadcrumb.uuid));
    }
});

export const Breadcrumbs = connect(mapStateToProps(), mapDispatchToProps)(BreadcrumbsComponent);
