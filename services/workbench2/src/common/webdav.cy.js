// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { WebDAV } from "./webdav";

describe('WebDAV', () => {
    it('makes use of provided config', async () => {
        const { open, load, setRequestHeader, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://foo.com/', headers: { Authorization: 'Basic' } }, createRequest);
        const promise = webdav.propfind('foo');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'PROPFIND', 'http://foo.com/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Authorization', 'Basic');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('allows to modify defaults after instantiation', async () => {
        const { open, load, setRequestHeader, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://foo.com/' }, createRequest);
        webdav.setAuthorization('Basic');
        const promise = webdav.propfind('foo');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'PROPFIND', 'http://foo.com/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Authorization', 'Basic');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('PROPFIND', async () => {
        const { open, load, setRequestHeader, createRequest } = mockCreateRequest();
        const webdav = new WebDAV(undefined, createRequest);
        const promise = webdav.propfind('foo');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'PROPFIND', 'foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('PUT', async () => {
        const { open, send, load, progress, setRequestHeader, createRequest } = mockCreateRequest();
        const webdav = new WebDAV(undefined, createRequest);
        const promise = webdav.put('foo', 'Test data');
        progress();
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'PUT', 'foo');
        cy.get('@send').should('have.been.calledWith', 'Test data');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('COPY', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://base' }, createRequest);
        const promise = webdav.copy('foo', 'foo-copy');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'COPY', 'http://base/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Destination', 'http://base/foo-copy');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('COPY - adds baseURL with trailing slash to Destination header', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://base' }, createRequest);
        const promise = webdav.copy('foo', 'foo-copy');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'COPY', 'http://base/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Destination', 'http://base/foo-copy');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('COPY - adds baseURL without trailing slash to Destination header', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://base' }, createRequest);
        const promise = webdav.copy('foo', 'foo-copy');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'COPY', 'http://base/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Destination', 'http://base/foo-copy');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('MOVE', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://base' }, createRequest);
        const promise = webdav.move('foo', 'foo-moved');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'MOVE', 'http://base/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Destination', 'http://base/foo-moved');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('MOVE - adds baseURL with trailing slash to Destination header', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://base' }, createRequest);
        const promise = webdav.move('foo', 'foo-moved');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'MOVE', 'http://base/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Destination', 'http://base/foo-moved');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('MOVE - adds baseURL without trailing slash to Destination header', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = new WebDAV({ baseURL: 'http://base' }, createRequest);
        const promise = webdav.move('foo', 'foo-moved');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'MOVE', 'http://base/foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Destination', 'http://base/foo-moved');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });

    it('DELETE', async () => {
        const { open, load, setRequestHeader, createRequest } = mockCreateRequest();
        const webdav = new WebDAV(undefined, createRequest);
        const promise = webdav.delete('foo');
        load();
        const request = await promise;
        cy.get('@open').should('have.been.calledWith', 'DELETE', 'foo');
        cy.get('@setRequestHeader').should('have.been.calledWith', 'Cache-Control', 'no-cache');
        expect(request).to.be.instanceOf(XMLHttpRequest);
    });
});

const mockCreateRequest = () => {
    const send = cy.stub().as('send');
    const open = cy.stub().as('open');
    const setRequestHeader = cy.stub().as('setRequestHeader');
    const request = new XMLHttpRequest();
    request.send = send;
    request.open = open;
    request.setRequestHeader = setRequestHeader;
    const load = () => request.dispatchEvent(new Event('load'));
    const progress = () => request.dispatchEvent(new Event('progress'));
    return {
        send,
        open,
        load,
        progress,
        setRequestHeader,
        createRequest: () => request
    };
};
