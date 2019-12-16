// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { connect, DispatchProp } from 'react-redux';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from '../../services/api/filter-builder';
import { debounce } from 'debounce';
import { ListItemText, Typography } from '@material-ui/core';
import { noop } from 'lodash/fp';
import { GroupClass } from '~/models/group';

export interface Person {
    name: string;
    email: string;
    uuid: string;
}

export interface PeopleSelectProps {

    items: Person[];
    label?: string;
    autofocus?: boolean;
    onlyPeople?: boolean;

    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onCreate?: (person: Person) => void;
    onDelete?: (index: number) => void;
    onSelect?: (person: Person) => void;

}

export interface PeopleSelectState {
    value: string;
    suggestions: any[];
}

export const PeopleSelect = connect()(
    class PeopleSelect extends React.Component<PeopleSelectProps & DispatchProp, PeopleSelectState> {

        state: PeopleSelectState = {
            value: '',
            suggestions: []
        };

        render() {
            const { label = 'Share' } = this.props;

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
                    onDelete={this.handleDelete}
                    onFocus={this.props.onFocus}
                    onBlur={this.props.onBlur}
                    renderChipValue={this.renderChipValue}
                    renderSuggestion={this.renderSuggestion} />
            );
        }

        renderChipValue({ name, uuid }: Person) {
            return name ? name : uuid;
        }

        renderSuggestion({ firstName, lastName, email, name }: any) {
            return (
                <ListItemText>
                    {name ?
                        <Typography noWrap>{name}</Typography> :
                        <Typography noWrap>{`${firstName} ${lastName} <<${email}>>`}</Typography>}
                </ListItemText>
            );
        }

        handleDelete = (_: Person, index: number) => {
            const { onDelete = noop } = this.props;
            onDelete(index);
        }

        handleCreate = () => {
            const { onCreate } = this.props;
            if (onCreate) {
                this.setState({ value: '', suggestions: [] });
                onCreate({
                    email: '',
                    name: '',
                    uuid: this.state.value,
                });
            }
        }

        handleSelect = ({ email, firstName, lastName, uuid, name }: any) => {
            const { onSelect = noop } = this.props;
            this.setState({ value: '', suggestions: [] });
            onSelect({
                email,
                name: `${name ? name : `${firstName} ${lastName}`}`,
                uuid,
            });
        }

        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            this.setState({ value: event.target.value }, this.getSuggestions);
        }

        getSuggestions = debounce(() => this.props.dispatch<any>(this.requestSuggestions), 500);

        requestSuggestions = async (_: void, __: void, { userService, groupsService }: ServiceRepository) => {
            const { value } = this.state;
            const filterGroups = new FilterBuilder()
                .addNotIn('group_class', [GroupClass.PROJECT])
                .addILike('name', value)
                .getFilters();
            const groupItems = await groupsService.list({ filters: filterGroups, limit: 5 });
            const filterUsers = new FilterBuilder()
                .addILike('email', value)
                .getFilters();
            const userItems: any = await userService.list({ filters: filterUsers, limit: 5 });
            const items = groupItems.items.concat(userItems.items);
            this.setState({ suggestions: this.props.onlyPeople ? userItems.items : items });
        }
    });
