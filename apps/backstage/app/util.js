module.exports = {
    choose: choose,
    debounce: debounce,
};

// util.choose('a', {a: 'A', b: 'B'}) --> return 'A'
// util.choose('a', {a: [console.log, 'foo']}) --> return console.log('foo')
function choose(key, options) {
    var option = options[key];
    if (option instanceof Array && option[0] instanceof Function)
        return option[0].apply(this, option.slice(1));
    else
        return option;
}

// util.debounce(250, key) --> Return a promise. If someone else
// calls debounce with the same key, reject the promise. If nobody
// else has done so after 250ms, resolve the promise.
function debounce(ms, key) {
    var newpending;
    util.debounce.pending = util.debounce.pending || [];
    util.debounce.pending.map(function(found) {
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
        util.debounce.pending.push(newpending);
    }
    newpending.deferred = m.deferred();
    m.startComputation();
    newpending.timer = window.setTimeout(function() {
        // Success, no more bouncing. Remove from pending list.
        util.debounce.pending.map(function(found, i) {
            if (found === newpending) {
                util.debounce.pending.splice(i, 1);
                found.deferred.resolve();
                m.endComputation();
            }
        });
    }, ms);
    return newpending.deferred.promise;
}
