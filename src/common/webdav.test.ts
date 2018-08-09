// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { WebDAV } from "./webdav";

describe('WebDAV', () => {
    it('makes use of provided config', async () => {
        const { open, load, setRequestHeader, createRequest } = mockCreateRequest();
        const webdav = WebDAV.create({ baseUrl: 'http://foo.com/', headers: { Authorization: 'Basic' } }, createRequest);
        const promise = webdav.propfind('foo');
        load();
        const request = await promise;
        expect(open).toHaveBeenCalledWith('PROPFIND', 'http://foo.com/foo');
        expect(setRequestHeader).toHaveBeenCalledWith('Authorization', 'Basic');
        expect(request).toBeInstanceOf(XMLHttpRequest);
    });

    it('PROPFIND', async () => {
        const { open, load, createRequest } = mockCreateRequest();
        const webdav = WebDAV.create(undefined, createRequest);
        const promise = webdav.propfind('foo');
        load();
        const request = await promise;
        expect(open).toHaveBeenCalledWith('PROPFIND', 'foo');
        expect(request).toBeInstanceOf(XMLHttpRequest);
    });

    it('PUT', async () => {
        const { open, send, load, createRequest } = mockCreateRequest();
        const webdav = WebDAV.create(undefined, createRequest);
        const promise = webdav.put('foo', { data: 'Test data' });
        load();
        const request = await promise;
        expect(open).toHaveBeenCalledWith('PUT', 'foo');
        expect(send).toHaveBeenCalledWith('Test data');
        expect(request).toBeInstanceOf(XMLHttpRequest);
    });

    it('COPY', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = WebDAV.create(undefined, createRequest);
        const promise = webdav.copy('foo', { destination: 'foo-copy' });
        load();
        const request = await promise;
        expect(open).toHaveBeenCalledWith('COPY', 'foo');
        expect(setRequestHeader).toHaveBeenCalledWith('Destination', 'foo-copy');
        expect(request).toBeInstanceOf(XMLHttpRequest);
    });

    it('MOVE', async () => {
        const { open, setRequestHeader, load, createRequest } = mockCreateRequest();
        const webdav = WebDAV.create(undefined, createRequest);
        const promise = webdav.move('foo', { destination: 'foo-copy' });
        load();
        const request = await promise;
        expect(open).toHaveBeenCalledWith('MOVE', 'foo');
        expect(setRequestHeader).toHaveBeenCalledWith('Destination', 'foo-copy');
        expect(request).toBeInstanceOf(XMLHttpRequest);
    });

    it('DELETE', async () => {
        const { open, load, createRequest } = mockCreateRequest();
        const webdav = WebDAV.create(undefined, createRequest);
        const promise = webdav.delete('foo');
        load();
        const request = await promise;
        expect(open).toHaveBeenCalledWith('DELETE', 'foo');
        expect(request).toBeInstanceOf(XMLHttpRequest);
    });
});

const mockCreateRequest = () => {
    const send = jest.fn();
    const open = jest.fn();
    const setRequestHeader = jest.fn();
    const request = new XMLHttpRequest();
    request.send = send;
    request.open = open;
    request.setRequestHeader = setRequestHeader;
    const load = () => request.dispatchEvent(new Event('load'));
    return {
        send,
        open,
        load,
        setRequestHeader,
        createRequest: () => request
    };
};
