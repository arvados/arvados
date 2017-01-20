// application entry point
window.jQuery = require('jquery')
window.Tether = require('tether')
require('bootstrap')
require('./example.js')
var m = require('mithril')
var Stream = require('mithril/stream')

var checklist = [
    {
        name: 'arvados-boot web gui',
        api: null,
        lastCheck: (new Date()).valueOf(),
        error: Stream(null),
        response: Stream('ok'),
    },
    {
        name: 'arvados-boot web backend',
        api: '/api/ping',
    },
    {
        name: 'arvados-boot fail canary',
        api: '/api/error',
    },
]

checklist.map(function(check) {
    if (!check.api) return
    if (!check.response) check.response = Stream()
    if (!check.error) check.error = Stream()
    m.request({method: 'GET', url: check.api}).then(check.response).catch(check.error)
})

var Home = {
    view: function(vnode) {
        return m('.panel', checklist.map(function(check) {
            return m('div.alert',
                     {class: (!check.response() || check.error()) ? 'alert-danger' : 'alert-success'}, 
                     [
                         check.name,
                         ': ',
                         JSON.stringify(check.response()),
                     ])
        }))
    }
}

m.route(document.getElementById('app'), '/', {
    '/': Home,
})
