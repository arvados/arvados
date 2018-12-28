// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Grid, Typography, Button, Card, CardContent, TableBody, TableCell, TableHead, TableRow, Table, Tooltip, IconButton } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { Link } from 'react-router-dom';
import { Dispatch, compose } from 'redux';
import { RootState } from '~/store/store';
import { HelpIcon, AddIcon, MoreOptionsIcon } from '~/components/icon/icon';
import { loadRepositoriesData, openRepositoriesSampleGitDialog, openRepositoryCreateDialog } from '~/store/repositories/repositories-actions';
import { RepositoryResource } from '~/models/repositories';
import { openRepositoryContextMenu } from '~/store/context-menu/context-menu-actions';
import { Routes } from '~/routes/routes';


type CssRules = 'link' | 'button' | 'icon' | 'iconRow' | 'moreOptionsButton' | 'moreOptions' | 'cloneUrls';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        textDecoration: 'none',
        color: theme.palette.primary.main,
        "&:hover": {
            color: theme.palette.primary.dark,
            transition: 'all 0.5s ease'
        }
    },
    button: {
        textAlign: 'right',
        alignSelf: 'center'
    },
    icon: {
        cursor: 'pointer',
        color: theme.palette.grey["500"],
        "&:hover": {
            color: theme.palette.common.black,
            transition: 'all 0.5s ease'
        }
    },
    iconRow: {
        paddingTop: theme.spacing.unit * 2,
        textAlign: 'right'
    },
    moreOptionsButton: {
        padding: 0
    },
    moreOptions: {
        textAlign: 'right',
        '&:last-child': {
            paddingRight: 0
        }
    },
    cloneUrls: {
        whiteSpace: 'pre-wrap'
    }
});

const mapStateToProps = (state: RootState) => {
    return {
        repositories: state.repositories.items
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<RepositoriesActionProps, 'onOptionsMenuOpen' | 'loadRepositories' | 'openRepositoriesSampleGitDialog' | 'openRepositoryCreateDialog'> => ({
    loadRepositories: () => dispatch<any>(loadRepositoriesData()),
    onOptionsMenuOpen: (event, repository) => {
        dispatch<any>(openRepositoryContextMenu(event, repository));
    },
    openRepositoriesSampleGitDialog: () => dispatch<any>(openRepositoriesSampleGitDialog()),
    openRepositoryCreateDialog: () => dispatch<any>(openRepositoryCreateDialog())
});

interface RepositoriesActionProps {
    loadRepositories: () => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, repository: RepositoryResource) => void;
    openRepositoriesSampleGitDialog: () => void;
    openRepositoryCreateDialog: () => void;
}

interface RepositoriesDataProps {
    repositories: RepositoryResource[];
}


type RepositoriesProps = RepositoriesDataProps & RepositoriesActionProps & WithStyles<CssRules>;

export const RepositoriesPanel = compose(
    withStyles(styles),
    connect(mapStateToProps, mapDispatchToProps))(
        class extends React.Component<RepositoriesProps> {
            componentDidMount() {
                this.props.loadRepositories();
            }
            render() {
                const { classes, repositories, onOptionsMenuOpen, openRepositoriesSampleGitDialog, openRepositoryCreateDialog } = this.props;
                return (
                    <Card>
                        <CardContent>
                            <Grid container direction="row">
                                <Grid item xs={8}>
                                    <Typography variant='body1'>
                                        When you are using an Arvados virtual machine, you should clone the https:// URLs. This will authenticate automatically using your API token. <br />
                                        In order to clone git repositories using SSH, <Link to={Routes.SSH_KEYS_USER} className={classes.link}>add an SSH key to your account</Link> and clone the git@ URLs.
                                    </Typography>
                                </Grid>
                                <Grid item xs={4} className={classes.button}>
                                    <Button variant="contained" color="primary" onClick={openRepositoryCreateDialog}>
                                        <AddIcon /> NEW REPOSITORY
                                    </Button>
                                </Grid>
                            </Grid>
                            <Grid item xs={12}>
                                <div className={classes.iconRow}>
                                    <Tooltip title="Sample git quick start">
                                        <IconButton className={classes.moreOptionsButton} onClick={openRepositoriesSampleGitDialog}>
                                            <HelpIcon className={classes.icon} />
                                        </IconButton>
                                    </Tooltip>
                                </div>
                            </Grid>
                            <Grid item xs={12}>
                                {repositories && <Table>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell>Name</TableCell>
                                            <TableCell>URL</TableCell>
                                            <TableCell />
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {repositories.map((repository, index) =>
                                            <TableRow key={index}>
                                                <TableCell>{repository.name}</TableCell>
                                                <TableCell className={classes.cloneUrls}>{repository.cloneUrls.join("\n")}</TableCell>
                                                <TableCell className={classes.moreOptions}>
                                                    <Tooltip title="More options" disableFocusListener>
                                                        <IconButton onClick={event => onOptionsMenuOpen(event, repository)} className={classes.moreOptionsButton}>
                                                            <MoreOptionsIcon />
                                                        </IconButton>
                                                    </Tooltip>
                                                </TableCell>
                                            </TableRow>)}
                                    </TableBody>
                                </Table>}
                            </Grid>
                        </CardContent>
                    </Card>
                );
            }
        }
    );