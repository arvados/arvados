// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect, DispatchProp } from 'react-redux';
import Typography from '@material-ui/core/Typography';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Tooltip } from '@material-ui/core';
import { CopyIcon } from 'components/icon/icon';
import CopyToClipboard from 'react-copy-to-clipboard';
import { ArvadosTheme } from 'common/custom-theme';
import classnames from "classnames";
import { Link } from 'react-router-dom';
import { RootState } from "store/store";
import { FederationConfig, getNavUrl } from "routes/routes";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

type CssRules = 'attribute' | 'label' | 'value' | 'lowercaseValue' | 'link' | 'copyIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    attribute: {
        marginBottom: ".6 rem"
    },
    label: {
        boxSizing: 'border-box',
        color: theme.palette.grey["500"],
        width: '100%'
    },
    value: {
        boxSizing: 'border-box',
        alignItems: 'flex-start'
    },
    lowercaseValue: {
        textTransform: 'lowercase'
    },
    link: {
        color: theme.palette.primary.main,
        textDecoration: 'none',
        overflowWrap: 'break-word',
        cursor: 'pointer'
    },
    copyIcon: {
        marginLeft: theme.spacing.unit,
        color: theme.palette.grey["500"],
        cursor: 'pointer',
        display: 'inline',
        '& svg': {
            fontSize: '1rem'
        }
    }
});

interface DetailsAttributeDataProps {
    label: string;
    classLabel?: string;
    value?: React.ReactNode;
    classValue?: string;
    lowercaseValue?: boolean;
    link?: string;
    children?: React.ReactNode;
    onValueClick?: () => void;
    linkToUuid?: string;
    copyValue?: string;
    uuidEnhancer?: Function;
}

type DetailsAttributeProps = DetailsAttributeDataProps & WithStyles<CssRules> & FederationConfig & DispatchProp;

const mapStateToProps = ({ auth }: RootState): FederationConfig => ({
    localCluster: auth.localCluster,
    remoteHostsConfig: auth.remoteHostsConfig,
    sessions: auth.sessions
});

export const DetailsAttribute = connect(mapStateToProps)(withStyles(styles)(
    class extends React.Component<DetailsAttributeProps> {

        onCopy = (message: string) => {
            this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                message,
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
        }

        render() {
            const { uuidEnhancer, link, value, classes, linkToUuid,
                localCluster, remoteHostsConfig, sessions } = this.props;
            let valueNode: React.ReactNode;

            if (linkToUuid) {
                const uuid = uuidEnhancer ? uuidEnhancer(linkToUuid) : linkToUuid;
                const linkUrl = getNavUrl(linkToUuid || "", { localCluster, remoteHostsConfig, sessions });
                if (linkUrl[0] === '/') {
                    valueNode = <Link to={linkUrl} className={classes.link}>{uuid}</Link>;
                } else {
                    valueNode = <a href={linkUrl} className={classes.link} target='_blank'>{uuid}</a>;
                }
            } else if (link) {
                valueNode = <a href={link} className={classes.link} target='_blank'>{value}</a>;
            } else {
                valueNode = value;
            }

            return <DetailsAttributeComponent {...this.props} value={valueNode} onCopy={this.onCopy} />;
        }
    }
));

interface DetailsAttributeComponentProps {
    value: React.ReactNode;
    onCopy?: (msg: string) => void;
}

export const DetailsAttributeComponent = withStyles(styles)(
    (props: DetailsAttributeDataProps & WithStyles<CssRules> & DetailsAttributeComponentProps) =>
        <Typography component="div" className={props.classes.attribute}>
            <Typography component="div" className={classnames([props.classes.label, props.classLabel])}>{props.label}</Typography>
            <Typography
                onClick={props.onValueClick}
                component="div"
                className={classnames([props.classes.value, props.classValue, { [props.classes.lowercaseValue]: props.lowercaseValue }])}>
                {props.value}
                {props.children}
                {(props.linkToUuid || props.copyValue) && props.onCopy && <Tooltip title="Copy to clipboard">
                    <span className={props.classes.copyIcon}>
                        <CopyToClipboard text={props.linkToUuid || props.copyValue || ""} onCopy={() => props.onCopy!("Copied")}>
                            <CopyIcon />
                        </CopyToClipboard>
                    </span>
                </Tooltip>}
            </Typography>
        </Typography>);

