// application entry point
window.jQuery = require('jquery')
window.Tether = require('tether')
require('bootstrap')
require('./example.js')
var m = require('mithril')
var Stream = require('mithril/stream')

var ctl = Stream({Tasks: [], Version: 0})

refresh.next = null
refresh.xhr = null
function refresh() {
    const timeout = 60
    if (refresh.xhr !== null) {
        refresh.xhr.abort()
        refresh.xhr = null
    }
    if (refresh.next !== null)
        window.clearTimeout(refresh.next)
    refresh.next = window.setTimeout(refresh, timeout*1000)
    var version = ctl().Version
    m.request({
        method: 'GET',
        url: '/api/tasks/ctl?timeout='+timeout+'&newerThan='+version,
        config: function(xhr) { refresh.xhr = xhr },
    })
        .then(ctl)
        .then(function() {
            if (ctl().Version != version) {
                // Got a new version -- assume the server is obeying
                // newerThan, and start listening for the next version
                // right away.
                refresh()
            } else {
                if (refresh.next !== null)
                    window.clearTimeout(refresh.next)
                refresh.next = window.setTimeout(refresh, 5000)
            }
        })
}

var Home = {
    view: function(vnode) {
        return [
            m('nav.navbar.navbar-toggleable-md.navbar-inverse.bg-primary',
              m('a.navbar-brand[href=#]', 'arvados-boot'),
              m('.collapse.navbar-collapse',
                m('ul.navbar-nav',
                  m('li.nav-item.active',
                    m('a.nav-link[href=/]', {config: m.route}, 'health', m('span.sr-only', '(current)')))))),
            m('.x-spacer', {height: '1em'}),
            m('table.table', {style: {width: '350px'}},
              m('tbody', ctl().Tasks.map(function(task) {
                  return m('tr', [
                      m('td', task.ShortName),
                      m('td',
                        m('span.badge',
                          {class: task.State == 'OK' ? 'badge-success' : 'badge-danger'},
                          task.State)),
                  ])
              }))),
        ]
    }
}

m.route(document.getElementById('app'), '/', {
    '/': Home,
})

refresh()
