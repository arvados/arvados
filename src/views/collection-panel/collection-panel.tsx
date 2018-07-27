// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { 
    StyleRulesCallback, WithStyles, withStyles, Card, CardHeader, IconButton, 
    CardContent, Grid, MenuItem, Menu, ListItemIcon, ListItemText, Typography 
} from '@material-ui/core';
import { connect } from 'react-redux';
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from '../../common/custom-theme';
import { RootState } from '../../store/store';
import { 
    MoreOptionsIcon, CollectionIcon, ShareIcon, RenameIcon, AddFavoriteIcon, MoveToIcon, 
    CopyIcon, ProvenanceGraphIcon, DetailsIcon, AdvancedIcon, RemoveIcon 
} from '../../components/icon/icon';
import { DetailsAttribute } from '../../components/details-attribute/details-attribute';
import { CollectionResource } from '../../models/collection';

type CssRules = 'card' | 'iconHeader';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: '20px'
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.yellow700
    }
});

const MENU_OPTIONS = [
    {
        title: 'Edit collection',
        icon: RenameIcon
    },
    {
        title: 'Share',
        icon: ShareIcon
    },
    {
        title: 'Move to',
        icon: MoveToIcon
    },
    {
        title: 'Add to favorites',
        icon: AddFavoriteIcon
    },
    {
        title: 'Copy to project',
        icon: CopyIcon
    },
    {
        title: 'View details',
        icon: DetailsIcon
    },
    {
        title: 'Provenance graph',
        icon: ProvenanceGraphIcon
    },
    {
        title: 'Advanced',
        icon: AdvancedIcon
    },
    {
        title: 'Remove',
        icon: RemoveIcon
    }
];

interface CollectionPanelDataProps {
    item: CollectionResource;
}

interface CollectionPanelActionProps {
    onItemRouteChange: (collectionId: string) => void;
}

type CollectionPanelProps = CollectionPanelDataProps & CollectionPanelActionProps 
                            & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const CollectionPanel = withStyles(styles)(
    connect((state: RootState) => ({ item: state.collectionPanel.item }))(
        class extends React.Component<CollectionPanelProps> { 

            state = {
                anchorEl: undefined
            };

            showMenu = (event: any) => {
                this.setState({ anchorEl: event.currentTarget });
            }

            closeMenu = () => {
                this.setState({ anchorEl: undefined });
            }

            displayMenuAction = () => {
                return <IconButton
                    aria-label="More options"
                    aria-owns={this.state.anchorEl ? 'submenu' : undefined}
                    aria-haspopup="true"
                    onClick={this.showMenu}>
                    <MoreOptionsIcon />
                </IconButton>;
            }

            render() {
                const { anchorEl } = this.state;
                const { classes, item } = this.props;
                return <div>
                        <Card className={classes.card}>
                            <CardHeader 
                                avatar={ <CollectionIcon className={classes.iconHeader} /> }
                                action={ 
                                    <IconButton
                                        aria-label="More options"
                                        aria-owns={anchorEl ? 'submenu' : undefined}
                                        aria-haspopup="true"
                                        onClick={this.showMenu}>
                                        <MoreOptionsIcon />
                                    </IconButton> 
                                }
                                title={item && item.name } />
                            <CardContent>
                                <Menu
                                    id="submenu"
                                    anchorEl={anchorEl}
                                    open={Boolean(anchorEl)}
                                    onClose={this.closeMenu}>
                                    {MENU_OPTIONS.map((option) => (
                                        <MenuItem key={option.title}>
                                            <ListItemIcon>
                                                <option.icon />
                                            </ListItemIcon>
                                            <ListItemText inset primary={
                                                <Typography variant='body1'>
                                                    {option.title}
                                                </Typography>
                                            }/>
                                        </MenuItem>
                                    ))}
                                </Menu>
                                <Grid container direction="column">
                                    <Grid item xs={6}>
                                    <DetailsAttribute label='Collection UUID' value={item && item.uuid} />
                                        <DetailsAttribute label='Content size' value='54 MB' />
                                    <DetailsAttribute label='Owner' value={item && item.ownerUuid} />
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>

                        <Card className={classes.card}>
                            <CardHeader title="Tags" />
                            <CardContent>
                                <Grid container direction="column">
                                    <Grid item xs={4}>
                                        Tags
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>

                        <Card className={classes.card}>
                            <CardHeader title="Files" />
                            <CardContent>
                                <Grid container direction="column">
                                    <Grid item xs={4}>
                                        Tags
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>
                    </div>;
            }

            componentWillReceiveProps({ match, item, onItemRouteChange }: CollectionPanelProps) {
                if (!item || match.params.id !== item.uuid) {
                    onItemRouteChange(match.params.id);
                }
            }

        }
    )
);