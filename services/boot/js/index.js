// application entry point
window.jQuery = require('jquery')
window.Tether = require('tether')
require('bootstrap')
require('./example.js')
var m = require('mithril')
var Stream = require('mithril/stream')

const refreshInterval = 5

var ctl = Stream({Tasks: [], Version: 0, Outdated: true})

refresh.xhr = null
function refresh() {
    if (refresh.xhr !== null) {
        refresh.xhr.abort()
        refresh.xhr = null
        ctl().Outdated = true
        m.redraw()
    }
    m.request({
        method: 'GET',
        url: '/api/tasks/ctl?timeout='+refreshInterval+'&newerThan='+ctl().version,
        config: function(xhr) { refresh.xhr = xhr },
    })
        .then(function(data) {
            var isNew = data.Version != ctl().Version
            ctl(data)
            refresh.xhr = null
            if (isNew)
                // Got a new version -- assume the server is obeying
                // newerThan, and start listening for the next version
                // right away.
                refresh()
        })
}
window.setInterval(refresh, refreshInterval*1000)
refresh()

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
              m('tbody', {style: {opacity: ctl().Outdated ? .5 : 1}}, ctl().Tasks.map(function(task) {
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
