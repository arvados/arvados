// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { UserResource, User } from '~/models/user';
import { connect, DispatchProp } from 'react-redux';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from '../../services/api/filter-builder';
import { debounce } from 'debounce';
import { ListItemText } from '@material-ui/core';

export interface PeopleSelectProps {

}

export interface PeopleSelectState {
    value: string;
    items: UserResource[];
    suggestions: UserResource[];
}

export const PeopleSelect = connect()(
    class PeopleSelect extends React.Component<PeopleSelectProps & DispatchProp, PeopleSelectState> {

        state: PeopleSelectState = {
            value: '',
            items: [],
            suggestions: []
        };

        render() {
            return (
                <Autocomplete
                    label='Invite people'
                    value={this.state.value}
                    items={this.state.items}
                    suggestions={this.state.suggestions}
                    onChange={this.handleChange}
                    onSelect={this.handleSelect}
                    renderChipValue={this.renderChipValue}
                    renderSuggestion={this.renderSuggestion} />
            );
        }

        renderChipValue({ firstName, lastName }: UserResource) {
            return `${firstName} ${lastName}`;
        }

        renderSuggestion({ firstName, lastName, email }: UserResource) {
            return (
                <ListItemText>
                    {`${firstName} ${lastName} <<${email}>>`}
                </ListItemText>
            );
        }

        handleSelect = (user: UserResource) => {
            const { items } = this.state;
            this.setState({ items: [...items, user], suggestions: [], value: '' });
        }

        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            this.setState({ value: event.target.value }, this.getSuggestions);
        }

        getSuggestions = debounce(() => this.props.dispatch<any>(this.requestSuggestions), 500);

        requestSuggestions = async (_: void, __: void, { userService }: ServiceRepository) => {
            const { value } = this.state;
            const filters = new FilterBuilder()
                .addILike('email', value)
                .getFilters();
            const { items } = await userService.list();
            // const { items } = await userService.list({ filters, limit: 5 });
            this.setState({ suggestions: items });
        }

    });
