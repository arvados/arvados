// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
// import { mount, configure } from "enzyme";
import { SearchInput, DEFAULT_SEARCH_DEBOUNCE } from "./search-input";
// import Adapter from 'enzyme-adapter-react-16';

// configure({ adapter: new Adapter() });

describe("<SearchInput />", () => {
    let WrappedComponent;
    let onSearch

    beforeEach(() => {
        cy.clock();
        onSearch = cy.spy().as('onSearch');
        // Wrap the component to test it with props update
        WrappedComponent = ({ selfClearProp = '', textProp }) => {
            const [text, setText] = React.useState(textProp);
            const [selfClear, setSelfClear] = React.useState(selfClearProp);

            window.updateProps = (newClear, newText) => {
                setText(newText);
                setSelfClear(newClear);
            };

            return <SearchInput selfClearProp={selfClear} value={text} onSearch={onSearch} />;
        };
    });

    describe("on submit", () => {
        it("calls onSearch with initial value passed via props", () => {
            cy.mount(<SearchInput selfClearProp="" value="initial value" onSearch={onSearch} />);
            cy.get('form').submit();
            cy.get('@onSearch').should('have.been.calledWith', 'initial value');
        });

        it("calls onSearch with current value", () => {
            cy.mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} />);
            cy.get('input').type('current value');
            cy.get('form').submit();
            cy.get('@onSearch').should('have.been.calledWith', 'current value');
        });

        it("calls onSearch with new value passed via props", () => {
            cy.mount(<WrappedComponent />);
            cy.get('input').type('current value');
            //simulate change of props
            cy.window().then((win) => {
                win.updateProps('', 'new value');
              });
            cy.get('form').submit();
            cy.get('@onSearch').should('have.been.calledWith', 'new value');
        });

        it("cancels timeout set on input value change", () => {
            cy.mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={1000} />);
            cy.get('input').type('current value');
            cy.get('form').submit();
            cy.get('@onSearch').should('have.been.calledOnce');
            cy.tick(1000)
            cy.get('@onSearch').should('have.been.calledOnce');
            cy.get('@onSearch').should('have.been.calledWith', 'current value');
        });

    });

    describe("on input value change", () => {
        it("calls onSearch after default timeout", () => {
            cy.mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} />);
            cy.get('input').type('current value');
            cy.get('@onSearch').should('not.have.been.called');
            cy.tick(DEFAULT_SEARCH_DEBOUNCE);
            cy.get('@onSearch').should('have.been.calledWith', 'current value');
        });

        it("calls onSearch after the time specified in props has passed", () => {
            cy.mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={2000}/>);
            cy.get('input').type('current value');
            cy.tick(1000);
            cy.get('@onSearch').should('not.have.been.called');
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.calledWith', 'current value');
        });

        it("calls onSearch only once after no change happened during the specified time", () => {
            cy.mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={1000}/>);
            cy.get('input').type('current value');
            cy.tick(500);
            cy.get('input').type('current value');
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.calledOnce');
        });

        it("calls onSearch again after the specified time has passed since previous call", () => {
            cy.mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={1000}/>);
            cy.get('input').type('current value');
            cy.tick(500);
            cy.get('input').clear();
            cy.get('input').type('intermediate value');
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.calledWith', 'intermediate value');
            cy.get('input').clear();
            cy.get('input').type('latest value');
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.calledWith', 'latest value');
            cy.get('@onSearch').should('have.been.calledTwice');

        });

    });

    describe("on input target change", () => {
        it("clears the input value on selfClearProp change", () => {
            cy.mount(<WrappedComponent selfClearProp="abc" />);

            // component should clear value upon creation
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.calledWith', '');
            cy.get('@onSearch').should('have.been.calledOnce');

            // component should not clear on same selfClearProp
            cy.window().then((win) => {
                win.updateProps('', 'abc');
              });
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.called');

            // component should clear on selfClearProp change
            cy.window().then((win) => {
                win.updateProps('', '111');
              });
            // cy.get('@onSearch').should('have.been.calledOnce');
            cy.tick(1000);
            cy.get('@onSearch').should('have.been.calledWith', '');
            cy.get('@onSearch').should('have.been.calledTwice');
        });
    });
});
