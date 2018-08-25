// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Breadcrumbs as BreadcrumbsComponent, BreadcrumbsProps } from '~/components/breadcrumbs/breadcrumbs';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { navigateTo } from '~/store/navigation/navigation-action';
import { getProperty } from '../../store/properties/properties';
import { ResourceBreadcrumb, BREADCRUMBS } from '../../store/breadcrumbs/breadcrumbs-actions';



type BreadcrumbsDataProps = Pick<BreadcrumbsProps, 'items'>;
type BreadcrumbsActionProps = Pick<BreadcrumbsProps, 'onClick' | 'onContextMenu'>;

const mapStateToProps = () => ({ properties }: RootState): BreadcrumbsDataProps => ({
    items: getProperty<ResourceBreadcrumb[]>(BREADCRUMBS)(properties) || []
});

const mapDispatchToProps = (dispatch: Dispatch): BreadcrumbsActionProps => ({
    onClick: ({ uuid }: ResourceBreadcrumb) => {
        dispatch<any>(navigateTo(uuid));
    },
    onContextMenu: () => { return; }
});

export const Breadcrumbs = connect(mapStateToProps(), mapDispatchToProps)(BreadcrumbsComponent);