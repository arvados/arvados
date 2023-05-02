// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from 'react';
import {
    WithStyles,
    withStyles,
    ButtonBase,
    StyleRulesCallback,
    Theme,
    Popover,
    Button,
    Card,
    CardActions,
    Typography,
    CardContent,
    Tooltip,
    IconButton,
} from '@material-ui/core';
import classnames from 'classnames';
import { DefaultTransformOrigin } from 'components/popover/helpers';
import { createTree } from 'models/tree';
import { getNodeDescendants } from 'models/tree';
import debounce from 'lodash/debounce';
import { green, grey } from '@material-ui/core/colors';

export type CssRules = 'root' | 'icon' | 'iconButton' | 'active' | 'checkbox';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        // border: '1px dashed green',
        margin: 0,
        borderRadius: '7px',
        '&:hover': {
            backgroundColor: grey[200],
        },
        '&:focus': {
            color: theme.palette.text.primary,
        },
    },
    active: {
        color: theme.palette.text.primary,
        '& $iconButton': {
            opacity: 1,
        },
    },
    icon: {
        // border: '1px solid red',
        cursor: 'pointer',
        fontSize: 20,
        userSelect: 'none',
        '&:hover': {
            color: theme.palette.text.primary,
        },
    },
    iconButton: {
        color: theme.palette.text.primary,
        opacity: 0.6,
        padding: 1,
        paddingBottom: 5,
    },
    checkbox: {
        width: 24,
        height: 24,
    },
});

enum SelectionMode {
    ALL = 'all',
    NONE = 'none',
}

export interface DataTableMultiselectProps {
    name: string;
    options: string[];
}

interface DataTableFMultiselectPopState {
    anchorEl?: HTMLElement;
}

export const DataTableMultiselectPopover = withStyles(styles)(
    class extends React.Component<DataTableMultiselectProps & WithStyles<CssRules>, DataTableFMultiselectPopState> {
        state: DataTableFMultiselectPopState = {
            anchorEl: undefined,
        };
        icon = React.createRef<HTMLElement>();

        render() {
            const { name, classes, children, options } = this.props;
            return (
                <>
                    <Tooltip disableFocusListener title='Multiselect Actions'>
                        <ButtonBase className={classnames(classes.root)} component='span' onClick={this.open} disableRipple>
                            {children}
                            <IconButton component='span' classes={{ root: classes.iconButton }} tabIndex={-1}>
                                <i className={classnames(['fas fa-sort-down', classes.icon])} data-fa-transform='shrink-3' ref={this.icon} />
                            </IconButton>
                        </ButtonBase>
                    </Tooltip>
                    <Popover
                        anchorEl={this.state.anchorEl}
                        open={!!this.state.anchorEl}
                        anchorOrigin={DefaultTransformOrigin}
                        transformOrigin={DefaultTransformOrigin}
                        onClose={this.close}
                    >
                        <Card>
                            <CardContent>
                                <Typography variant='caption'>{'OPTIONS'}</Typography>
                            </CardContent>
                            {options.length && options.map((option, i) => <div key={i}>{option}</div>)}
                            <CardActions>
                                <Button color='primary' variant='outlined' size='small' onClick={this.close}>
                                    Close
                                </Button>
                            </CardActions>
                        </Card>
                    </Popover>
                    <this.MountHandler />
                </>
            );
        }

        open = () => {
            this.setState({ anchorEl: this.icon.current || undefined });
        };

        submit = debounce(() => {
            // const { onChange } = this.props;
            // if (onChange) {
            //     onChange(this.state.filters);
            // }
        }, 1000);

        MountHandler = () => {
            useEffect(() => {
                return () => {
                    this.submit.cancel();
                };
            }, []);
            return null;
        };

        close = () => {
            this.setState((prev) => ({
                ...prev,
                anchorEl: undefined,
            }));
        };
    }
);
