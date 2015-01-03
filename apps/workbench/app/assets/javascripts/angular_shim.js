// Compile any new HTML content that was loaded via jQuery.ajax().
// Currently this only works for tabs, and only because they emit an
// arv:pane:loaded event after updating the DOM.

$(document).on('arv:pane:loaded', function(event, $updatedElement) {
    if (angular && $updatedElement) {
        angular.element($updatedElement).injector().invoke(function($compile) {
            var scope = angular.element($updatedElement).scope();
            $compile($updatedElement)(scope);
        });
    }
});
