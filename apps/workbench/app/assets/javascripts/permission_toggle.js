$(document).
    on('click', '[data-toggle-permission] input[type=checkbox]', function() {
        var data = {};
        var keys = ['data-permission-uuid',
                    'data-permission-name',
                    'data-permission-head',
                    'data-permission-tail'];
        var attr;
        for(var i in keys) {
            attr = keys[i];
            data[attr] = $(this).closest('[' + attr + ']').attr(attr);
            if (data[attr] === undefined) {
                console.log(["Error: no " + attr + " established here.", this]);
                return;
            }
        }
        var is_checked = $(this).prop('checked');

        if (is_checked) {
            $.ajax('/links',
                   {dataType: 'json',
                    type: 'POST',
                    data: {'link[tail_uuid]': data['data-permission-tail'],
                           'link[head_uuid]': data['data-permission-head'],
                           'link[link_class]': 'permission',
                           'link[name]': data['data-permission-name']},
                    context: this}).
                fail(function(jqxhr, status, error) {
                    $(this).prop('checked', false);
                }).
                done(function(data, status, jqxhr) {
                    $(this).attr('data-permission-uuid', data['uuid']);
                }).
                always(function() {
                    $(this).prop('disabled', false);
                });
        }
        else {
            $.ajax('/links/' + data['data-permission-uuid'],
                   {dataType: 'json',
                    type: 'POST',
                    data: {'_method': 'DELETE'},
                    context: this}).
                fail(function(jqxhr, status, error) {
                    $(this).prop('checked', true);
                }).
                done(function(data, status, jqxhr) {
                    $(this).attr('data-permission-uuid', 'x');
                }).
                always(function() {
                    $(this).prop('disabled', false);
                });
        }
        $(this).prop('disabled', true);
    });
