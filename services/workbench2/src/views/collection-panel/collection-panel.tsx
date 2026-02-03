// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { CollectionIcon } from 'components/icon/icon';
import { CollectionResource } from 'models/collection';
import { CollectionPanelFiles } from 'views-components/collection-panel-files/collection-panel-files';
import { ResourcesState, getResource } from 'store/resources/resources';
import { GroupResource } from 'models/group';
import { UserResource } from 'models/user';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { resourceIsFrozen } from 'common/frozen-resources';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { collectionPanelActions } from 'store/collection-panel/collection-panel-action';
import { DetailsCardRoot } from 'views-components/details-card/details-card-root';
import { OverviewPanel } from 'components/overview-panel/overview-panel';
import { CollectionAttributes } from './collection-attributes';

type CssRules =
    'root'
    | 'mpvRoot'
    | 'filesCard'
    | 'value'

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
    },
    mpvRoot: {
        width: '100%',
        height: '100%',
    },
    filesCard: {
        padding: 0,
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    },
});

interface CollectionPanelDataProps {
    currentUserUUID: string;
    resources: ResourcesState;
}

type CollectionPanelProps = CollectionPanelDataProps & DispatchProp & WithStyles<CssRules>

type CollectionPanelState = {
    item: CollectionResource | null;
    itemOwner: GroupResource | UserResource | null;
    isWritable: boolean;
    isOldVersion: boolean;
}

export const CollectionPanel = withStyles(styles)(connect(
    (state: RootState) => {
        return {
            currentUserUUID: state.auth.user?.uuid,
            resources: state.resources
        };
    })(
        class extends React.Component<CollectionPanelProps & RouteComponentProps<{ id: string }>> {
            state: CollectionPanelState = {
                item: null,
                itemOwner: null,
                isWritable: false,
                isOldVersion: false,
            }

            shouldComponentUpdate( nextProps: Readonly<CollectionPanelProps & RouteComponentProps<{ id: string }>>, nextState: Readonly<CollectionPanelState>, nextContext: any ): boolean {
                    return this.props.match.params.id !== nextProps.match.params.id
                        || this.props.resources !== nextProps.resources
                        || this.state.isWritable !== nextState.isWritable
            }

            componentDidUpdate( prevProps: Readonly<CollectionPanelProps>, prevState: Readonly<CollectionPanelState>, snapshot?: any ): void {
                const { currentUserUUID, resources } = this.props;
                const collection = getResource<CollectionResource>(this.props.match.params.id)(this.props.resources);
                if (!this.state.item && collection) this.setState({ item: collection });
                if (collection) {
                    this.setState({
                        hasDescription: collection.description && collection.description.length > 0,
                    });
                    const itemOwner = collection ? getResource<GroupResource | UserResource>(collection.ownerUuid)(this.props.resources) : undefined;
                    if (prevState.item !== collection) {
                        this.props.dispatch<any>(setSelectedResourceUuid(collection.uuid))
                        this.setState({
                            item: collection,
                            itemOwner: itemOwner,
                            isOldVersion: collection.currentVersionUuid !== collection.uuid,
                        });
                    }
                    if (prevProps.resources !== resources && itemOwner) {
                        const isWritable = this.checkIsWritable(collection, itemOwner, currentUserUUID, resourceIsFrozen(collection, resources));
                        this.setState({ isWritable: isWritable });
                    }
                }
            }

            componentWillUnmount(): void {
                this.props.dispatch<any>(collectionPanelActions.RESET_COLLECTION_PANEL());
            }

            checkIsWritable = (item: CollectionResource, itemOwner: GroupResource | UserResource | null, currentUserUUID: string, isFrozen: boolean): boolean => {
                let isWritable = false;

                if (item && !this.state.isOldVersion) {
                    if (item.ownerUuid === currentUserUUID) {
                        isWritable = true;
                    } else {
                        if (itemOwner) {
                            isWritable = itemOwner.canWrite;
                        }
                    }
                }
                if (item && isWritable) {
                    isWritable = !isFrozen;
                }
                return isWritable;
            }

            render() {
                const { classes } = this.props;
                const { isWritable, item } = this.state;
                // Set up panels and default tab
                const panelsData: MPVPanelState[] = [
                    { name: "Overview" },
                    { name: "Files", visible: true },
                ];
                return item
                    ? <section className={classes.root}>
                        <DetailsCardRoot />
                        <MPVContainer container className={classes.mpvRoot} justifyContent="flex-start" panelStates={panelsData}>
                            <MPVPanelContent item xs>
                                <OverviewPanel detailsElement={<CollectionAttributes />} />
                            </MPVPanelContent>
                            <MPVPanelContent item xs>
                                <section className={classes.filesCard}>
                                    <CollectionPanelFiles isWritable={isWritable} />
                                </section>
                            </MPVPanelContent>
                        </MPVContainer >
                    </section>
                    : <NotFoundView
                        icon={CollectionIcon}
                        messages={["Collection not found"]}
                    />;
            }
        }
    )
);
