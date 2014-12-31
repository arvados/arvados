var $ = require('jquery')
, m = require('mithril');

module.exports = md;

function md(cell) {
    var div = $('<div></div>')[0];
    m.render(div, cell);
    return div.children;
}
