module.exports = {
    choose: choose,
    debounce: debounce,
};

// choose('a', {a: 'A', b: 'B'}) --> return 'A'
// choose('a', {a: [console.log, 'foo']}) --> return console.log('foo')
function choose(key, options) {
    var option = options[key];
    if (option instanceof Array && option[0] instanceof Function)
        return option[0].apply(this, option.slice(1));
    else
        return option;
}

// debounce(250, key) --> Return a promise. If someone else
// calls debounce with the same key, reject the promise. If nobody
// else has done so after 250ms, resolve the promise.
function debounce(ms, key) {
    var newpending;
    debounce.pending = debounce.pending || [];
    debounce.pending.map(function(found) {
        if (!newpending && found.key === key) {
            // Promise already pending with this key. Reject the old
            // one, reuse its slot for the new one.
            window.clearTimeout(found.timer);
            found.deferred.reject();
            m.endComputation();
            newpending = found;
        }
    });
    if (!newpending) {
        // No pending promise with this key.
        newpending = {key: key}
        debounce.pending.push(newpending);
    }
    newpending.deferred = m.deferred();
    m.startComputation();
    newpending.timer = window.setTimeout(function() {
        // Success, no more bouncing. Remove from pending list.
        debounce.pending.map(function(found, i) {
            if (found === newpending) {
                debounce.pending.splice(i, 1);
                found.deferred.resolve();
                m.endComputation();
            }
        });
    }, ms);
    return newpending.deferred.promise;
}

// Override mithril's default deferred.onerror, with more error checking
var m = require('mithril');
m.deferred.onerror = function(e) {
    if ({}.toString.call(e) === "[object Error]" &&
        !(e.constructor &&
          e.constructor.toString() &&
          e.constructor.toString().match(/ Error/)))
        throw e;
};
