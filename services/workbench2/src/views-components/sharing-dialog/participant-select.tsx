// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Autocomplete } from 'components/autocomplete/autocomplete';
import { connect, DispatchProp } from 'react-redux';
import { ServiceRepository } from 'services/services';
import { FilterBuilder } from '../../services/api/filter-builder';
import { debounce } from 'debounce';
import { ListItemText, Typography } from '@material-ui/core';
import { noop } from 'lodash/fp';
import { GroupClass, GroupResource } from 'models/group';
import { getUserDetailsString, getUserDisplayName, UserResource } from 'models/user';
import { Resource, ResourceKind } from 'models/resource';
import { ListResults } from 'services/common-service/common-service';

export interface Participant {
    name: string;
    tooltip: string;
    uuid: string;
}

type ParticipantResource = GroupResource | UserResource;

interface ParticipantSelectProps {
    items: Participant[];
    excludedParticipants?: string[];
    label?: string;
    autofocus?: boolean;
    onlyPeople?: boolean;
    onlyActive?: boolean;
    disabled?: boolean;

    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onCreate?: (person: Participant) => void;
    onDelete?: (index: number) => void;
    onSelect?: (person: Participant) => void;
}

interface ParticipantSelectState {
    value: string;
    suggestions: ParticipantResource[];
}

const getDisplayName = (item: GroupResource | UserResource, detailed: boolean) => {
    switch (item.kind) {
        case ResourceKind.USER:
            return getUserDisplayName(item, detailed, detailed);
        case ResourceKind.GROUP:
            return item.name + `(${`(${(item as Resource).uuid})`})`;
        default:
            return (item as Resource).uuid;
    }
};

const getDisplayTooltip = (item: GroupResource | UserResource) => {
    switch (item.kind) {
        case ResourceKind.USER:
            return getUserDetailsString(item);
        case ResourceKind.GROUP:
            return item.name + `(${`(${(item as Resource).uuid})`})`;
        default:
            return (item as Resource).uuid;
    }
};

export const ParticipantSelect = connect()(
    class ParticipantSelect extends React.Component<ParticipantSelectProps & DispatchProp, ParticipantSelectState> {
        state: ParticipantSelectState = {
            value: '',
            suggestions: []
        };

        render() {
            const { label = 'Add people and groups' } = this.props;

            return (
                <Autocomplete
                    label={label}
                    value={this.state.value}
                    items={this.props.items}
                    suggestions={this.state.suggestions}
                    autofocus={this.props.autofocus}
                    onChange={this.handleChange}
                    onCreate={this.handleCreate}
                    onSelect={this.handleSelect}
                    onDelete={this.props.onDelete && !this.props.disabled ? this.handleDelete : undefined}
                    onFocus={this.props.onFocus}
                    onBlur={this.onBlur}
                    renderChipValue={this.renderChipValue}
                    renderChipTooltip={this.renderChipTooltip}
                    renderSuggestion={this.renderSuggestion}
                    disabled={this.props.disabled} />
            );
        }

        onBlur = (e) => {
            if (this.props.onBlur) {
                this.props.onBlur(e);
            }
            setTimeout(() => this.setState({ value: '', suggestions: [] }), 200);
        }

        renderChipValue(chipValue: Participant) {
            const { name, uuid } = chipValue;
            return name || uuid;
        }

        renderChipTooltip(item: Participant) {
            return item.tooltip;
        }

        renderSuggestion(item: ParticipantResource) {
            return (
                <ListItemText>
                    <Typography noWrap>{getDisplayName(item, true)}</Typography>
                </ListItemText>
            );
        }

        handleDelete = (_: Participant, index: number) => {
            const { onDelete = noop } = this.props;
            onDelete(index);
        }

        handleCreate = () => {
            const { onCreate } = this.props;
            if (onCreate) {
                this.setState({ value: '', suggestions: [] });
                onCreate({
                    name: '',
                    tooltip: '',
                    uuid: this.state.value,
                });
            }
        }

        handleSelect = (selection: ParticipantResource) => {
            const { uuid } = selection;
            const { onSelect = noop } = this.props;
            this.setState({ value: '', suggestions: [] });
            onSelect({
                name: getDisplayName(selection, false),
                tooltip: getDisplayTooltip(selection),
                uuid,
            });
        }

        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            this.setState({ value: event.target.value }, this.getSuggestions);
        }

        getSuggestions = debounce(() => this.props.dispatch<any>(this.requestSuggestions), 500);

        requestSuggestions = async (_: void, __: void, { userService, groupsService }: ServiceRepository) => {
            const { value } = this.state;
            const limit = 5; // FIXME: Does this provide a good UX?

            const filterUsers = new FilterBuilder()
                .addILike('any', value)
                .addEqual('is_active', this.props.onlyActive || undefined)
                .addNotIn('uuid', this.props.excludedParticipants)
                .getFilters();
            const userItems: ListResults<any> = await userService.list({ filters: filterUsers, limit, count: "none" });

            const filterGroups = new FilterBuilder()
                .addNotIn('group_class', [GroupClass.PROJECT, GroupClass.FILTER])
                .addNotIn('uuid', this.props.excludedParticipants)
                .addILike('name', value)
                .getFilters();

            const groupItems: ListResults<any> = await groupsService.list({ filters: filterGroups, limit, count: "none" });
            this.setState({
                suggestions: this.props.onlyPeople
                    ? userItems.items
                    : userItems.items.concat(groupItems.items)
            });
        }
    });
